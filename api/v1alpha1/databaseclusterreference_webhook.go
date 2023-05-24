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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/digitalocean/godo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var databaseclusterreferencelog = logf.Log.WithName("databaseclusterreference-resource")

func (r *DatabaseClusterReference) SetupWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databaseclusterreference,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseclusterreferences,verbs=create;update,versions=v1alpha1,name=vdatabaseclusterreference.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DatabaseClusterReference{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseClusterReference) ValidateCreate() (warnings admission.Warnings, err error) {
	databaseclusterreferencelog.Info("validate create", "name", r.Name)

	dbUUID := r.Spec.UUID
	uuidPath := field.NewPath("spec").Child("uuid")
	_, resp, err := godoClient.Databases.Get(context.TODO(), dbUUID)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			return warnings, field.Invalid(uuidPath, dbUUID, "database does not exist; you must create it before referencing it")
		}
		return warnings, fmt.Errorf("failed to look up database: %v", err)
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseClusterReference) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	databaseclusterreferencelog.Info("validate update", "name", r.Name)

	oldDBRef, ok := old.(*DatabaseClusterReference)
	if !ok {
		return warnings, fmt.Errorf("old is unexpected type %T", old)
	}
	uuidPath := field.NewPath("spec").Child("uuid")
	if r.Spec.UUID != oldDBRef.Spec.UUID {
		return warnings, field.Forbidden(uuidPath, "database UUID is immutable")
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseClusterReference) ValidateDelete() (warnings admission.Warnings, err error) {
	databaseclusterreferencelog.Info("validate delete", "name", r.Name)
	return warnings, nil
}
