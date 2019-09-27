package database

import (
	"context"
	"fmt"
	"os"
	"time"

	doopv1alpha1 "github.com/digitalocean/do-operator/pkg/apis/doop/v1alpha1"
	"github.com/digitalocean/do-operator/pkg/do"
	"github.com/digitalocean/godo"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	databaseStatusOnline = "online"
	databaseFinalizer    = "finalizer.database.doop.do.co"
)

var (
	log = logf.Log.WithName("controller_database")

	doAccessToken = os.Getenv("DIGITALOCEAN_ACCESS_TOKEN")
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Database Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileDatabase{
		client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		doClient: do.NewClient(context.Background(), doAccessToken),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("database-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Database
	err = c.Watch(&source.Kind{Type: &doopv1alpha1.Database{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileDatabase implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileDatabase{}

// ReconcileDatabase reconciles a Database object
type ReconcileDatabase struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	doClient *godo.Client
}

// Reconcile reads that state of the cluster for a Database object and makes changes based on the state read
// and what is in the Database.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileDatabase) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Database")

	// Fetch the Database instance
	instance := &doopv1alpha1.Database{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Check if the instance is marked to be deleted, which is indicated by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil {
		if contains(instance.GetFinalizers(), databaseFinalizer) {
			// Run finalization logic for databaseFinalizer. If the
			// finalization logic fails, don't remove the finalizer so
			// that we can retry during the next reconciliation.
			if err := r.finalizeDatabase(reqLogger, instance); err != nil {
				return reconcile.Result{}, err
			}

			// Remove databaseFinalizer.
			// Once all finalizers have been removed, the object will be deleted.
			instance.SetFinalizers(remove(instance.GetFinalizers(), databaseFinalizer))
			err := r.client.Update(context.TODO(), instance)
			if err != nil {
				return reconcile.Result{}, err
			}
		}
		return reconcile.Result{}, nil
	}

	// Add finalizer for this CR
	if !contains(instance.GetFinalizers(), databaseFinalizer) {
		if err := r.addFinalizer(reqLogger, instance); err != nil {
			return reconcile.Result{}, err
		}
	}

	databaseID := instance.Status.ID

	if databaseID == "" {
		// Create the DO database instance.
		database, _, err := r.doClient.Databases.Create(context.Background(), instance.Spec.ToDO())
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Created Database", "Database.Name", database.Name, "Database.ID", database.ID)

		// Populate the instance status with DO object.
		instance.Status.FromDO(database)
		err = r.client.Status().Update(context.Background(), instance)
		if err != nil {
			return reconcile.Result{}, err
		}
		reqLogger.Info("Updated Database status", "Database.Name", database.Name, "Database.ID", database.ID)

		// Create a secret for the database connection.
		_, err = r.createConnectionSecret(reqLogger, instance, database, "-connection")
		if err != nil {
			return reconcile.Result{}, err
		}

		// Create a secret for the database private connection.
		_, err = r.createConnectionSecret(reqLogger, instance, database, "-private-connection")
		if err != nil {
			return reconcile.Result{}, err
		}

		return reconcile.Result{Requeue: true}, nil
	}

	database, _, err := r.doClient.Databases.Get(context.Background(), databaseID)
	if err != nil {
		if errors.IsNotFound(err) {
			reqLogger.Info("Unknown Database", "Database.Name", database.Name, "Database.ID", database.ID)
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// Populate the instance status with DO object.
	instance.Status.FromDO(database)
	err = r.client.Status().Update(context.Background(), instance)
	if err != nil {
		return reconcile.Result{}, err
	}
	reqLogger.Info("Updated Database status", "Database.Name", database.Name, "Database.ID", database.ID)

	// If status is not yet online, then requeue and check again on next reconcile.
	if database.Status != databaseStatusOnline {
		reqLogger.Info("Waiting for Database to be online", "Database.Name", database.Name, "Database.ID", database.ID, "Database.Status", database.Status)
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 30 * time.Second,
		}, nil
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileDatabase) createConnectionSecret(reqLogger logr.Logger, instance *doopv1alpha1.Database, database *godo.Database, secretNameSuffix string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s%s", instance.Name, secretNameSuffix),
			Namespace: instance.Namespace,
			Labels: map[string]string{
				"app": instance.Name,
			},
		},
		StringData: doopv1alpha1.DatabaseConnectionToSringData(database.Connection),
	}
	err := r.client.Create(context.Background(), secret)
	if err != nil {
		return nil, err
	}

	// Set Database instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, secret, r.scheme); err != nil {
		return nil, err
	}

	return secret, nil
}

func (r *ReconcileDatabase) finalizeDatabase(reqLogger logr.Logger, instance *doopv1alpha1.Database) error {
	databaseID := instance.Status.ID
	_, err := r.doClient.Databases.Delete(context.TODO(), databaseID)
	if err != nil {
		return err
	}
	reqLogger.Info("Deleted Database", databaseID, "Database.Name", instance.Spec.Name)
	return nil
}

func (r *ReconcileDatabase) addFinalizer(reqLogger logr.Logger, m *doopv1alpha1.Database) error {
	reqLogger.Info("Adding Finalizer for the Database")
	m.SetFinalizers(append(m.GetFinalizers(), databaseFinalizer))

	// Update CR
	err := r.client.Update(context.TODO(), m)
	if err != nil {
		reqLogger.Error(err, "Failed to update Database with finalizer")
		return err
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
