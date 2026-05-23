package controllers

import (
	"fmt"

	"github.com/digitalocean/do-operator/api/v1alpha1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
)

var _ = Describe("DatabaseUser controller", func() {
	Context("When reconciling a DatabaseUser", func() {
		It("should manage the lifecycle of a database user referencing a DatabaseCluster", func() {
			const (
				userName = "db-cluster-user"
			)

			var (
				dbCluster       = mustCreateDatabaseCluster()
				dbUserLookupKey = types.NamespacedName{
					Name:      "db-cluster-user-crd",
					Namespace: "default",
				}
				createdDBUser        = &v1alpha1.DatabaseUser{}
				dbUserOwnerReference metav1.OwnerReference
			)

			By("creating a DatabaseUser object referencing a DatabaseCluster", func() {
				dbUser := &v1alpha1.DatabaseUser{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseUserKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db-cluster-user-crd",
						Namespace: "default",
					},
					Spec: v1alpha1.DatabaseUserSpec{
						Cluster: corev1.TypedLocalObjectReference{
							APIGroup: &v1alpha1.GroupVersion.Group,
							Kind:     v1alpha1.DatabaseClusterKind,
							Name:     dbCluster.Name,
						},
						Username: userName,
					},
				}
				Expect(k8sClient.Create(ctx, dbUser)).To(Succeed())
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbUserOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseUserKind,
					Name:               "db-cluster-user-crd",
					UID:                createdDBUser.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseUser status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)).To(Succeed())
					g.Expect(createdDBUser.Status.Role).NotTo(BeEmpty())
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the user exists in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbCluster.Status.UUID, createdDBUser.Spec.Username)
					return err
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "db-cluster-user-crd-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbUserOwnerReference))
				Expect(string(secret.Data["username"])).To(Equal(userName))
				Expect(secret.Data["password"]).NotTo(BeEmpty())
				Expect(string(secret.Data["uri"])).To(Equal(fmt.Sprintf("postgresql://%s:%s@host:12345/database?sslmode=require", secret.Data["username"], secret.Data["password"])))
				Expect(string(secret.Data["private_uri"])).To(Equal(fmt.Sprintf("postgresql://%s:%s@private-host:12345/private-database?sslmode=require", secret.Data["username"], secret.Data["password"])))
			})

			By("deleting the DatabaseUser object", func() {
				Expect(k8sClient.Delete(ctx, createdDBUser)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the user is deleted in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbCluster.Status.UUID, createdDBUser.Spec.Username)
					return err
				}, timeout, interval).ShouldNot(Succeed())
			})
		})

		It("should manage the lifecycle of a database user referencing a DatabaseClusterReference", func() {
			const (
				userName = "db-cluster-user"
			)

			var (
				dbRef           = mustCreateDatabaseClusterReference()
				dbUserLookupKey = types.NamespacedName{
					Name:      "db-ref-user-crd",
					Namespace: "default",
				}
				createdDBUser        = &v1alpha1.DatabaseUser{}
				dbUserOwnerReference metav1.OwnerReference
			)

			By("creating a DatabaseUser object referencing a DatabaseCluster", func() {
				dbUser := &v1alpha1.DatabaseUser{
					TypeMeta: metav1.TypeMeta{
						APIVersion: v1alpha1.GroupVersion.String(),
						Kind:       v1alpha1.DatabaseUserKind,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db-ref-user-crd",
						Namespace: "default",
					},
					Spec: v1alpha1.DatabaseUserSpec{
						Cluster: corev1.TypedLocalObjectReference{
							APIGroup: &v1alpha1.GroupVersion.Group,
							Kind:     v1alpha1.DatabaseClusterReferenceKind,
							Name:     dbRef.Name,
						},
						Username: userName,
					},
				}
				Expect(k8sClient.Create(ctx, dbUser)).To(Succeed())
				// Make sure we can read it back.
				Eventually(func() error {
					return k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)
				}, timeout, interval).Should(Succeed())
				// Construct the OwnerReference for later use.
				dbUserOwnerReference = metav1.OwnerReference{
					APIVersion:         v1alpha1.GroupVersion.String(),
					Kind:               v1alpha1.DatabaseUserKind,
					Name:               "db-ref-user-crd",
					UID:                createdDBUser.UID,
					Controller:         pointer.Bool(true),
					BlockOwnerDeletion: pointer.Bool(true),
				}
			})

			By("ensuring the DatabaseUser status gets filled in", func() {
				Eventually(func(g Gomega) {
					g.Expect(k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)).To(Succeed())
					g.Expect(createdDBUser.Status.Role).NotTo(BeEmpty())
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the user exists in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbRef.Spec.UUID, createdDBUser.Spec.Username)
					return err
				}, timeout, interval).Should(Succeed())
			})

			By("ensuring the credentials Secret is created", func() {
				secretKey := types.NamespacedName{
					Name:      "db-ref-user-crd-credentials",
					Namespace: "default",
				}
				secret := &corev1.Secret{}
				Eventually(func() error {
					return k8sClient.Get(ctx, secretKey, secret)
				}, timeout, interval).Should(Succeed())
				Expect(secret.OwnerReferences).To(ContainElement(dbUserOwnerReference))
				Expect(string(secret.Data["username"])).To(Equal(userName))
				Expect(secret.Data["password"]).NotTo(BeEmpty())
				Expect(string(secret.Data["uri"])).To(Equal(fmt.Sprintf("postgresql://%s:%s@host:12345/database?sslmode=require", secret.Data["username"], secret.Data["password"])))
				Expect(string(secret.Data["private_uri"])).To(Equal(fmt.Sprintf("postgresql://%s:%s@private-host:12345/private-database?sslmode=require", secret.Data["username"], secret.Data["password"])))
			})

			By("deleting the DatabaseUser object", func() {
				Expect(k8sClient.Delete(ctx, createdDBUser)).To(Succeed())
				// Wait for the object to go away.
				Eventually(func() bool {
					err := k8sClient.Get(ctx, dbUserLookupKey, createdDBUser)
					return kerrors.IsNotFound(err)
				}, timeout, interval).Should(BeTrue())
			})

			By("ensuring the user is deleted in godo", func() {
				Eventually(func() error {
					_, _, err := fakeDatabasesService.GetUser(ctx, dbRef.Spec.UUID, createdDBUser.Spec.Username)
					return err
				}, timeout, interval).ShouldNot(Succeed())
			})
		})
	})
})
