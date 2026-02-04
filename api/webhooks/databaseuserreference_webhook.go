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
	"errors"
	"fmt"
	"net/http"
	"strings"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	"github.com/google/go-cmp/cmp"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

// log is for logging in this package.
var databaseuserreferencelog = logf.Log.WithName("databaseuserreference-resource")

func SetupDatabaseUserReferenceWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)
	initGlobalK8sClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr, &v1alpha1.DatabaseUserReference{}).
		WithValidator(&DatabaseUserReferenceValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseuserreference,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseuserreferences,verbs=create;update,versions=v1alpha1,name=vdatabaseuserreference.kb.io,admissionReviewVersions=v1
type DatabaseUserReferenceValidator struct{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserReferenceValidator) ValidateCreate(ctx context.Context, ref *v1alpha1.DatabaseUserReference) (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate create", "name", ref.Name)

	clusterPath := field.NewPath("spec").Child("cluster")

	clusterAPIGroup := pointer.StringDeref(ref.Spec.Cluster.APIGroup, "")
	if clusterAPIGroup != v1alpha1.GroupVersion.Group {
		return warnings, field.Invalid(clusterPath.Child("apiGroup"), clusterAPIGroup, "apiGroup must be "+v1alpha1.GroupVersion.Group)
	}

	var (
		clusterNN = types.NamespacedName{
			Namespace: ref.Namespace,
			Name:      ref.Spec.Cluster.Name,
		}
		clusterKind = ref.Spec.Cluster.Kind
		clusterUUID string
		engine      string
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
		engine = cluster.Spec.Engine
	case strings.ToLower(v1alpha1.DatabaseClusterReferenceKind):
		var clusterRef v1alpha1.DatabaseClusterReference
		if err := webhookClient.Get(ctx, clusterNN, &clusterRef); err != nil {
			if kerrors.IsNotFound(err) {
				return warnings, field.NotFound(clusterPath, clusterNN)
			}
			return warnings, fmt.Errorf("failed to fetch DatabaseClusterReference %s: %s", clusterNN, err)
		}
		clusterUUID = clusterRef.Spec.UUID
		engine = clusterRef.Status.Engine
	default:
		return warnings, field.TypeInvalid(
			clusterPath.Child("kind"),
			clusterKind,
			"kind must be DatabaseCluster or DatabaseClusterReference",
		)
	}

	switch engine {
	case "":
		// This is most likely for DatabaseClusterReferences that haven't been
		// reconciled yet.
		return warnings, errors.New("could not determine database engine")
	case "mongodb":
		return warnings, field.Invalid(clusterPath, ref.Spec.Cluster, "user references are not supported for MongoDB databases")
	case "redis":
		return warnings, field.Invalid(clusterPath, ref.Spec.Cluster, "user management is not supported for Redis databases")
	}

	_, resp, err := godoClient.Databases.GetUser(ctx, clusterUUID, ref.Spec.Username)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return warnings, field.Invalid(
				field.NewPath("spec").Child("username"),
				ref.Spec.Username,
				"user does not exist; you must create it before referencing it",
			)
		}
		return warnings, fmt.Errorf("failed to look up user: %v", err)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserReferenceValidator) ValidateUpdate(ctx context.Context, oldRef, newRef *v1alpha1.DatabaseUserReference) (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate update", "name", newRef.Name)

	usernamePath := field.NewPath("spec").Child("username")
	if newRef.Spec.Username != oldRef.Spec.Username {
		return warnings, field.Forbidden(usernamePath, "username is immutable")
	}
	clusterPath := field.NewPath("spec").Child("cluster")
	if !cmp.Equal(newRef.Spec.Cluster, oldRef.Spec.Cluster) {
		return warnings, field.Forbidden(clusterPath, "cluster is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseUserReferenceValidator) ValidateDelete(ctx context.Context, ref *v1alpha1.DatabaseUserReference) (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate delete", "name", ref.Name)

	return warnings, nil
}
