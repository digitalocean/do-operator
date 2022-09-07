/*
Copyright 2022 DigitalOcean.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseUserReferenceSpec defines the desired state of DatabaseUserReference
type DatabaseUserReferenceSpec struct {
	// Cluster is a reference to the DatabaseCluster or DatabaseClusterReference
	// that represents the database cluster in which the user exists.
	Cluster corev1.TypedLocalObjectReference `json:"databaseCluster"`
	// Username is the username of the referenced user.
	Username string `json:"username"`
}

// DatabaseUserReferenceStatus defines the observed state of DatabaseUserReference
type DatabaseUserReferenceStatus struct {
	// ClusterUUID is the UUID of the cluster this user is in. We keep this in
	// the status so that we can reference the user even if the referenced
	// Cluster CR is deleted.
	ClusterUUID string `json:"clusterUUID,omitempty"`
	// Role is the user's role.
	Role string `json:"role,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DatabaseUserReference is the Schema for the databaseuserreferences API
type DatabaseUserReference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseUserReferenceSpec   `json:"spec,omitempty"`
	Status DatabaseUserReferenceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseUserReferenceList contains a list of DatabaseUserReference
type DatabaseUserReferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseUserReference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseUserReference{}, &DatabaseUserReferenceList{})
}
