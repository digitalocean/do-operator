package v1alpha1

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("DatabaseCluster validating webhook", func() {
	Context("When creating a DatabaseCluster", func() {
		It("should reject when the validation API returns an error", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "invalid-engine-db",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Engine:   "invalid",
					Name:     "invalid-engine-db",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).To(HaveOccurred())
		})
		// TODO(awg) Test the specific field error handling once the validation
		// API is supported in godo for easier mocking/faking. We shouldn't
		// replicate all the validation logic and response creation in fakegodo.
		It("should accept when the validation API returns success", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "valid-engine-db",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Engine:   "mysql",
					Name:     "valid-engine-db",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("When updating a DatabaseCluster", func() {
		It("should reject changes to the engine", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-engine",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-engine",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.Engine = "redis"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should reject changes to the name", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-name",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-name",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.Name = "another-name"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should reject changes to the version", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-version",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-version",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.Version = "8"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should reject changes to the region", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-region",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-region",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.Region = "dev1"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should reject invalid changes to the numNodes", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-numnodes",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-numnodes",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.NumNodes = 100
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should reject invalid changes to the size", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-size",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-size",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.Size = "db-s-8vcpu-8gb"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).To(HaveOccurred())
		})

		It("should accept valid changes", func() {
			db := &DatabaseCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: GroupVersion.String(),
					Kind:       DatabaseClusterKind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "db-to-update-valid",
					Namespace: "default",
				},
				Spec: DatabaseClusterSpec{
					Name:     "db-to-update-valid",
					Engine:   "mysql",
					Version:  "6",
					NumNodes: 1,
					Size:     "db-s-1vcpu-1gb",
					Region:   "dev0",
				},
			}

			err := k8sClient.Create(ctx, db)
			Expect(err).NotTo(HaveOccurred())

			updatedDB := db.DeepCopy()
			updatedDB.Spec.NumNodes = 2
			updatedDB.Spec.Size = "db-s-2vcpu-2gb"
			err = k8sClient.Patch(ctx, updatedDB, client.MergeFrom(db))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
