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
	"github.com/digitalocean/do-operator/extgodo"
	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DatabaseClusterSpec defines the desired state of DatabaseCluster
type DatabaseClusterSpec struct {
	// Engine is the database engine to use.
	Engine string `json:"engine"`
	// Name is the name of the database cluster.
	Name string `json:"name"`
	// Version is the DB version to use.
	Version string `json:"version"`
	// NumNodes is the number of nodes in the database cluster.
	NumNodes int64 `json:"numNodes"`
	// Size is the slug of the node size to use.
	Size string `json:"size"`
	// Region is the slug of the DO region for the cluster.
	Region string `json:"region"`
}

// ToGodoCreateRequest returns a create request for a database that will fulfill
// the DatabaseClusterSpec.
func (spec *DatabaseClusterSpec) ToGodoCreateRequest() *godo.DatabaseCreateRequest {
	return &godo.DatabaseCreateRequest{
		EngineSlug: spec.Engine,
		Name:       spec.Name,
		Version:    spec.Version,
		SizeSlug:   spec.Size,
		Region:     spec.Region,
		NumNodes:   int(spec.NumNodes),
	}
}

// ToGodoValidateCreateRequest returns a validation request for a database that
// will fulfill the DatabaseClusterSpec.
func (spec *DatabaseClusterSpec) ToGodoValidateCreateRequest() *extgodo.DatabaseValidateCreateRequest {
	createReq := spec.ToGodoCreateRequest()
	return &extgodo.DatabaseValidateCreateRequest{
		DatabaseCreateRequest: *createReq,
		DryRun:                true,
	}
}

// DatabaseClusterStatus defines the observed state of DatabaseCluster
type DatabaseClusterStatus struct {
	// UUID is the UUID of the database cluster.
	UUID string `json:"uuid,omitempty"`
	// Status is the status of the database cluster.
	Status string `json:"status,omitempty"`
	// CreatedAt is the time at which the database cluster was created.
	CreatedAt metav1.Time `json:"createdAt,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// DatabaseCluster is the Schema for the databaseclusters API
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Cluster name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.status`
type DatabaseCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseClusterSpec   `json:"spec,omitempty"`
	Status DatabaseClusterStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// DatabaseClusterList contains a list of DatabaseCluster
type DatabaseClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DatabaseCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DatabaseCluster{}, &DatabaseClusterList{})
}
