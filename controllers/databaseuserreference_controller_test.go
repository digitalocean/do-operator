package controllers

import (
	"github.com/digitalocean/do-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("DatabaseUserReference controller", func() {
	Context("When reconciling a DatabaseUserReference", func() {
		It("should manage secrets for a user when referencing a DatabaseCluster", func() {
			const (
				userName = "db-cluster-user"
			)

			var (
				dbCluster          = mustCreateDatabaseCluster()
				godoUser           = mustCreateGodoDBUser(dbCluster.Status.UUID, userName)
				dbUserRefLookupKey = types.NamespacedName{
					Name:      "db-cluster-user-ref-crd",
					Namespace: "default",
				}
				createdDBUserRef        = &v1alpha1.DatabaseUserReference{}
				dbUserRefOwnerReference metav1.OwnerReference
			)

			By("creating a DatabaseUserReference object referencing a DatabaseCluster", func() {
				dbUserRef := &v1alpha1.DatabaseUserReference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseUserKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db-cluster-user-ref-crd",
						Namespace: "default",
					},
					Spec: v1alpha1.DatabaseUserReferenceSpec{
						Cluster: corev1.TypedLocalObjectReference{
							APIGroup: &v1alpha1.GroupVersion.Group,
							Kind:     v1alpha1.DatabaseClusterKind,
							Name:     dbCluster.Name,
						},
						Username: userName,
					},
				}
				Expect(k8sClient.Create(ctx, dbUserRef)).To(Succeed())
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbUserRefOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseUserReferenceKind,
					Name:               "db-cluster-user-ref-crd",
					UID:                createdDBUserRef.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseUserReference status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)).To(Succeed())
					g.Expect(createdDBUserRef.Status.Role).To(Equal(godoUser.Role))
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "db-cluster-user-ref-crd-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbUserRefOwnerReference))
				Expect(string(secret.Data["username"])).To(Equal(userName))
				Expect(secret.Data["password"]).NotTo(BeEmpty())
			})

			By("deleting the DatabaseUserReference object", func() {
				Expect(k8sClient.Delete(ctx, createdDBUserRef)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the user is not deleted in godo", func() {
				Consistently(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbCluster.Status.UUID, createdDBUserRef.Spec.Username)
					return err
				}, timeout, interval).Should(Succeed())
			})
		})

		It("should manage secrets for a user when referencing a DatabaseClusterReference", func() {
			const (
				userName = "db-cluster-user"
			)

			var (
				dbRef              = mustCreateDatabaseClusterReference()
				godoUser           = mustCreateGodoDBUser(dbRef.Spec.UUID, userName)
				dbUserRefLookupKey = types.NamespacedName{
					Name:      "db-ref-user-ref-crd",
					Namespace: "default",
				}
				createdDBUserRef        = &v1alpha1.DatabaseUserReference{}
				dbUserRefOwnerReference metav1.OwnerReference
			)

			By("creating a DatabaseUserReference object referencing a DatabaseClusterReference", func() {
				dbUserRef := &v1alpha1.DatabaseUserReference{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseUserKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db-ref-user-ref-crd",
						Namespace: "default",
					},
					Spec: v1alpha1.DatabaseUserReferenceSpec{
						Cluster: corev1.TypedLocalObjectReference{
							APIGroup: &v1alpha1.GroupVersion.Group,
							Kind:     v1alpha1.DatabaseClusterReferenceKind,
							Name:     dbRef.Name,
						},
						Username: userName,
					},
				}
				Expect(k8sClient.Create(ctx, dbUserRef)).To(Succeed())
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbUserRefOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseUserReferenceKind,
					Name:               "db-ref-user-ref-crd",
					UID:                createdDBUserRef.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseUserReference status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)).To(Succeed())
					g.Expect(createdDBUserRef.Status.Role).To(Equal(godoUser.Role))
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "db-ref-user-ref-crd-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbUserRefOwnerReference))
				Expect(string(secret.Data["username"])).To(Equal(userName))
				Expect(secret.Data["password"]).NotTo(BeEmpty())
			})

			By("deleting the DatabaseUserReference object", func() {
				Expect(k8sClient.Delete(ctx, createdDBUserRef)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbUserRefLookupKey, createdDBUserRef)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the user is not deleted in godo", func() {
				Consistently(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbRef.Spec.UUID, createdDBUserRef.Spec.Username)
					return err
				}, timeout, interval).Should(Succeed())
			})
		})
	})
})
