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

package v1alpha1

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/digitalocean/godo"
	"github.com/google/go-cmp/cmp"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var databaseuserlog = logf.Log.WithName("databaseuser-resource")

func (r *DatabaseUser) SetupWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)
	initGlobalK8sClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseuser,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseusers,verbs=create;update,versions=v1alpha1,name=vdatabaseuser.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DatabaseUser{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUser) ValidateCreate() error {
	databaseuserlog.Info("validate create", "name", r.Name)
	ctx := context.TODO()

	clusterPath := field.NewPath("spec").Child("cluster")

	clusterAPIGroup := pointer.StringDeref(r.Spec.Cluster.APIGroup, "")
	if clusterAPIGroup != GroupVersion.Group {
		return field.Invalid(clusterPath.Child("apiGroup"), clusterAPIGroup, "apiGroup must be "+GroupVersion.Group)
	}

	var (
		clusterNN = types.NamespacedName{
			Namespace: r.Namespace,
			Name:      r.Spec.Cluster.Name,
		}
		clusterKind = r.Spec.Cluster.Kind
		clusterUUID string
	)

	switch strings.ToLower(clusterKind) {
	case strings.ToLower(DatabaseClusterKind):
		var cluster DatabaseCluster
		if err := webhookClient.Get(ctx, clusterNN, &cluster); err != nil {
			if kerrors.IsNotFound(err) {
				return field.NotFound(clusterPath, clusterNN)
			}
			return fmt.Errorf("failed to fetch DatabaseCluster %s: %s", clusterNN, err)
		}
		clusterUUID = cluster.Status.UUID
	case strings.ToLower(DatabaseClusterReferenceKind):
		var clusterRef DatabaseClusterReference
		if err := webhookClient.Get(ctx, clusterNN, &clusterRef); err != nil {
			if kerrors.IsNotFound(err) {
				return field.NotFound(clusterPath, clusterNN)
			}
			return fmt.Errorf("failed to fetch DatabaseClusterReference %s: %s", clusterNN, err)
		}
		clusterUUID = clusterRef.Spec.UUID
	default:
		return field.TypeInvalid(
			clusterPath.Child("kind"),
			clusterKind,
			"kind must be DatabaseCluster or DatabaseClusterReference",
		)
	}

	_, resp, err := godoClient.Databases.GetUser(ctx, clusterUUID, r.Spec.Username)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to look up database: %v", err)
	}
	if err == nil {
		return field.Duplicate(field.NewPath("spec").Child("username"), r.Spec.Username)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUser) ValidateUpdate(old runtime.Object) error {
	databaseuserlog.Info("validate update", "name", r.Name)

	oldUser, ok := old.(*DatabaseUser)
	if !ok {
		return fmt.Errorf("old is unexpected type %T", old)
	}
	usernamePath := field.NewPath("spec").Child("username")
	if r.Spec.Username != oldUser.Spec.Username {
		return field.Forbidden(usernamePath, "username is immutable")
	}
	clusterPath := field.NewPath("spec").Child("cluster")
	if !cmp.Equal(r.Spec.Cluster, oldUser.Spec.Cluster) {
		return field.Forbidden(clusterPath, "cluster is immutable")
	}

	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUser) ValidateDelete() error {
	databaseuserlog.Info("validate delete", "name", r.Name)
	return nil
}
