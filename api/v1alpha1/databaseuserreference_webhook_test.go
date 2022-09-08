package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DatabaseUserReference validating webhook", func() {
	Context("When creating a DatabaseUserReference", func() {
		It("should reject if the cluster group is invalid", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-group",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: pointer.String("does.not.exist"),
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if the cluster kind is invalid", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-kind",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     "DoesNotExist",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster does not exist", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     "does-not-exist",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference does not exist", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterReferenceKind,
						Name:     "my-cluster-ref",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject for redis databases", func() {
			db := createDatabaseClusterFixture("redis")

			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "redis",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     db.Name,
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject for mongodb databases", func() {
			db := createDatabaseClusterFixture("mongodb")

			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mongodb",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     db.Name,
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster user does not exist", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: "does-not-exist",
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference user does not exist", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterReferenceKind,
						Name:     existingDBRef.Name,
					},
					Username: "does-not-exist",
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should accept an existing DatabaseCluster user", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-cluser-user",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept an existing DatabaseClusterReference user", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-clusterreference-user",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterReferenceKind,
						Name:     existingDBRef.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).NotTo(HaveOccurred())

		})
	})

	Context("When updating a DatabaseUserReference", func() {
		It("should reject changes to the cluster", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-cluster",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).NotTo(HaveOccurred())

			updatedUser := userRef.DeepCopy()
			updatedUser.Spec.Cluster.Name = "other-db"
			err = k8sClient.Patch(ctx, updatedUser, client.MergeFrom(userRef))
			Expect(err).To(HaveOccurred())
		})

		It("should reject changes to the username", func() {
			userRef := &DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-username",
					Namespace: "default",
				},
				Spec: DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &GroupVersion.Group,
						Kind:     DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).NotTo(HaveOccurred())

			updatedUser := userRef.DeepCopy()
			updatedUser.Spec.Username = "different-name"
			err = k8sClient.Patch(ctx, updatedUser, client.MergeFrom(userRef))
			Expect(err).To(HaveOccurred())
		})
	})
})
