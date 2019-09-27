package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/digitalocean/do-operator/mocks"
	doopv1alpha1 "github.com/digitalocean/do-operator/pkg/apis/doop/v1alpha1"
	"github.com/digitalocean/godo"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// https://github.com/operator-framework/operator-sdk/blob/master/doc/user/unit-testing.md

func TestDatabaseControllerCreateDelete(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(doopv1alpha1.SchemeGroupVersion, &doopv1alpha1.Database{})

	example := &doopv1alpha1.Database{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example",
			Namespace: "doop",
			Labels: map[string]string{
				"foo": "bar",
			},
		},
	}

	objs := []runtime.Object{example}
	cl := fake.NewFakeClient(objs...)

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()
	mockDatabasesService := mocks.NewMockDatabasesService(mockCtrl)

	fakeDatabase := &godo.Database{
		ID:                "1",
		Name:              "foo",
		Connection:        &godo.DatabaseConnection{},
		PrivateConnection: &godo.DatabaseConnection{},
		MaintenanceWindow: &godo.DatabaseMaintenanceWindow{},
		Status:            "creating",
	}
	mockDatabasesService.EXPECT().Create(gomock.Any(), gomock.Any()).Return(fakeDatabase, nil, nil).Times(1)

	r := &ReconcileDatabase{
		client: cl,
		scheme: s,
		doClient: &godo.Client{
			Databases: mockDatabasesService,
		},
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      example.Name,
			Namespace: example.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.Equal(t, true, res.Requeue)

	database := &doopv1alpha1.Database{}
	err = r.client.Get(context.TODO(), req.NamespacedName, database)
	require.NoError(t, err)
	require.Equal(t, doopv1alpha1.DatabaseStatus{
		ID:                "1",
		Name:              "foo",
		MaintenanceWindow: &doopv1alpha1.DatabaseMaintenanceWindow{},
		Status:            "creating",
	}, database.Status)

	// Check that the connection secret was created.
	connectionSecretNamespace := types.NamespacedName{
		Namespace: example.Namespace,
		Name:      fmt.Sprintf("%s-connection", example.Name),
	}
	secret := &corev1.Secret{}
	err = r.client.Get(context.TODO(), connectionSecretNamespace, secret)
	require.NoError(t, err)
	require.Equal(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      connectionSecretNamespace.Name,
			Namespace: connectionSecretNamespace.Namespace,
			Labels: map[string]string{
				"app": example.Name,
			},
		},
		StringData: map[string]string{
			"uri":      "",
			"database": "",
			"host":     "",
			"port":     "0",
			"user":     "",
			"password": "",
			"ssl":      "false",
		},
	}, secret)

	// Check that the private connection secret was created.
	privateConnectionSecretNamespace := types.NamespacedName{
		Namespace: example.Namespace,
		Name:      fmt.Sprintf("%s-private-connection", example.Name),
	}
	secret = &corev1.Secret{}
	err = r.client.Get(context.TODO(), privateConnectionSecretNamespace, secret)
	require.NoError(t, err)
	require.Equal(t, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      privateConnectionSecretNamespace.Name,
			Namespace: privateConnectionSecretNamespace.Namespace,
			Labels: map[string]string{
				"app": example.Name,
			},
		},
		StringData: map[string]string{
			"uri":      "",
			"database": "",
			"host":     "",
			"port":     "0",
			"user":     "",
			"password": "",
			"ssl":      "false",
		},
	}, secret)

	// Check reconcile after status is online.
	fakeDatabase.Status = databaseStatusOnline
	mockDatabasesService.EXPECT().Get(gomock.Any(), fakeDatabase.ID).Return(fakeDatabase, nil, nil).Times(1)

	res, err = r.Reconcile(req)
	require.NoError(t, err)
	require.Equal(t, false, res.Requeue)

	database = &doopv1alpha1.Database{}
	err = r.client.Get(context.TODO(), req.NamespacedName, database)
	require.NoError(t, err)
	require.Equal(t, doopv1alpha1.DatabaseStatus{
		ID:                "1",
		Name:              "foo",
		MaintenanceWindow: &doopv1alpha1.DatabaseMaintenanceWindow{},
		Status:            databaseStatusOnline,
	}, database.Status)

	// Delete the object and ensure the resource is deleted from DO.
	mockDatabasesService.EXPECT().Delete(gomock.Any(), fakeDatabase.ID).Return(nil, nil).Times(1)

	database.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	// TODO: Use r.client.Delete here instead of Update.
	err = r.client.Update(context.TODO(), database)
	require.NoError(t, err)

	res, err = r.Reconcile(req)
	require.NoError(t, err)
	require.Equal(t, false, res.Requeue)
}
