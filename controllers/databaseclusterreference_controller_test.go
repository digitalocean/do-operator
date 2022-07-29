package controllers

import (
	"time"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/do-operator/fakegodo"
	"github.com/digitalocean/godo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("DatabaseClusterReference controller", func() {
	Context("When reconciling a DatabaseClusterReference", func() {
		const (
			dbEngine   = "mysql"
			dbName     = "sql-is-cool"
			dbVersion  = "8"
			dbNumNodes = 3
			dbSize     = "sql-size-slug"
			dbRegion   = "dev2"
		)

		// Make the controller refresh status every 30 seconds so we're
		// guaranteed to see updates reflected within our 1-minute timeout.
		clusterReferenceRefreshTime = 30 * time.Second

		It("should manage ConfigMaps and Secrets for the database", func() {
			var (
				dbRefLookupKey = types.NamespacedName{
					Name:      "dbref-crd",
					Namespace: "default",
				}
				createdDBRef        = &v1alpha1.DatabaseClusterReference{}
				dbRefOwnerReference metav1.OwnerReference
				dbUUID              string
			)

			By("creating the database in godo", func() {
				db, _, err := fakeDatabasesService.Create(ctx, &godo.DatabaseCreateRequest{
					Name:       dbName,
					EngineSlug: dbEngine,
					Version:    dbVersion,
					SizeSlug:   dbSize,
					Region:     dbRegion,
					NumNodes:   dbNumNodes,
				})
				Expect(err).NotTo(HaveOccurred())
				// Save the UUID for future use.
				dbUUID = db.ID
			})

			By("creating the DatabsaeClusterReference object", func() {
				dbRef := &v1alpha1.DatabaseClusterReference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseClusterReferenceKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "dbref-crd",
						Namespace: "default",
					},
					Spec: v1alpha1.DatabaseClusterReferenceSpec{
						UUID: dbUUID,
					},
				}
				Expect(k8sClient.Create(ctx, dbRef)).To(Succeed())
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbRefLookupKey, createdDBRef)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbRefOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseClusterReferenceKind,
					Name:               "dbref-crd",
					UID:                createdDBRef.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseClusterReference status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbRefLookupKey, createdDBRef)).To(Succeed())
					g.Expect(createdDBRef.Status.Engine).To(Equal(dbEngine))
					g.Expect(createdDBRef.Status.Name).To(Equal(dbName))
					g.Expect(createdDBRef.Status.NumNodes).To(Equal(int64(dbNumNodes)))
					g.Expect(createdDBRef.Status.Size).To(Equal(dbSize))
					g.Expect(createdDBRef.Status.Region).To(Equal(dbRegion))
					g.Expect(createdDBRef.Status.Version).To(Equal(dbVersion))
					g.Expect(createdDBRef.Status.Status).To(Equal(fakegodo.OnlineStatus))
					g.Expect(createdDBRef.Status.CreatedAt.IsZero()).NotTo(BeTrue())
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the connection ConfigMap is created", func() {
				configMapKey := types.NamespacedName{
					Name:      "dbref-crd-connection",
					Namespace: "default",
				}
				cm := &corev1.ConfigMap{}
				Eventually(func() error {
					return k8sClient.Get(ctx, configMapKey, cm)
				}, timeout, interval).Should(Succeed())
				Expect(cm.OwnerReferences).To(ContainElement(dbRefOwnerReference))
				Expect(cm.Data["host"]).NotTo(BeEmpty())
				Expect(cm.Data["port"]).NotTo(BeEmpty())
				Expect(cm.Data["ssl"]).NotTo(BeEmpty())
				Expect(cm.Data["database"]).NotTo(BeEmpty())
			})

			By("ensuring the default credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "dbref-crd-default-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbRefOwnerReference))
				Expect(secret.Data["uri"]).NotTo(BeEmpty())
				Expect(secret.Data["username"]).NotTo(BeEmpty())
				Expect(secret.Data["password"]).NotTo(BeEmpty())
			})

			By("modifying the size of the database in godo", func() {
				_, err := fakeDatabasesService.Resize(ctx, dbUUID, &godo.DatabaseResizeRequest{
					SizeSlug: "extremely-large",
					NumNodes: 1000,
				})
				Expect(err).NotTo(HaveOccurred())
			})

			By("ensuring the DatabaseClusterReference status gets refreshed", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbRefLookupKey, createdDBRef)).To(Succeed())
					g.Expect(createdDBRef.Status.Size).To(Equal("extremely-large"))
					g.Expect(createdDBRef.Status.NumNodes).To(Equal(int64(1000)))
				}, timeout, interval).Should(Succeed())
			})

			By("deleting the DatabaseClusterReference object", func() {
				Expect(k8sClient.Delete(ctx, createdDBRef)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbRefLookupKey, createdDBRef)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the database is not deleted in godo", func() {
				Consistently(func() error {
					_, _, err := fakeDatabasesService.Get(ctx, dbUUID)
					return err
				}, timeout, interval).Should(Succeed())
			})
		})
	})
})
