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

// DatabaseUserReferenceReconciler reconciles a DatabaseUserReference object
type DatabaseUserReferenceReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GodoClient *godo.Client
}

//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseuserreferences,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseuserreferences/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseuserreferences/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseUserReferenceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	ll := log.FromContext(ctx)
	ll.Info("reconciling DatabaseUserReference", "name", req.Name)

	var userRef v1alpha1.DatabaseUserReference
	err := r.Get(ctx, req.NamespacedName, &userRef)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return result, nil
		}
		return result, fmt.Errorf("failed to get DatabaseUserReference %s: %s", req.NamespacedName, err)
	}

	originalUserRef := userRef.DeepCopy()

	defer func() {
		var (
			updated = false
			errs    []error
		)

		if diff := cmp.Diff(userRef.Status, originalUserRef.Status); diff != "" {
			ll.WithValues("diff", diff).Info("status diff detected")

			if err := r.Status().Patch(ctx, &userRef, client.MergeFrom(originalUserRef)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseUserReference status: %s", err))
			} else {
				updated = true
			}
		}

		if len(errs) == 0 {
			if updated {
				ll.Info("DatabaseUserReference update succeeded")
			} else {
				ll.Info("no DatabaseUserReference update necessary")
			}
		}

		retErr = utilerror.NewAggregate(append([]error{retErr}, errs...))
	}()

	var (
		clusterUUID = userRef.Status.ClusterUUID
		clusterNN   = types.NamespacedName{
			Namespace: userRef.Namespace,
			Name:      userRef.Spec.Cluster.Name,
		}
	)

	// If we haven't noted the cluster's UUID yet, look it up. Subsequent
	// reconciles won't have to do this.
	if clusterUUID == "" {
		switch userRef.Spec.Cluster.Kind {
		case v1alpha1.DatabaseClusterKind:
			var cluster v1alpha1.DatabaseCluster
			if err := r.Get(ctx, clusterNN, &cluster); err != nil {
				return result, fmt.Errorf("failed to get DatabaseCluster %s: %s", clusterNN.Name, err)
			}
			clusterUUID = cluster.Status.UUID
		case v1alpha1.DatabaseClusterReferenceKind:
			var clusterRef v1alpha1.DatabaseClusterReference
			if err := r.Get(ctx, clusterNN, &clusterRef); err != nil {
				return result, fmt.Errorf("failed to get DatabaseClusterReference %s: %s", clusterNN.Name, err)
			}
			clusterUUID = clusterRef.Spec.UUID
		default:
			// Validating webhook should ensure we never get here.
			return result, fmt.Errorf("unexpected Kind for Cluster: %s", userRef.Spec.Cluster.Kind)
		}

		userRef.Status.ClusterUUID = clusterUUID
	}

	ll.Info("reconciling DatabaseUserReference")
	return r.reconcileDBUserReference(ctx, clusterUUID, &userRef)
}

func (r *DatabaseUserReferenceReconciler) reconcileDBUserReference(ctx context.Context, clusterUUID string, userRef *v1alpha1.DatabaseUserReference) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues(
		"cluster_uuid", clusterUUID,
		"user_name", userRef.Spec.Username,
	)

	// The validating webhook checks that the user exists, so normally this
	// should work. However, the user could have been deleted in which case
	// we'll fail and back off in case it gets re-created.
	dbUser, _, err := r.GodoClient.Databases.GetUser(ctx, clusterUUID, userRef.Spec.Username)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("looking up DB user: %v", err)
	}

	userRef.Status.Role = dbUser.Role

	err = r.ensureOwnedObjects(ctx, userRef, dbUser)
	if err != nil {
		ll.Error(err, "unable to ensure user-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring user-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DatabaseUserReferenceReconciler) ensureOwnedObjects(ctx context.Context, userRef *v1alpha1.DatabaseUserReference, dbUser *godo.DatabaseUser) error {
	obj := credentialsSecretForDBUser(userRef, dbUser)
	controllerutil.SetControllerReference(userRef, obj, r.Scheme)
	if err := r.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("do-operator")); err != nil {
		return fmt.Errorf("applying object %s: %s", client.ObjectKeyFromObject(obj), err)
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseUserReferenceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseUserReference{}).
		Complete(r)
}
