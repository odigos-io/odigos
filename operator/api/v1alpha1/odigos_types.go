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
	// (Optional) OnPremToken is an enterprise token for Odigos Enterprise.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="On-Prem Token",order=1
	OnPremToken string `json:"onPremToken,omitempty"`

	// (Optional) UIMode sets the UI mode to either "normal" or "readonly".
	// In "normal" mode the UI is fully interactive, allowing users to view and edit
	// Odigos configuration and settings. In "readonly" mode, the UI can only be
	// used to view current Odigos configuration and is not interactive.
	// Default=default
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="UI Mode",order=2
	UIMode common.UiMode `json:"uiMode,omitempty"`

	// (Optional) TelemetryEnabled records general telemetry regarding Odigos usage.
	// Default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	TelemetryEnabled bool `json:"telemetryEnabled,omitempty"`

	// (Optional) IgnoredNamespaces is a list of namespaces to not show in the Odigos UI.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	IgnoredNamespaces []string `json:"ignoredNamespaces,omitempty"`

	// (Optional) IgnoredContainers is a list of container names to exclude from instrumentation (useful for ignoring sidecars).
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	IgnoredContainers []string `json:"ignoredContainers,omitempty"`

	// (Optional) Profiles is a list of preset profiles with a specific configuration.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	Profiles []common.ProfileName `json:"profiles,omitempty"`

	// (Optional) AgentEnvVarsInjectionMethod is the method to inject the Odigos agent env vars into the pod.
	// Default=pod-manifest
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	AgentEnvVarsInjectionMethod common.EnvInjectionMethod `json:"agentEnvVarsInjectionMethod,omitempty"`

	// (Optional) NodeSelector is a map of key-value Kubernetes NodeSelector labels to apply to all Odigos components.
	// Note that Odigos will only be able to instrument applications on the same node.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// (Optional) SkipWebhookIssuerCreation skips creating the Issuer and Certificate for the Instrumentor pod webhook if cert-manager is installed.
	// Default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SkipWebhookIssuerCreation bool `json:"skipWebhookIssuerCreation,omitempty"`

	// (Optional) PodSecurityPolicy allows Odigos pods to use a privileged pod security policy.
	// Default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PodSecurityPolicy bool `json:"podSecurityPolicy,omitempty"`

	// (Optional) ImagePrefix is a prefix for all container images.
	// This should only be used if you are pulling Odigos images from the non-default registry.
	// Default: registry.odigos.io
	// Default (OpenShift): registry.connect.redhat.com
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ImagePrefix string `json:"imagePrefix,omitempty"`

	// (Optional) LogLevel sets the default log level for all Odigos components (e.g. autoscaler, odiglet, scheduler).
	// Accepted values: debug, info, warn, error. Default=info when unset.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Log Level"
	LogLevel string `json:"logLevel,omitempty"`

	// (Optional) MountMethod optionally defines the mechanism for mounting Odigos files into instrumented pods.
	// One of "k8s-virtual-device" (default) or "k8s-host-path".
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Mount Method"
	MountMethod common.MountMethod `json:"mountMethod,omitempty"`

	// (Optional) OpenShiftEnabled configures selinux on OpenShift nodes.
	// DEPRECATED: OpenShift clusters are auto-detected and this API field will be removed in a future release.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="OpenShift Enabled"
	OpenShiftEnabled bool `json:"openshiftEnabled,omitempty"`
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
