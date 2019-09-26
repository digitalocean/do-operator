package database

import (
	"testing"

	doopv1alpha1 "github.com/snormore/dodb-operator/pkg/apis/doop/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// https://github.com/operator-framework/operator-sdk/blob/master/doc/user/unit-testing.md

func TestDatabaseControllerCreate(t *testing.T) {
	// Register operator types with the runtime scheme.
	s := scheme.Scheme
	s.AddKnownTypes(doopv1alpha1.SchemeGroupVersion, &doopv1alpha1.Database{})
	s.AddKnownTypes(doopv1alpha1.SchemeGroupVersion, &doopv1alpha1.DatabaseList{})

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

	r := &ReconcileDatabase{
		client: cl,
		scheme: s,
	}

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      example.Name,
			Namespace: example.Namespace,
		},
	}

	res, err := r.Reconcile(req)
	require.NoError(t, err)
	require.Equal(t, false, res.Requeue)
}
