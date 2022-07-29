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

package controllers

import (
	"context"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilerror "k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	databasesv1alpha1 "github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	"github.com/google/go-cmp/cmp"
)

// DatabaseClusterReferenceReconciler reconciles a DatabaseClusterReference object
type DatabaseClusterReferenceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GodoClient *godo.Client
}

//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusterreferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusterreferences/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusterreferences/finalizers,verbs=update

var (
	// clusterReferenceRefereshTime is how often we refresh the
	// DatabaseClusterReference status from the DO API. It's a variable so we
	// can adjust it for expedient integration testing.
	clusterReferenceRefreshTime = 5 * time.Minute
)

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseClusterReferenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	ll := log.FromContext(ctx)
	ll.Info("reconciling DatabaseClusterReference", "name", req.Name)

	var ref v1alpha1.DatabaseClusterReference
	err := r.Get(ctx, req.NamespacedName, &ref)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return result, nil
		}
		return result, fmt.Errorf("failed to get DatabaseClusterReference %s: %s", req.NamespacedName, err)
	}

	originalRef := ref.DeepCopy()

	defer func() {
		var (
			updated = false
			errs    []error
		)

		if diff := cmp.Diff(ref.Status, originalRef.Status); diff != "" {
			ll.WithValues("diff", diff).Info("status diff detected")

			if err := r.Status().Patch(ctx, &ref, client.MergeFrom(originalRef)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseClusterReference status: %s", err))
			} else {
				updated = true
			}
		}

		if len(errs) == 0 {
			if updated {
				ll.Info("DatabaseClusterReference update succeeded")
			} else {
				ll.Info("no DatabaseClusterReference update necessary")
			}
		}

		retErr = utilerror.NewAggregate(append([]error{retErr}, errs...))
	}()

	db, _, err := r.GodoClient.Databases.Get(ctx, ref.Spec.UUID)
	if err != nil {
		ll.Error(err, "unable to fetch existing DB")
		return ctrl.Result{}, fmt.Errorf("getting existing DB cluster: %v", err)
	}

	ref.Status.Engine = db.EngineSlug
	ref.Status.Name = db.Name
	ref.Status.Version = db.VersionSlug
	ref.Status.NumNodes = int64(db.NumNodes)
	ref.Status.Size = db.SizeSlug
	ref.Status.Region = db.RegionSlug
	ref.Status.Status = db.Status
	ref.Status.CreatedAt = metav1.NewTime(db.CreatedAt)

	err = r.ensureOwnedObjects(ctx, &ref, db)
	if err != nil {
		ll.Error(err, "unable to ensure DB-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring DB-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: clusterReferenceRefreshTime}, nil
}

func (r *DatabaseClusterReferenceReconciler) ensureOwnedObjects(ctx context.Context, cluster *v1alpha1.DatabaseClusterReference, db *godo.Database) error {
	objs := []client.Object{}
	if db.Connection != nil {
		objs = append(objs, connectionConfigMapForDB("-connection", cluster, db.Connection))
	}
	if db.PrivateConnection != nil {
		objs = append(objs, connectionConfigMapForDB("-private-connection", cluster, db.PrivateConnection))
	}

	if db.Connection != nil && db.Connection.Password != "" {
		// MongoDB doesn't return the default user password with the DB except
		// on creation. Don't update the credentials if the password is empty,
		// but create the secret if we have the password.
		objs = append(objs, credentialsSecretForDefaultDBUser(cluster, db))
	}

	for _, obj := range objs {
		controllerutil.SetControllerReference(cluster, obj, r.Scheme)
		if err := r.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("do-operator")); err != nil {
			return fmt.Errorf("applying object %s: %s", client.ObjectKeyFromObject(obj), err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseClusterReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseClusterReference{}).
		Complete(r)
}
