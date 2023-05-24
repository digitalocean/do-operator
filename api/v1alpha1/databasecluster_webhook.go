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
	"strconv"

	"github.com/digitalocean/godo"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var databaseclusterlog = logf.Log.WithName("databasecluster-resource")

func (r *DatabaseCluster) SetupWebhookWithManager(mgr ctrl.Manager, godoClient *godo.Client) error {
	initGlobalGodoClient(godoClient)

	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-databases-digitalocean-com-v1alpha1-databasecluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=databases.digitalocean.com,resources=databaseclusters,verbs=create;update,versions=v1alpha1,name=vdatabasecluster.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &DatabaseCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseCluster) ValidateCreate() (warnings admission.Warnings, err error) {
	databaseclusterlog.Info("validate create", "name", r.Name)
	ctx := context.TODO()

	godoReq := r.Spec.ToGodoValidateCreateRequest()
	req, err := godoClient.NewRequest(ctx, http.MethodPost, "/v2/databases", godoReq)
	if err != nil {
		return warnings, fmt.Errorf("creating http request: %v", err)
	}
	_, err = godoClient.Do(ctx, req, nil)
	if err != nil {
		var (
			specField = field.NewPath("spec")
			godoErr   = &godo.ErrorResponse{}
		)
		if errors.As(err, &godoErr) {
			// In some cases we can map specific errors back to specific spec
			// fields to provide a more informative error message. For any other
			// messages, show the raw error from the API.
			invalidSpecErr := field.Invalid(specField, r.Spec, godoErr.Message)

			// Get options so we can show the valid engines, sizes, etc.
			opts, _, err := godoClient.Databases.ListOptions(ctx)
			if err != nil {
				databaseclusterlog.Error(err, "getting database options from the DigitalOcean api")
				return warnings, invalidSpecErr
			}
			engineOpts, haveOpts := engineOptsFromOptions(opts, r.Spec.Engine)

			switch godoErr.Message {
			case "invalid cluster name":
				return warnings, field.Invalid(specField.Child("name"), r.Spec.Name, godoErr.Message)
			case "invalid node count":
				var validNumNodes []string
				if !haveOpts {
					return warnings, field.Invalid(specField.Child("numNodes"), r.Spec.NumNodes, godoErr.Message)
				}
				for _, layout := range engineOpts.Layouts {
					validNumNodes = append(validNumNodes, strconv.Itoa(layout.NodeNum))
				}
				return warnings, field.NotSupported(specField.Child("numNodes"), r.Spec.NumNodes, validNumNodes)
			case "invalid engine":
				// The options API doesn't return us the list of engines in a
				// friendly format, so we just hardcode them.
				return warnings, field.NotSupported(specField.Child("engine"), r.Spec.Engine, []string{"mysql", "pg", "redis", "mongodb"})
			case "invalid size":
				if !haveOpts {
					return warnings, field.Invalid(specField.Child("size"), r.Spec.Size, godoErr.Message)
				}
				for _, layout := range engineOpts.Layouts {
					if layout.NodeNum == int(r.Spec.NumNodes) {
						return warnings, field.NotSupported(specField.Child("size"), r.Spec.Size, layout.Sizes)
					}
				}
				return warnings, field.Invalid(specField.Child("size"), r.Spec.Size, godoErr.Message)
			case "invalid region":
				if !haveOpts {
					return warnings, field.Invalid(specField.Child("region"), r.Spec.Region, godoErr.Message)
				}
				return warnings, field.NotSupported(specField.Child("region"), r.Spec.Region, engineOpts.Regions)
			case "invalid cluster engine version":
				if !haveOpts {
					return warnings, field.Invalid(specField.Child("version"), r.Spec.Version, godoErr.Message)
				}
				return warnings, field.NotSupported(specField.Child("version"), r.Spec.Version, engineOpts.Versions)
			}

			return warnings, invalidSpecErr
		}

		return warnings, field.Invalid(specField, r.Spec, err.Error())
	}

	return warnings, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseCluster) ValidateUpdate(old runtime.Object) (warnings admission.Warnings, err error) {
	databaseclusterlog.Info("validate update", "name", r.Name)
	ctx := context.TODO()

	oldCluster, ok := old.(*DatabaseCluster)
	if !ok {
		return warnings, fmt.Errorf("old is unexpected type %T", old)
	}

	enginePath := field.NewPath("spec").Child("engine")
	if oldCluster.Spec.Engine != r.Spec.Engine {
		return warnings, field.Forbidden(enginePath, "engine is immutable")
	}
	namePath := field.NewPath("spec").Child("name")
	if oldCluster.Spec.Name != r.Spec.Name {
		return warnings, field.Forbidden(namePath, "name is immutable")
	}
	// TODO(awg) Remove once we support upgrades in the controller.
	versionPath := field.NewPath("spec").Child("version")
	if oldCluster.Spec.Version != r.Spec.Version {
		return warnings, field.Forbidden(versionPath, "database upgrades are not yet supported in the do-operator")
	}
	// TODO(awg) Remove once we support migrations in the controller.
	regionPath := field.NewPath("spec").Child("region")
	if oldCluster.Spec.Region != r.Spec.Region {
		return warnings, field.Forbidden(regionPath, "database region migrations are not yet supported in the do-operator")
	}

	opts, _, err := godoClient.Databases.ListOptions(ctx)
	if err != nil {
		return warnings, fmt.Errorf("getting database options from the DigitalOcean api: %v", err)
	}
	engineOpts, ok := engineOptsFromOptions(opts, r.Spec.Engine)
	if !ok {
		// We *should* get options back for all supported engines, but if the
		// API is missing one we shouldn't block the user from updating.
		return warnings, nil
	}

	var (
		selectedLayout godo.DatabaseLayout
		numNodesValid  bool
		validNumNodes  []string
		sizeValid      bool
	)
	for _, layout := range engineOpts.Layouts {
		validNumNodes = append(validNumNodes, strconv.Itoa(layout.NodeNum))
		if layout.NodeNum == int(r.Spec.NumNodes) {
			numNodesValid = true
			selectedLayout = layout
			break
		}
	}
	if !numNodesValid {
		numNodesPath := field.NewPath("spec").Child("numNodes")
		return warnings, field.NotSupported(numNodesPath, r.Spec.NumNodes, validNumNodes)
	}
	for _, size := range selectedLayout.Sizes {
		if size == r.Spec.Size {
			sizeValid = true
			break
		}
	}
	if !sizeValid {
		sizePath := field.NewPath("spec").Child("size")
		return warnings, field.NotSupported(sizePath, r.Spec.Size, selectedLayout.Sizes)
	}

	return warnings, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *DatabaseCluster) ValidateDelete() (warnings admission.Warnings, err error) {
	databaseclusterlog.Info("validate delete", "name", r.Name)
	return warnings, nil
}
