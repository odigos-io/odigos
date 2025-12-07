/*
Copyright 2022.

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

// NodeDetailsSpec defines the desired state of NodeDetails
type NodeDetailsSpec struct {
	// WaspEnabled indicates whether wasp is enabled on this node
	WaspEnabled bool `json:"waspEnabled"`

	// KernelVersion is the version of the kernel running on the node
	KernelVersion string `json:"kernelVersion"`

	// CPUCapacity is the CPU capacity of the node (number of cores)
	CPUCapacity int `json:"cpuCapacity"`

	// MemoryCapacity is the memory capacity of the node in MB (megabytes)
	MemoryCapacity int `json:"memoryCapacity"`

	// DiscoveryOdigletPodName is the name of the odiglet pod that discovered this node
	DiscoveryOdigletPodName string `json:"discoveryOdigletPodName"`
}

// NodeDetailsStatus defines the observed state of NodeDetails
type NodeDetailsStatus struct {
}

//+genclient
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=nodedetailses,singular=nodedetails
//+kubebuilder:metadata:labels=odigos.io/system-object=true

// NodeDetails is the Schema for the nodedetails API
type NodeDetails struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeDetailsSpec   `json:"spec,omitempty"`
	Status NodeDetailsStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeDetailsList contains a list of NodeDetails
type NodeDetailsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeDetails `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeDetails{}, &NodeDetailsList{})
}
