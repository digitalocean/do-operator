package v1alpha1

import (
	"github.com/digitalocean/godo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DatabaseSpec defines the desired state of Database
// +k8s:openapi-gen=true
// https://github.com/digitalocean/godo/blob/master/databases.go#L125
type DatabaseSpec struct {
	Name               string   `json:"name,omitempty"`
	EngineSlug         string   `json:"engine,omitempty"`
	Version            string   `json:"version,omitempty"`
	SizeSlug           string   `json:"size,omitempty"`
	Region             string   `json:"region,omitempty"`
	NumNodes           int      `json:"num_nodes,omitempty"`
	PrivateNetworkUUID string   `json:"private_network_uuid"`
	Tags               []string `json:"tags,omitempty"`
}

func (s *DatabaseSpec) ToDO() *godo.DatabaseCreateRequest {
	return &godo.DatabaseCreateRequest{
		Name:               s.Name,
		EngineSlug:         s.EngineSlug,
		Version:            s.Version,
		SizeSlug:           s.SizeSlug,
		Region:             s.Region,
		NumNodes:           s.NumNodes,
		PrivateNetworkUUID: s.PrivateNetworkUUID,
		Tags:               s.Tags,
	}
}

// DatabaseStatus defines the observed state of Database
// +k8s:openapi-gen=true
// https://github.com/digitalocean/godo/blob/master/databases.go#L68
type DatabaseStatus struct {
	ID                 string                     `json:"id,omitempty"`
	Name               string                     `json:"name,omitempty"`
	EngineSlug         string                     `json:"engine,omitempty"`
	VersionSlug        string                     `json:"version,omitempty"`
	Connection         *DatabaseConnection        `json:"connection,omitempty"`
	PrivateConnection  *DatabaseConnection        `json:"private_connection,omitempty"`
	Users              []DatabaseUser             `json:"users,omitempty"`
	NumNodes           int                        `json:"num_nodes,omitempty"`
	SizeSlug           string                     `json:"size,omitempty"`
	DBNames            []string                   `json:"db_names,omitempty"`
	RegionSlug         string                     `json:"region,omitempty"`
	Status             string                     `json:"status,omitempty"`
	MaintenanceWindow  *DatabaseMaintenanceWindow `json:"maintenance_window,omitempty"`
	CreatedAt          *metav1.Time               `json:"created_at,omitempty"`
	PrivateNetworkUUID string                     `json:"private_network_uuid,omitempty"`
	Tags               []string                   `json:"tags,omitempty"`
}

// DatabaseConnection represents a database connection
// +k8s:openapi-gen=true
// https://github.com/digitalocean/godo/blob/master/databases.go#L92
type DatabaseConnection struct {
	URI      string `json:"uri,omitempty"`
	Database string `json:"database,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	SSL      bool   `json:"ssl,omitempty"`
}

// DatabaseUser represents a user in the database
// +k8s:openapi-gen=true
// https://github.com/digitalocean/godo/blob/master/databases.go#L103
type DatabaseUser struct {
	Name     string `json:"name,omitempty"`
	Role     string `json:"role,omitempty"`
	Password string `json:"password,omitempty"`
}

// DatabaseMaintenanceWindow represents the maintenance_window of a database
// cluster
// +k8s:openapi-gen=true
// https://github.com/digitalocean/godo/blob/master/databases.go#L110
type DatabaseMaintenanceWindow struct {
	Day         string   `json:"day,omitempty"`
	Hour        string   `json:"hour,omitempty"`
	Pending     bool     `json:"pending,omitempty"`
	Description []string `json:"description,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Database is the Schema for the databases API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}
