/*
Copyright 2026.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EC2InstanceSpec defines the desired state of Ec2Instance
type Ec2InstanceSpec struct {

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html
	InstanceType      string            `json:"instanceType"`
	AMIId             string            `json:"amiId"`
	Region            string            `json:"region"`
	AvailabilityZone  string            `json:"availabilityZone,omitempty"`
	KeyPair           string            `json:"keyPair,omitempty"`
	SecurityGroup     string            `json:"securityGroup,omitempty"`
	Subnet            string            `json:"subnet,omitempty"`
	UserData          string            `json:"userData,omitempty"`
	Tags              map[string]string `json:"tags,omitempty"`
	Storage           StorageConfig     `json:"storage,omitempty"`
	AssociatePublicIP bool              `json:"associatePublicIP,omitempty"`
}

type StorageConfig struct {
	RootVolume        VolumeConfig   `json:"rootVolume,omitempty"`
	AdditionalVolumes []VolumeConfig `json:"additionalVolumes,omitempty"`
}

type VolumeConfig struct {
	Size       int32  `json:"size"`
	Type       string `json:"type,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	Encrypted  bool   `json:"encrypted,omitempty"`
}

// Ec2InstanceStatus defines the observed state of Ec2Instance.
type Ec2InstanceStatus struct {
	InstanceID string `json:"instanceID,omitempty"`
	State      string `json:"state,omitempty"`
	PublicIP   string `json:"publicIP,omitempty"`
	PrivateIP  string `json:"privateIP,omitempty"`
	PublicDNS  string `json:"publicDNS,omitempty"`
	PrivateDNS string `json:"privateDNS,omitempty"`
	LaunchTime string `json:"launchTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Ec2Instance is the Schema for the ec2instances API
type Ec2Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   Ec2InstanceSpec   `json:"spec,omitempty"`
	Status Ec2InstanceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// Ec2InstanceList contains a list of Ec2Instance
type Ec2InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Ec2Instance `json:"items"`
}

type CreatedInstanceInfo struct {
	InstanceID string `json:"instanceID"`
	PublicIP   string `json:"publicIP"`
	PrivateIP  string `json:"privateIP"`
	PublicDNS  string `json:"publicDNS"`
	PrivateDNS string `json:"privateDNS"`
	State      string `json:"state"`
}

func init() {
	SchemeBuilder.Register(&Ec2Instance{}, &Ec2InstanceList{})
}
