/*
Copyright 2025.

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
	"github.com/odigos-io/odigos/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OdigosSpec defines the desired state of Odigos
type OdigosSpec struct {
	// Version is the version of Odigos to install
	Version string `json:"version"`

	// TelemetryEnabled records general telemetry regarding Odigos usage.
	TelemetryEnabled bool `json:"telemetryEnabled,omitempty"`

	// OpenShiftEnabled configures selinux on OpenShift nodes.
	OpenShiftEnabled bool `json:"openshiftEnabled,omitempty"`

	// IgnoredNamespaces is a list of namespaces to not show in the Odigos UI
	IgnoredNamespaces []string `json:"ignoredNamespaces,omitempty"`

	// IgnoredContainers is a list of container names to exclude from instrumentation (useful for sidecars)
	IgnoredContainers []string `json:"ignoredContainers,omitempty"`

	// SkipWebhookIssuerCreation skips creating the Issuer and Certificate for the Instrumentor pod webhook if cert-manager is installed.
	SkipWebhookIssuerCreation bool `json:"skipWebhookIssuerCreation,omitempty"`

	// PodSecurityPolicy enables the pod security policy.
	PodSecurityPolicy bool `json:"podSecurityPolicy,omitempty"`

	// ImagePrefix is the prefix for all container images. used when your cluster doesn't have access to docker hub
	ImagePrefix string `json:"imagePrefix,omitempty"`

	// Profiles is a list of preset profiles with a specific configuration.
	Profiles []common.ProfileName `json:"profiles,omitempty"`

	// UIMode sets the UI mode (one-of: normal, readonly)
	UIMode string `json:"uiMode,omitempty"`

	// OnPremToken is an optional enterprise token for Odigos Enterprise.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	OnPremToken string `json:"onPremToken,omitempty"`
	// APIKey is an API Key for Odigos Cloud
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	APIKey string `json:"apiKey,omitempty"`
}

// OdigosStatus defines the observed state of Odigos
type OdigosStatus struct {
	// Conditions store the status conditions of the Odigos instances
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Odigos is the Schema for the odigos API
type Odigos struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OdigosSpec   `json:"spec,omitempty"`
	Status OdigosStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OdigosList contains a list of Odigos
type OdigosList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Odigos `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Odigos{}, &OdigosList{})
}
