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

const (
	finalizerName = "databases.digitalocean.com"
)

// DatabaseClusterReconciler reconciles a DatabaseCluster object
type DatabaseClusterReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GodoClient *godo.Client
}

//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseclusters/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps;secrets,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	ll := log.FromContext(ctx)
	ll.Info("reconciling DatabaseCluster", "name", req.Name)

	var cluster v1alpha1.DatabaseCluster
	err := r.Get(ctx, req.NamespacedName, &cluster)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return result, nil
		}
		return result, fmt.Errorf("failed to get DatabaseCluster %s: %s", req.NamespacedName, err)
	}

	originalCluster := cluster.DeepCopy()
	inDeletion := !cluster.DeletionTimestamp.IsZero()

	defer func() {
		var (
			updated = false
			errs    []error
		)

		if !cmp.Equal(cluster.Finalizers, originalCluster.Finalizers) {
			ll.Info("updating DatabaseCluster finalizers")
			if err := r.Patch(ctx, cluster.DeepCopy(), client.MergeFrom(originalCluster)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseCluster: %s", err))
			} else {
				updated = true
			}
		}

		if diff := cmp.Diff(cluster.Status, originalCluster.Status); diff != "" {
			ll.WithValues("diff", diff).Info("status diff detected")

			if err := r.Status().Patch(ctx, &cluster, client.MergeFrom(originalCluster)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseCluster status: %s", err))
			} else {
				updated = true
			}
		}

		if len(errs) == 0 {
			if updated {
				ll.Info("DatabaseCluster update succeeded")
			} else {
				ll.Info("no DatabaseCluster update necessary")
			}
		}

		retErr = utilerror.NewAggregate(append([]error{retErr}, errs...))
	}()

	if inDeletion {
		ll.Info("deleting DatabaseCluster")
		result, err = r.reconcileDeletedDB(ctx, &cluster)
	} else if cluster.Status.UUID != "" {
		ll.Info("reconciling existing DatabaseCluster")
		result, err = r.reconcileExistingDB(ctx, &cluster)
	} else {
		ll.Info("reconciling new DatabaseCluster")
		result, err = r.reconcileNewDB(ctx, &cluster)
	}

	return result, err
}

func (r *DatabaseClusterReconciler) reconcileNewDB(ctx context.Context, cluster *v1alpha1.DatabaseCluster) (ctrl.Result, error) {
	ll := log.FromContext(ctx)

	createReq := cluster.Spec.ToGodoCreateRequest()
	db, _, err := r.GodoClient.Databases.Create(ctx, createReq)
	if err != nil {
		ll.Error(err, "unable to create DB")
		return ctrl.Result{}, fmt.Errorf("creating DB cluster: %v", err)
	}

	controllerutil.AddFinalizer(cluster, finalizerName)
	cluster.Status.UUID = db.ID
	cluster.Status.CreatedAt = metav1.NewTime(db.CreatedAt)
	cluster.Status.Status = db.Status

	ca, _, err := r.GodoClient.Databases.GetCA(ctx, db.ID)
	if err != nil {
		ll.Error(err, "unable to get database CA")
		return ctrl.Result{}, fmt.Errorf("getting database CA: %v", err)
	}

	err = r.ensureOwnedObjects(ctx, cluster, db, ca)
	if err != nil {
		ll.Error(err, "unable to ensure DB-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring DB-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: time.Minute}, nil
}

func (r *DatabaseClusterReconciler) reconcileExistingDB(ctx context.Context, cluster *v1alpha1.DatabaseCluster) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues("db_uuid", cluster.Status.UUID)

	db, _, err := r.GodoClient.Databases.Get(ctx, cluster.Status.UUID)
	if err != nil {
		ll.Error(err, "unable to fetch existing DB")
		return ctrl.Result{}, fmt.Errorf("getting existing DB cluster: %v", err)
	}

	ca, _, err := r.GodoClient.Databases.GetCA(ctx, db.ID)
	if err != nil {
		ll.Error(err, "unable to get existing database database CA")
		return ctrl.Result{}, fmt.Errorf("getting existing database CA: %v", err)
	}

	// Resize if either of the size parameters in the spec has changed.
	if db.NumNodes != int(cluster.Spec.NumNodes) || db.SizeSlug != cluster.Spec.Size {
		ll.Info("resizing database",
			"current_num_nodes", db.NumNodes, "desired_num_nodes", cluster.Spec.NumNodes,
			"current_size", db.SizeSlug, "desired_size", cluster.Spec.Size)
		_, err = r.GodoClient.Databases.Resize(ctx, cluster.Status.UUID, &godo.DatabaseResizeRequest{
			SizeSlug: cluster.Spec.Size,
			NumNodes: int(cluster.Spec.NumNodes),
		})
		if err != nil {
			ll.Error(err, "unable to resize existing DB")
			return ctrl.Result{}, fmt.Errorf("resizing existing DB cluster: %v", err)
		}
		// Requeue immediately so we pick up status updates.
		return ctrl.Result{Requeue: true}, nil
	}

	// Update status. By default we'll reconcile again in a minute, but if the
	// status is "online" (running normally) we can delay longer, and if it's
	// "creating" then we wait less so that we can get the online status ASAP.
	requeueTime := time.Minute
	if db.Status != cluster.Status.Status {
		cluster.Status.Status = db.Status
		if err := r.Status().Update(ctx, cluster); err != nil {
			ll.Error(err, "unable to update DatabaseCluster status")
			return ctrl.Result{}, fmt.Errorf("updating status: %v", err)
		}
		switch db.Status {
		case "online":
			requeueTime = 5 * time.Minute
		case "creating":
			requeueTime = 30 * time.Second
		}
	}

	err = r.ensureOwnedObjects(ctx, cluster, db, ca)
	if err != nil {
		ll.Error(err, "unable to ensure DB-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring DB-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: requeueTime}, nil
}

func (r *DatabaseClusterReconciler) reconcileDeletedDB(ctx context.Context, cluster *v1alpha1.DatabaseCluster) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues("db_uuid", cluster.Status.UUID)

	if cluster.Status.UUID == "" {
		// DB was never created. We can just remove the finalizer.
		controllerutil.RemoveFinalizer(cluster, finalizerName)
		return ctrl.Result{}, nil
	}

	_, err := r.GodoClient.Databases.Delete(ctx, cluster.Status.UUID)
	if err != nil {
		ll.Error(err, "unable to delete DB")
		return ctrl.Result{}, fmt.Errorf("deleting DB: %v", err)
	}
	controllerutil.RemoveFinalizer(cluster, finalizerName)

	return ctrl.Result{}, nil
}

func (r *DatabaseClusterReconciler) ensureOwnedObjects(ctx context.Context, cluster *v1alpha1.DatabaseCluster, db *godo.Database, ca *godo.DatabaseCA) error {
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
		objs = append(objs, credentialsSecretForDefaultDBUser(cluster, db, ca))
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
func (r *DatabaseClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseCluster{}).
		Complete(r)
}
