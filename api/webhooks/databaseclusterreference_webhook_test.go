package webhooks

import (
	"github.com/digitalocean/do-operator/api/v1alpha1"
	"github.com/digitalocean/godo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DatabaseClusterReference validating webhook", func() {
	var existingDB *godo.Database
	BeforeEach(func() {
		db, _, err := fakeDatabasesService.Create(ctx, &godo.DatabaseCreateRequest{
			Name: "my-db",
		})
		Expect(err).NotTo(HaveOccurred())
		existingDB = db
	})

	Context("When creating a DatabaseClusterReference", func() {
		It("should reject if the database does not exist", func() {
			dbRef := &v1alpha1.DatabaseClusterReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseClusterReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "missing-uuid",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseClusterReferenceSpec{
					UUID: "does-not-exist",
				},
			}

			err := k8sClient.Create(ctx, dbRef)
			Expect(err).To(HaveOccurred())
		})

		It("should accept if the database does exist", func() {
			dbRef := &v1alpha1.DatabaseClusterReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseClusterReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "existing-db",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseClusterReferenceSpec{
					UUID: existingDB.ID,
				},
			}

			err := k8sClient.Create(ctx, dbRef)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When updating a DatabaseClusterReference", func() {
		It("should reject changes to the uuid", func() {
			dbRef := &v1alpha1.DatabaseClusterReference{
				TypeMeta: metav1.TypeMeta{
					APIVersion: v1alpha1.GroupVersion.String(),
					Kind:       v1alpha1.DatabaseClusterReferenceKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update",
					Namespace: "default",
				},
				Spec: v1alpha1.DatabaseClusterReferenceSpec{
					UUID: existingDB.ID,
				},
			}

			err := k8sClient.Create(ctx, dbRef)
			Expect(err).NotTo(HaveOccurred())

			updatedRef := dbRef.DeepCopy()
			updatedRef.Spec.UUID = "another-uuid"
			err = k8sClient.Patch(ctx, updatedRef, client.MergeFrom(dbRef))
			Expect(err).To(HaveOccurred())
		})
	})
})
