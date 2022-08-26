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

// DatabaseUserReconciler reconciles a DatabaseUser object
type DatabaseUserReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	GodoClient *godo.Client
}

//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseusers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseusers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=databases.digitalocean.com,resources=databaseusers/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *DatabaseUserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, retErr error) {
	ll := log.FromContext(ctx)
	ll.Info("reconciling DatabaseUser", "name", req.Name)

	var user v1alpha1.DatabaseUser
	err := r.Get(ctx, req.NamespacedName, &user)
	if err != nil {
		if kerrors.IsNotFound(err) {
			return result, nil
		}
		return result, fmt.Errorf("failed to get DatabaseUser %s: %s", req.NamespacedName, err)
	}

	originalUser := user.DeepCopy()
	inDeletion := !user.DeletionTimestamp.IsZero()

	defer func() {
		var (
			updated = false
			errs    []error
		)

		if !cmp.Equal(user.Finalizers, originalUser.Finalizers) {
			ll.Info("updating DatabaseUser finalizers")
			if err := r.Patch(ctx, user.DeepCopy(), client.MergeFrom(originalUser)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseUser: %s", err))
			} else {
				updated = true
			}
		}

		if diff := cmp.Diff(user.Status, originalUser.Status); diff != "" {
			ll.WithValues("diff", diff).Info("status diff detected")

			if err := r.Status().Patch(ctx, &user, client.MergeFrom(originalUser)); err != nil {
				errs = append(errs, fmt.Errorf("failed to update DatabaseUser status: %s", err))
			} else {
				updated = true
			}
		}

		if len(errs) == 0 {
			if updated {
				ll.Info("DatabaseUser update succeeded")
			} else {
				ll.Info("no DatabaseUser update necessary")
			}
		}

		retErr = utilerror.NewAggregate(append([]error{retErr}, errs...))
	}()

	if inDeletion {
		ll.Info("deleting DatabaseUser")
		if user.Status.ClusterUUID == "" {
			// User was never actually created; nothing to do.
			controllerutil.RemoveFinalizer(&user, finalizerName)
			return ctrl.Result{}, nil
		}
		return r.reconcileDeletedDBUser(ctx, user.Status.ClusterUUID, &user)
	}

	var (
		clusterUUID   = user.Status.ClusterUUID
		clusterStatus string
		clusterNN     = types.NamespacedName{
			Namespace: user.Namespace,
			Name:      user.Spec.Cluster.Name,
		}
	)

	// If we haven't noted the cluster's UUID yet, look it up. Subsequent
	// reconciles won't have to do this.
	if clusterUUID == "" {
		switch user.Spec.Cluster.Kind {
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
			return result, fmt.Errorf("unexpected Kind for Cluster: %s", user.Spec.Cluster.Kind)
		}

		// User creation will fail if the cluster is still being created. Schedule a
		// quick retry in those cases so we don't exponentially back off.
		if clusterStatus == "" || clusterStatus == "creating" {
			ll.Info("database is still creating; waiting to create user")
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}

		user.Status.ClusterUUID = clusterUUID
	}

	user.Status.ClusterUUID = clusterUUID
	ll.Info("reconciling DatabaseUser")
	return r.reconcileDBUser(ctx, clusterUUID, &user)
}

func (r *DatabaseUserReconciler) reconcileDBUser(ctx context.Context, clusterUUID string, user *v1alpha1.DatabaseUser) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues(
		"cluster_uuid", clusterUUID,
		"user_name", user.Spec.Username,
	)

	// The validating webhook checks that the user doesn't already exist, so we
	// assume that if we find it to exist now we created it. If the user was
	// created between validation passing and getting here, we could assume
	// ownership of an existing DB user. That's not ideal, but since users don't
	// have an ID other than the username we don't have a way to distinguish for
	// sure.

	dbUser, resp, err := r.GodoClient.Databases.GetUser(ctx, clusterUUID, user.Spec.Username)
	if err != nil && resp.StatusCode != http.StatusNotFound {
		return ctrl.Result{}, fmt.Errorf("checking for existing DB user: %v", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		createReq := &godo.DatabaseCreateUserRequest{
			Name: user.Spec.Username,
		}
		dbUser, _, err = r.GodoClient.Databases.CreateUser(ctx, clusterUUID, createReq)
		if err != nil {
			ll.Error(err, "unable to create user")
			return ctrl.Result{}, fmt.Errorf("creating DB user: %v", err)
		}
	}

	controllerutil.AddFinalizer(user, finalizerName)
	user.Status.Role = dbUser.Role

	err = r.ensureOwnedObjects(ctx, user, dbUser)
	if err != nil {
		ll.Error(err, "unable to ensure user-related objects")
		return ctrl.Result{}, fmt.Errorf("ensuring user-related objects: %v", err)
	}

	return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
}

func (r *DatabaseUserReconciler) ensureOwnedObjects(ctx context.Context, user *v1alpha1.DatabaseUser, dbUser *godo.DatabaseUser) error {
	// For some database engines the password is not returned when fetching a
	// user, only on initial creation. Avoid creating or updating the user
	// credentials secret if the password is empty, so we don't clear the
	// password after creation.
	if dbUser.Password == "" {
		return nil
	}

	obj := credentialsSecretForDBUser(user, dbUser)
	controllerutil.SetControllerReference(user, obj, r.Scheme)
	if err := r.Patch(ctx, obj, client.Apply, client.ForceOwnership, client.FieldOwner("do-operator")); err != nil {
		return fmt.Errorf("applying object %s: %s", client.ObjectKeyFromObject(obj), err)
	}

	return nil
}

func (r *DatabaseUserReconciler) reconcileDeletedDBUser(ctx context.Context, clusterUUID string, user *v1alpha1.DatabaseUser) (ctrl.Result, error) {
	ll := log.FromContext(ctx)
	ll = ll.WithValues(
		"cluster_uuid", clusterUUID,
		"user_name", user.Spec.Username,
	)

	_, err := r.GodoClient.Databases.DeleteUser(ctx, clusterUUID, user.Spec.Username)
	if err != nil {
		ll.Error(err, "unable to delete user")
		return ctrl.Result{}, fmt.Errorf("deleting user: %v", err)
	}
	controllerutil.RemoveFinalizer(user, finalizerName)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseUserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&databasesv1alpha1.DatabaseUser{}).
		Complete(r)
}
