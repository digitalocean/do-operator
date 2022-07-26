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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseClusterReferenceSpec defines the desired state of DatabaseClusterReference
type DatabaseClusterReferenceSpec struct {
	// UUID is the UUID of an existing database.
	UUID string `json:"uuid"`
}

// DatabaseClusterReferenceStatus defines the observed state of DatabaseClusterReference
type DatabaseClusterReferenceStatus struct {
	// Engine is the database engine to use.
	Engine string `json:"engine,omitempty"`
	// Name is the name of the database cluster.
	Name string `json:"name,omitempty"`
	// Version is the DB version to use.
	Version string `json:"version,omitempty"`
	// NumNodes is the number of nodes in the database cluster.
	NumNodes int64 `json:"numNodes,omitempty"`
	// Size is the slug of the node size to use.
	Size string `json:"size,omitempty"`
	// Region is the slug of the DO region for the cluster.
	Region string `json:"region,omitempty"`
	// Status is the status of the database cluster.
	Status string `json:"status,omitempty"`
	// CreatedAt is the time at which the database cluster was created.
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DatabaseClusterReference is the Schema for the databaseclusterreferences API
type DatabaseClusterReference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseClusterReferenceSpec   `json:"spec,omitempty"`
	Status DatabaseClusterReferenceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseClusterReferenceList contains a list of DatabaseClusterReference
type DatabaseClusterReferenceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseClusterReference `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseClusterReference{}, &DatabaseClusterReferenceList{})
}
