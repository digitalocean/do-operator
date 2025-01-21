package webhooks

import (
	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	existingDB       *v1alpha1.DatabaseCluster
	existingDBRef    *v1alpha1.DatabaseClusterReference
	existingUsername = "existing-user"
)

func createDatabaseClusterFixture(engine string) *v1alpha1.DatabaseCluster {
	name := "my-" + engine + "-db"

	db := &v1alpha1.DatabaseCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.DatabaseClusterKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.DatabaseClusterSpec{
			Engine:   engine,
			Name:     name,
			Version:  "8",
			NumNodes: 1,
			Size:     "size-slug",
			Region:   "dev1",
		},
	}
	Expect(k8sClient.Create(ctx, db)).To(Succeed())

	// We're not running the controller in these tests, so fake it out.
	godoDB, _, err := fakeDatabasesService.Create(ctx, &godo.DatabaseCreateRequest{
		Name:       name,
		EngineSlug: engine,
	})
	Expect(err).NotTo(HaveOccurred())
	db.Status = v1alpha1.DatabaseClusterStatus{
		UUID:      godoDB.ID,
		Status:    godoDB.Status,
		CreatedAt: metav1.NewTime(godoDB.CreatedAt),
	}
	Expect(k8sClient.Status().Update(ctx, db)).To(Succeed())

	return db
}

func createUserWebhookTestFixtures() {
	existingDB = createDatabaseClusterFixture("mysql")

	dbRef := &v1alpha1.DatabaseClusterReference{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.GroupVersion.String(),
			Kind:       v1alpha1.DatabaseClusterReferenceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-db-ref",
			Namespace: "default",
		},
		Spec: v1alpha1.DatabaseClusterReferenceSpec{
			UUID: existingDB.Status.UUID,
		},
	}
	Expect(k8sClient.Create(ctx, dbRef)).To(Succeed())
	dbRef.Status.Engine = "mysql"
	Expect(k8sClient.Status().Update(ctx, dbRef)).To(Succeed())
	existingDBRef = dbRef

	// Create a user in the DB to test duplicate names.
	_, _, err := fakeDatabasesService.CreateUser(ctx, existingDB.Status.UUID, &godo.DatabaseCreateUserRequest{
		Name: existingUsername,
	})
	Expect(err).NotTo(HaveOccurred())
}

var _ = Describe("DatabaseUser validating webhook", func() {
	Context("When creating a DatabaseUser", func() {
		It("should reject if the cluster group is invalid", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-group",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: pointer.String("does.not.exist"),
					},
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if the cluster kind is invalid", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-kind",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     "DoesNotExist",
					},
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster does not exist", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     "does-not-exist",
					},
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference does not exist", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "does-not-exist",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
						Name:     "my-cluster-ref",
					},
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseCluster user already exists", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should reject if a DatabaseClusterReference user already exists", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "already-exists",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
						Name:     existingDBRef.Name,
					},
					Username: existingUsername,
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).To(HaveOccurred())
		})

		It("should accept a new DatabaseCluster user", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-cluser-user",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: "new-user",
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should accept a new DatabaseClusterReference user", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "new-clusterreference-user",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterReferenceKind,
						Name:     existingDBRef.Name,
					},
					Username: "new-user",
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When updating a DatabaseUser", func() {
		It("should reject changes to the cluster", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-cluster",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: "user-to-update-cluster",
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).NotTo(HaveOccurred())

			updatedUser := dbUser.DeepCopy()
			updatedUser.Spec.Cluster.Name = "other-db"
			err = k8sClient.Patch(ctx, updatedUser, client.MergeFrom(dbUser))
			Expect(err).To(HaveOccurred())
		})

		It("should reject changes to the username", func() {
			dbUser := &v1alpha1.DatabaseUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseUserKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "user-to-update-username",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseUserSpec{
					Cluster: corev1.TypedLocalObjectReference{
						APIGroup: &v1alpha1.GroupVersion.Group,
						Kind:     v1alpha1.DatabaseClusterKind,
						Name:     existingDB.Name,
					},
					Username: "user-to-update-username",
				},
			}

			err := k8sClient.Create(ctx, dbUser)
			Expect(err).NotTo(HaveOccurred())

			updatedUser := dbUser.DeepCopy()
			updatedUser.Spec.Username = "different-name"
			err = k8sClient.Patch(ctx, updatedUser, client.MergeFrom(dbUser))
			Expect(err).To(HaveOccurred())
		})
	})
})
