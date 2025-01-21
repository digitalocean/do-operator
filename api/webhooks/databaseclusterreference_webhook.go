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

	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var databaseclusterreferencelog = logf.Log.WithName("databaseclusterreference-resource")

func SetupDatabaseClusterReferenceWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)

	return ctrl.NewWebhookManagedBy(mgr).
		For(&v1alpha1.DatabaseClusterReference{}).
		WithValidator(&DatabaseClusterReferenceValidator{}).
		Complete()
}

// +kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseclusterreference,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseclusterreferences,verbs=create;update,versions=v1alpha1,name=vdatabaseclusterreference.kb.io,admissionReviewVersions=v1
type DatabaseClusterReferenceValidator struct{}

var _ webhook.CustomValidator = &DatabaseClusterReferenceValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseClusterReferenceValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	ref, ok := obj.(*v1alpha1.DatabaseClusterReference)
	if !ok {
		return nil, fmt.Errorf("expected a DatabaseClusterReference object but got %T", obj)
	}

	databaseclusterreferencelog.Info("validate create", "name", ref.Name)

	dbUUID := ref.Spec.UUID
	uuidPath := field.NewPath("spec").Child("uuid")
	_, resp, err := godoClient.Databases.Get(ctx, dbUUID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return warnings, field.Invalid(uuidPath, dbUUID, "database does not exist; you must create it before referencing it")
		}
		return warnings, fmt.Errorf("failed to look up database: %v", err)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseClusterReferenceValidator) ValidateUpdate(ctx context.Context, objOld, objNew runtime.Object) (warnings admission.Warnings, err error) {
	oldRef, ok := objOld.(*v1alpha1.DatabaseClusterReference)
	if !ok {
		return nil, fmt.Errorf("expected a DatabaseClusterReference old object but got %T", objOld)
	}
	newRef, ok := objNew.(*v1alpha1.DatabaseClusterReference)
	if !ok {
		return nil, fmt.Errorf("expected a DatabaseClusterReference new object but got %T", objNew)
	}

	databaseclusterreferencelog.Info("validate update", "name", newRef.Name)

	uuidPath := field.NewPath("spec").Child("uuid")
	if newRef.Spec.UUID != oldRef.Spec.UUID {
		return warnings, field.Forbidden(uuidPath, "database UUID is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (v *DatabaseClusterReferenceValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (warnings admission.Warnings, err error) {
	ref, ok := obj.(*v1alpha1.DatabaseClusterReference)
	if !ok {
		return nil, fmt.Errorf("expected a DatabaseClusterReference object but got %T", obj)
	}

	databaseclusterreferencelog.Info("validate delete", "name", ref.Name)
	return warnings, nil
}
