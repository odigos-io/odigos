/*
Copyright 2024.

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

// CloudConnectorWorkloadCapability is the customer's allow-list entry for one directly-instrumentable
// resource type (a workload, e.g. "aws.lambda", "aws.fargate-task"): whether the connector may discover
// and/or instrument that type. It records intent — instrumentation only runs once the provider's connector
// spec implements it; until then the connector simply knows it is allowed.
type CloudConnectorWorkloadCapability struct {
	// Type is the provider connector spec type key (e.g. "aws.lambda").
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Discovery allows the connector to discover workloads of this type.
	// +kubebuilder:validation:Optional
	Discovery bool `json:"discovery,omitempty"`

	// Instrumentation allows the connector to instrument workloads of this type directly.
	// +kubebuilder:validation:Optional
	Instrumentation bool `json:"instrumentation,omitempty"`
}

// CloudConnectorComputePlatformCapability is the customer's allow-list entry for one managed-environment
// type (a compute platform, e.g. "aws.ecs-cluster", "aws.eks", "aws.ec2"): whether the connector may
// discover the environment and/or install a separate Odigos agent onto it. Installation records intent —
// it only runs once the provider's connector spec implements it.
type CloudConnectorComputePlatformCapability struct {
	// Type is the provider connector spec type key (e.g. "aws.eks").
	// +kubebuilder:validation:Required
	Type string `json:"type"`

	// Discovery allows the connector to discover compute platforms of this type.
	// +kubebuilder:validation:Optional
	Discovery bool `json:"discovery,omitempty"`

	// Installation allows the connector to install a separate Odigos agent onto a discovered compute
	// platform of this type (e.g. the Odigos ecs-agent onto an EC2-backed ECS cluster).
	// +kubebuilder:validation:Optional
	Installation bool `json:"installation,omitempty"`
}

// CloudConnectorPhase is a coarse, cached lifecycle phase for the connector runtime. It is a cache of
// runtime-authoritative state computed by the connector-runtime controller; the connector's Postgres
// schema and runtime API remain authoritative.
// +kubebuilder:validation:Enum=Pending;Starting;Connected;Degraded;Error;Disabled;Deleting
type CloudConnectorPhase string

const (
	CloudConnectorPhasePending   CloudConnectorPhase = "Pending"
	CloudConnectorPhaseStarting  CloudConnectorPhase = "Starting"
	CloudConnectorPhaseConnected CloudConnectorPhase = "Connected"
	CloudConnectorPhaseDegraded  CloudConnectorPhase = "Degraded"
	CloudConnectorPhaseError     CloudConnectorPhase = "Error"
	CloudConnectorPhaseDisabled  CloudConnectorPhase = "Disabled"
	CloudConnectorPhaseDeleting  CloudConnectorPhase = "Deleting"
)

// CloudConnectorAccount identifies the external provider account boundary a connector runtime owns:
// one AWS account / Azure subscription / GCP project / SaaS tenant.
type CloudConnectorAccount struct {
	// Provider account/subscription/project/tenant identifier (e.g. an AWS account id).
	// +kubebuilder:validation:Required
	ID string `json:"id"`

	// Human-friendly account name shown in the UI.
	// +kubebuilder:validation:Optional
	DisplayName string `json:"displayName,omitempty"`
}

// CloudConnectorCredentialsSecretRef references the Kubernetes Secret holding provider credentials.
// Credentials are never stored in the CRD itself (see the security model).
type CloudConnectorCredentialsSecretRef struct {
	// Name of the Secret in the connector's namespace.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// CloudConnectorRuntime describes the connector workload the connector-runtime controller manages.
type CloudConnectorRuntime struct {
	// Container image for the provider connector runtime. When empty, the controller resolves a
	// default from its configured catalog for the provider/connectorType.
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// Number of runtime replicas. One runtime owns one account boundary; defaults to 1.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Resource requirements for the runtime container.
	// +kubebuilder:validation:Optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
}

// CloudConnectorDiscovery configures how the runtime discovers provider resources.
type CloudConnectorDiscovery struct {
	// How often the runtime re-scans the provider for inventory changes (e.g. "10m").
	// +kubebuilder:validation:Optional
	RefreshInterval *metav1.Duration `json:"refreshInterval,omitempty"`

	// Selectors narrows discovery (e.g. only resources carrying specific tags).
	// +kubebuilder:validation:Optional
	Selectors *CloudConnectorDiscoverySelectors `json:"selectors,omitempty"`
}

type CloudConnectorDiscoverySelectors struct {
	// Tags restricts discovery to provider resources carrying all of these tag key/values.
	// +kubebuilder:validation:Optional
	Tags map[string]string `json:"tags,omitempty"`
}

type OdigosCloudConnectorSpec struct {
	// Provider family this connector targets (e.g. "aws", "azure", "gcp"). Kept as a free-form string
	// so new providers need no API change.
	// +kubebuilder:validation:Required
	Provider string `json:"provider"`

	// ConnectorType selects the specific connector implementation/image family. Usually equals Provider.
	// +kubebuilder:validation:Optional
	ConnectorType string `json:"connectorType,omitempty"`

	// DisplayName is the human-friendly connector name shown in the UI.
	// +kubebuilder:validation:Optional
	DisplayName string `json:"displayName,omitempty"`

	// Account is the provider account boundary this connector owns.
	// +kubebuilder:validation:Required
	Account CloudConnectorAccount `json:"account"`

	// CredentialsSecretRef references the Secret holding provider credentials.
	// +kubebuilder:validation:Required
	CredentialsSecretRef CloudConnectorCredentialsSecretRef `json:"credentialsSecretRef"`

	// Workloads is the customer's per-type allow-list for directly-instrumentable resources: what the
	// connector may do (discovery and/or instrumentation) per workload type (e.g. aws.lambda,
	// aws.fargate-task). The connector scans every region enabled in the account; there is no per-connector
	// region selection.
	// +kubebuilder:validation:Optional
	Workloads []CloudConnectorWorkloadCapability `json:"workloads,omitempty"`

	// ComputePlatforms is the customer's per-type allow-list for managed environments: what the connector
	// may do (discovery and/or installation) per compute-platform type (e.g. aws.ecs-cluster, aws.eks,
	// aws.ec2).
	// +kubebuilder:validation:Optional
	ComputePlatforms []CloudConnectorComputePlatformCapability `json:"computePlatforms,omitempty"`

	// Runtime configures the connector workload (image/replicas/resources).
	// +kubebuilder:validation:Optional
	Runtime CloudConnectorRuntime `json:"runtime,omitempty"`

	// Discovery configures provider resource discovery behavior.
	// +kubebuilder:validation:Optional
	Discovery CloudConnectorDiscovery `json:"discovery,omitempty"`
}

// CloudConnectorRuntimeStatus mirrors runtime-authoritative liveness into the CRD status cache.
type CloudConnectorRuntimeStatus struct {
	// WorkloadName is the connector Deployment name created by the controller.
	WorkloadName string `json:"workloadName,omitempty"`
	// Namespace the connector workload runs in.
	Namespace string `json:"namespace,omitempty"`
	// Image currently running.
	Image string `json:"image,omitempty"`
	// Version reported by the runtime at registration.
	Version string `json:"version,omitempty"`
	// LastSeenAt is the last heartbeat time mirrored from the platform heartbeat table.
	LastSeenAt *metav1.Time `json:"lastSeenAt,omitempty"`
	// Message is a human-readable runtime status detail.
	Message string `json:"message,omitempty"`
}

// CloudConnectorProviderIdentity records the verified provider identity for the account.
type CloudConnectorProviderIdentity struct {
	AccountID   string `json:"accountId,omitempty"`
	DisplayName string `json:"displayName,omitempty"`
}

type OdigosCloudConnectorStatus struct {
	// Phase is a coarse cached lifecycle phase.
	// +optional
	Phase CloudConnectorPhase `json:"phase,omitempty"`

	// LastSyncedAt marks the freshness of this status cache.
	// +optional
	LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`

	// ObservedGeneration is the spec generation the controller last reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Runtime mirrors runtime-authoritative liveness.
	// +optional
	Runtime *CloudConnectorRuntimeStatus `json:"runtime,omitempty"`

	// ProviderIdentity records the verified provider identity.
	// +optional
	ProviderIdentity *CloudConnectorProviderIdentity `json:"providerIdentity,omitempty"`

	// Conditions represent the latest available observations of the connector's state.
	// Known condition types: RuntimeDeploymentReady, SchemaReady, RuntimeRegistered, Authenticated,
	// DiscoveryReady, CleanupFailed.
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// OdigosCloudConnector is the generic installation desired-state for a cloud/SaaS connector runtime
// (platformType = connector). One OdigosCloudConnector maps to one connector Deployment that owns one
// provider account boundary.
// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.provider`
// +kubebuilder:printcolumn:name="Account",type=string,JSONPath=`.spec.account.id`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
// +kubebuilder:metadata:labels=odigos.io/system-object=true
type OdigosCloudConnector struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OdigosCloudConnectorSpec   `json:"spec"`
	Status OdigosCloudConnectorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

type OdigosCloudConnectorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OdigosCloudConnector `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OdigosCloudConnector{}, &OdigosCloudConnectorList{})
}
