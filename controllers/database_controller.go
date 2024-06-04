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
	"net/http"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GodoClient *godo.Client
}

//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databases/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databases/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	ll := log.FromContext(ctx)
	ll.Info("reconciling Database", "name", req.Name)

	var database v1alpha1.Database
	err := r.Get(ctx, req.NamespacedName, &database)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return result, nil
		}
		return result, fmt.Errorf("failed to get Database %s: %s", req.NamespacedName, err)
	}

	originalDatabase := database.DeepCopy()
	inDeletion := !database.DeletionTimestamp.IsZero()

	defer func() {
		var (
			updated = false
			errs    []error
		)

		if !cmp.Equal(database.Finalizers, originalDatabase.Finalizers) {
			ll.Info("updating Database finalizers")
			if err := r.Patch(ctx, database.DeepCopy(), client.MergeFrom(originalDatabase)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update Database: %s", err))
			} else {
				updated = true
			}
		}

		if diff := cmp.Diff(database.Status, originalDatabase.Status); diff != "" {
			ll.WithValues("diff", diff).Info("status diff detected")

			if err := r.Status().Patch(ctx, &database, client.MergeFrom(originalDatabase)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update Database status: %s", err))
			} else {
				updated = true
			}
		}

		if len(errs) == 0 {
			if updated {
				ll.Info("Database update succeeded")
			} else {
				ll.Info("no Database update necessary")
			}
		}

		retErr = utilerror.NewAggregate(append([]error{retErr}, errs...))
	}()

	if inDeletion {
		ll.Info("deleting Database")
		if database.Status.ClusterUUID == "" {
			// Database was never actually created; nothing to do.
			controllerutil.RemoveFinalizer(&database, finalizerName)
			return ctrl.Result{}, nil
		}
		return r.reconcileDeletedDB(ctx, database.Status.ClusterUUID, &database)
	}

	var (
		clusterUUID   = database.Status.ClusterUUID
		clusterStatus string
		clusterNN     = types.NamespacedName{
			Namespace: database.Namespace,
			Name:      database.Spec.Cluster.Name,
		}
	)

	// If we haven't noted the cluster's UUID yet, look it up. Subsequent
	// reconciles won't have to do this.
	if clusterUUID == "" {
		switch database.Spec.Cluster.Kind {
		case v1alpha1.DatabaseClusterKind:
			var cluster v1alpha1.DatabaseCluster
			if err := r.Get(ctx, clusterNN, &cluster); err != nil {
				return result, fmt.Errorf("failed to get DatabaseCluster %s: %s", clusterNN.Name, err)
			}
			clusterUUID = cluster.Status.UUID
			clusterStatus = cluster.Status.Status
		case v1alpha1.DatabaseClusterReferenceKind:
			var clusterRef v1alpha1.DatabaseClusterReference
			if err := r.Get(ctx, clusterNN, &clusterRef); err != nil {
				return result, fmt.Errorf("failed to get DatabaseClusterReference %s: %s", clusterNN.Name, err)
			}
			clusterUUID = clusterRef.Spec.UUID
			clusterStatus = clusterRef.Status.Status
		default:
			// Validating webhook should ensure we never get here.
			return result, fmt.Errorf("unexpected Kind for Cluster: %s", database.Spec.Cluster.Kind)
		}

		// Database creation will fail if the cluster is still being created. Schedule a
		// quick retry in those cases so we don't exponentially back off.
		if clusterStatus == "" || clusterStatus == "creating" {
			ll.Info("cluster is still creating; waiting to create database")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		database.Status.ClusterUUID = clusterUUID
	}

	database.Status.ClusterUUID = clusterUUID
	ll.Info("reconciling Database")
	return r.reconcileDB(ctx, clusterUUID, &database)
}

func (r *DatabaseReconciler) reconcileDB(ctx context.Context, clusterUUID string, database *v1alpha1.Database) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues(
		"cluster_uuid", clusterUUID,
		"name", database.Spec.Name,
	)

	// The validating webhook checks that the database doesn't already exist, so we
	// assume that if we find it to exist now we created it. If the database was
	// created between validation passing and getting here, we could assume
	// ownership of an existing DB database. That's not ideal, but since databases don't
	// have an ID other than the databasename we don't have a way to distinguish for
	// sure.

	dbName, resp, err := r.GodoClient.Databases.GetDB(ctx, clusterUUID, database.Spec.Name)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return ctrl.Result{}, fmt.Errorf("checking for existing DB : %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		createReq := &godo.DatabaseCreateDBRequest{
			Name: database.Spec.Name,
		}
		dbName, _, err = r.GodoClient.Databases.CreateDB(ctx, clusterUUID, createReq)
		if err != nil {
			ll.Error(err, "unable to create database")
			return ctrl.Result{}, fmt.Errorf("creating DB database: %v", err)
		}
	}

	controllerutil.AddFinalizer(database, finalizerName)

	err = r.ensureOwnedObjects(ctx, database, dbName)
	if err != nil {
		ll.Error(err, "unable to ensure database-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring database-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DatabaseReconciler) ensureOwnedObjects(ctx context.Context, database *v1alpha1.Database, dbName *godo.DatabaseDB) error {
	// For some database engines the password is not returned when fetching a
	// database, only on initial creation. Avoid creating or updating the database
	// credentials secret if the password is empty, so we don't clear the
	// password after creation.

	return nil
}

func (r *DatabaseReconciler) reconcileDeletedDB(ctx context.Context, clusterUUID string, database *v1alpha1.Database) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues(
		"cluster_uuid", clusterUUID,
		"name", database.Spec.Name,
	)

	_, err := r.GodoClient.Databases.DeleteDB(ctx, clusterUUID, database.Spec.Name)
	if err != nil {
		ll.Error(err, "unable to delete database")
		return ctrl.Result{}, fmt.Errorf("deleting database: %v", err)
	}
	controllerutil.RemoveFinalizer(database, finalizerName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.Database{}).
		Complete(r)
}
