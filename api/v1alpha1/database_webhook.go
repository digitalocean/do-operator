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

	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

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
var databaselog = logf.Log.WithName("database-resource")

func (r *Database) SetupWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)
	initGlobalK8sClient(mgr.GetClient())

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-database,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databases,verbs=create;update,versions=v1alpha1,name=vdatabase.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &Database{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateCreate() (warnings admission.Warnings, err error) {
	databaselog.Info("validate create", "name", r.Name)
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
	case strings.ToLower(DatabaseClusterReferenceKind):
		var clusterRef DatabaseClusterReference
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

	_, resp, err := godoClient.Databases.GetDB(ctx, clusterUUID, r.Spec.Name)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return warnings, fmt.Errorf("failed to look up database: %v", err)
	}
	if err == nil {
		return warnings, field.Duplicate(field.NewPath("spec").Child("name"), r.Spec.Name)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	databaselog.Info("validate update", "name", r.Name)

	oldDatabase, ok := old.(*Database)
	if !ok {
		return warnings, fmt.Errorf("old is unexpected type %T", old)
	}
	namePath := field.NewPath("spec").Child("name")
	if r.Spec.Name != oldDatabase.Spec.Name {
		return warnings, field.Forbidden(namePath, "name is immutable")
	}
	clusterPath := field.NewPath("spec").Child("cluster")
	if !cmp.Equal(r.Spec.Cluster, oldDatabase.Spec.Cluster) {
		return warnings, field.Forbidden(clusterPath, "cluster is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Database) ValidateDelete() (warnings admission.Warnings, err error) {
	databaselog.Info("validate delete", "name", r.Name)
	return warnings, nil
}
