package controllers

import (
	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/do-operator/fakegodo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DatabaseCluster controller", func() {
	Context("When reconciling a DatabaseCluster", func() {
		const (
			dbEngine   = "mongodb"
			dbName     = "dev-null"
			dbVersion  = "5.0"
			dbNumNodes = 1
			dbSize     = "size-slug"
			dbRegion   = "dev1"
		)

		It("should manage the lifecycle of a database", func() {
			var (
				dbClusterLookupKey = types.NamespacedName{
					Name:      "db-crd",
					Namespace: "default",
				}
				createdDBCluster        = &v1alpha1.DatabaseCluster{}
				dbClusterOwnerReference metav1.OwnerReference
			)

			By("creating the DatabaseCluster object", func() {
				dbCluster := &v1alpha1.DatabaseCluster{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseClusterKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db-crd",
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
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbClusterOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseClusterKind,
					Name:               "db-crd",
					UID:                createdDBCluster.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseCluster status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster)).To(Succeed())
					g.Expect(createdDBCluster.Status.UUID).NotTo(BeEmpty())
					// The status should be "creating" at first and then move to
					// "online", but we might not catch it before the status
					// changes, so accept either.
					g.Expect(createdDBCluster.Status.Status).
						To(BeElementOf(fakegodo.CreatingStatus, fakegodo.OnlineStatus))
					g.Expect(createdDBCluster.Status.CreatedAt.IsZero()).NotTo(BeTrue())
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the database exists in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.Get(ctx, createdDBCluster.Status.UUID)
					return err
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the connection ConfigMap is created", func() {
				configMapKey := types.NamespacedName{
					Name:      "db-crd-connection",
					Namespace: "default",
				}
				cm := &corev1.ConfigMap{}
				Eventually(func() error {
					return k8sClient.Get(ctx, configMapKey, cm)
				}, timeout, interval).Should(Succeed())
				Expect(cm.OwnerReferences).To(ContainElement(dbClusterOwnerReference))
				Expect(cm.Data["host"]).NotTo(BeEmpty())
				Expect(cm.Data["port"]).NotTo(BeEmpty())
				Expect(cm.Data["ssl"]).NotTo(BeEmpty())
				Expect(cm.Data["database"]).NotTo(BeEmpty())
			})

			By("ensuring the default credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "db-crd-default-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbClusterOwnerReference))
				Expect(secret.Data["uri"]).NotTo(BeEmpty())
				Expect(secret.Data["username"]).NotTo(BeEmpty())
				Expect(secret.Data["password"]).NotTo(BeEmpty())
			})

			By("ensuring the DatabaseCluster status gets refreshed later", func() {
				Eventually(func() (string, error) {
					if err := k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster); err != nil {
						return "", err
					}
					return createdDBCluster.Status.Status, nil
				}, timeout, interval).Should(Equal(fakegodo.OnlineStatus))
			})

			By("modifying the size configuration in the DatabaseCluster object", func() {
				updatedDBCluster := createdDBCluster.DeepCopy()
				updatedDBCluster.Spec.NumNodes = 100
				updatedDBCluster.Spec.Size = "much-bigger-size"

				Expect(k8sClient.Patch(ctx, updatedDBCluster, client.MergeFrom(createdDBCluster))).To(Succeed())
				// Update our in-memory copy.
				Expect(k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster)).To(Succeed())
			})

			By("ensuring the database was updated in godo", func() {
				Eventually(func() (bool, error) {
					db, _, err := fakeDatabasesService.Get(ctx, createdDBCluster.Status.UUID)
					if err != nil {
						return false, err
					}

					return db.SizeSlug == "much-bigger-size" && db.NumNodes == 100, nil
				}, timeout, interval).Should(BeTrue())
			})

			By("deleting the DatabaseCluster object", func() {
				Expect(k8sClient.Delete(ctx, createdDBCluster)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbClusterLookupKey, createdDBCluster)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the database is deleted in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.Get(ctx, createdDBCluster.Status.UUID)
					return err
				}, timeout, interval).ShouldNot(Succeed())
			})
		})
	})
})
