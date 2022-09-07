package controllers

import (
	"context"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
)

func mustCreateDatabaseCluster() *v1alpha1.DatabaseCluster {
	const (
		dbEngine   = "mongodb"
		dbVersion  = "5.0"
		dbNumNodes = 1
		dbSize     = "size-slug"
		dbRegion   = "dev1"
	)
	var (
		dbName             = rand.String(16)
		dbClusterLookupKey = types.NamespacedName{
			Name:      dbName,
			Namespace: "default",
		}
		createdDBCluster = &v1alpha1.DatabaseCluster{}
	)

	dbCluster := &v1alpha1.DatabaseCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.DatabaseClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbName,
			Namespace: "default",
		},
		Spec: v1alpha1.DatabaseClusterSpec{
			Engine:   dbEngine,
			Name:     dbName,
			Version:  dbVersion,
			NumNodes: dbNumNodes,
			Size:     dbSize,
			Region:   dbRegion,
		},
	}
	Expect(k8sClient.Create(ctx, dbCluster)).To(Succeed())
	// Wait for the DB to be created.
	Eventually(func(g Gomega) {
		g.Expect(k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster)).To(Succeed())
		g.Expect(createdDBCluster.Status.UUID).NotTo(BeEmpty())
	}, timeout, interval).Should(Succeed())

	return createdDBCluster
}

func mustCreateDatabaseClusterReference() *v1alpha1.DatabaseClusterReference {
	const (
		dbEngine   = "mongodb"
		dbVersion  = "5.0"
		dbNumNodes = 1
		dbSize     = "size-slug"
		dbRegion   = "dev1"
	)
	var (
		dbName         = rand.String(16)
		dbRefLookupKey = types.NamespacedName{
			Name:      dbName,
			Namespace: "default",
		}
		createdDBRef = &v1alpha1.DatabaseClusterReference{}
	)

	godoDB, _, err := fakeDatabasesService.Create(context.Background(), &godo.DatabaseCreateRequest{
		Name:       dbName,
		EngineSlug: dbEngine,
		Version:    dbVersion,
		SizeSlug:   dbSize,
		Region:     dbRegion,
		NumNodes:   dbNumNodes,
	})
	Expect(err).NotTo(HaveOccurred())

	dbRef := &v1alpha1.DatabaseClusterReference{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.DatabaseClusterReferenceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dbName,
			Namespace: "default",
		},
		Spec: v1alpha1.DatabaseClusterReferenceSpec{
			UUID: godoDB.ID,
		},
	}
	Expect(k8sClient.Create(ctx, dbRef)).To(Succeed())
	// Make sure we can read it back.
	Eventually(func() error {
		return k8sClient.Get(ctx, dbRefLookupKey, createdDBRef)
	}, timeout, interval).Should(Succeed())

	return createdDBRef
}

func mustCreateGodoDBUser(clusterUUID string, username string) *godo.DatabaseUser {
	user, _, err := fakeDatabasesService.CreateUser(context.Background(), clusterUUID, &godo.DatabaseCreateUserRequest{
		Name: username,
	})
	Expect(err).NotTo(HaveOccurred())
	return user
}
