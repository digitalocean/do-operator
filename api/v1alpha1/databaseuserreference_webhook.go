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
	"errors"
	"fmt"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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
var databaseuserreferencelog = logf.Log.WithName("databaseuserreference-resource")

func (r *DatabaseUserReference) SetupWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)
	initGlobalK8sClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseuserreference,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseuserreferences,verbs=create;update,versions=v1alpha1,name=vdatabaseuserreference.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DatabaseUserReference{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUserReference) ValidateCreate() (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate create", "name", r.Name)
	ctx := context.TODO()

	clusterPath := field.NewPath("spec").Child("cluster")

	clusterAPIGroup := pointer.StringDeref(r.Spec.Cluster.APIGroup, "")
	if clusterAPIGroup != GroupVersion.Group {
		return warnings, field.Invalid(clusterPath.Child("apiGroup"), clusterAPIGroup, "apiGroup must be "+GroupVersion.Group)
	}

	var (
		clusterNN = types.NamespacedName{
			Namespace: r.Namespace,
			Name:      r.Spec.Cluster.Name,
		}
		clusterKind = r.Spec.Cluster.Kind
		clusterUUID string
		engine      string
	)

	switch strings.ToLower(clusterKind) {
	case strings.ToLower(DatabaseClusterKind):
		var cluster DatabaseCluster
		if err := webhookClient.Get(ctx, clusterNN, &cluster); err != nil {
			if kerrors.IsNotFound(err) {
				return warnings, field.NotFound(clusterPath, clusterNN)
			}
			return warnings, fmt.Errorf("failed to fetch DatabaseCluster %s: %s", clusterNN, err)
		}
		clusterUUID = cluster.Status.UUID
		engine = cluster.Spec.Engine
	case strings.ToLower(DatabaseClusterReferenceKind):
		var clusterRef DatabaseClusterReference
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
		return warnings, field.Invalid(clusterPath, r.Spec.Cluster, "user references are not supported for MongoDB databases")
	case "redis":
		return warnings, field.Invalid(clusterPath, r.Spec.Cluster, "user management is not supported for Redis databases")
	}

	_, resp, err := godoClient.Databases.GetUser(ctx, clusterUUID, r.Spec.Username)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return warnings, field.Invalid(
				field.NewPath("spec").Child("username"),
				r.Spec.Username,
				"user does not exist; you must create it before referencing it",
			)
		}
		return warnings, fmt.Errorf("failed to look up user: %v", err)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUserReference) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate update", "name", r.Name)

	oldRef, ok := old.(*DatabaseUserReference)
	if !ok {
		return warnings, fmt.Errorf("old is unexpected type %T", old)
	}
	usernamePath := field.NewPath("spec").Child("username")
	if r.Spec.Username != oldRef.Spec.Username {
		return warnings, field.Forbidden(usernamePath, "username is immutable")
	}
	clusterPath := field.NewPath("spec").Child("cluster")
	if !cmp.Equal(r.Spec.Cluster, oldRef.Spec.Cluster) {
		return warnings, field.Forbidden(clusterPath, "cluster is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseUserReference) ValidateDelete() (warnings admission.Warnings, err error) {
	databaseuserreferencelog.Info("validate delete", "name", r.Name)

	return warnings, nil
}
