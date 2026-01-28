/*
Copyright 2022 DigitalOcean.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	"github.com/google/go-cmp/cmp"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var databaseuserlog = logf.Log.WithName("databaseuser-resource")

func SetupDatabaseUserWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)
	initGlobalK8sClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.DatabaseUser{}).
		WithValidator(&DatabaseUserValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseuser,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseusers,verbs=create;update,versions=v1alpha1,name=vdatabaseuser.kb.io,admissionReviewVersions=v1
type DatabaseUserValidator struct{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserValidator) ValidateCreate(ctx context.Context, user *v1alpha1.DatabaseUser) (warnings admission.Warnings, err error) {
	databaseuserlog.Info("validate create", "name", user.Name)

	clusterPath := field.NewPath("spec").Child("cluster")

	clusterAPIGroup := pointer.StringDeref(user.Spec.Cluster.APIGroup, "")
	if clusterAPIGroup != v1alpha1.GroupVersion.Group {
		return warnings, field.Invalid(clusterPath.Child("apiGroup"), clusterAPIGroup, "apiGroup must be "+v1alpha1.GroupVersion.Group)
	}

	var (
		clusterNN = types.NamespacedName{
			Namespace: user.Namespace,
			Name:      user.Spec.Cluster.Name,
		}
		clusterKind = user.Spec.Cluster.Kind
		clusterUUID string
	)

	switch strings.ToLower(clusterKind) {
	case strings.ToLower(v1alpha1.DatabaseClusterKind):
		var cluster v1alpha1.DatabaseCluster
		if err := webhookClient.Get(ctx, clusterNN, &cluster); err != nil {
			if kerrors.IsNotFound(err) {
				return warnings, field.NotFound(clusterPath, clusterNN)
			}
			return warnings, fmt.Errorf("failed to fetch DatabaseCluster %s: %s", clusterNN, err)
		}
		clusterUUID = cluster.Status.UUID
	case strings.ToLower(v1alpha1.DatabaseClusterReferenceKind):
		var clusterRef v1alpha1.DatabaseClusterReference
		if err := webhookClient.Get(ctx, clusterNN, &clusterRef); err != nil {
			if kerrors.IsNotFound(err) {
				return warnings, field.NotFound(clusterPath, clusterNN)
			}
			return warnings, fmt.Errorf("failed to fetch DatabaseClusterReference %s: %s", clusterNN, err)
		}
		clusterUUID = clusterRef.Spec.UUID
	default:
		return warnings, field.TypeInvalid(
			clusterPath.Child("kind"),
			clusterKind,
			"kind must be DatabaseCluster or DatabaseClusterReference",
		)
	}

	_, resp, err := godoClient.Databases.GetUser(ctx, clusterUUID, user.Spec.Username)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return warnings, fmt.Errorf("failed to look up database: %v", err)
	}
	if err == nil {
		return warnings, field.Duplicate(field.NewPath("spec").Child("username"), user.Spec.Username)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserValidator) ValidateUpdate(ctx context.Context, oldUser, newUser *v1alpha1.DatabaseUser) (warnings admission.Warnings, err error) {
	databaseuserlog.Info("validate update", "name", newUser.Name)

	usernamePath := field.NewPath("spec").Child("username")
	if newUser.Spec.Username != oldUser.Spec.Username {
		return warnings, field.Forbidden(usernamePath, "username is immutable")
	}
	clusterPath := field.NewPath("spec").Child("cluster")
	if !cmp.Equal(newUser.Spec.Cluster, oldUser.Spec.Cluster) {
		return warnings, field.Forbidden(clusterPath, "cluster is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserValidator) ValidateDelete(ctx context.Context, user *v1alpha1.DatabaseUser) (warnings admission.Warnings, err error) {
	databaseuserlog.Info("validate delete", "name", user.Name)
	return warnings, nil
}
