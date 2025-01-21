package webhooks

import (
	"github.com/digitalocean/do-operator/api/v1alpha1"
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
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-group",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: pointer.String("does.not.exist"),
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if the cluster kind is invalid", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-kind",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     "DoesNotExist",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster does not exist", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     "does-not-exist",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference does not exist", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
						Name:     "my-cluster-ref",
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject for redis databases", func() {
			db := createDatabaseClusterFixture("redis")

			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "redis",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     db.Name,
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject for mongodb databases", func() {
			db := createDatabaseClusterFixture("mongodb")

			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "mongodb",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     db.Name,
					},
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster user does not exist", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: "does-not-exist",
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference user does not exist", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
						Name:     existingDBRef.Name,
					},
					Username: "does-not-exist",
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).To(HaveOccurred())
		})

		It("should accept an existing DatabaseCluster user", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-cluser-user",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, userRef)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept an existing DatabaseClusterReference user", func() {
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-clusterreference-user",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
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
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-cluster",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
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
			userRef := &v1alpha1.DatabaseUserReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-username",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserReferenceSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
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
