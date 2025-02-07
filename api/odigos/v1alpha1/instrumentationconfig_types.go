package v1alpha1

import (
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// InstrumentationConfig is the Schema for the instrumentationconfig API
type InstrumentationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationConfigSpec   `json:"spec,omitempty"`
	Status InstrumentationConfigStatus `json:"status,omitempty"`
}

// conditions for the InstrumentationConfigStatus
const (
	// this const is the Type field in the conditions of the InstrumentationConfigStatus.
	AgentEnabledStatusConditionType = "AgentEnabled"
	// reports whether the workload associated with the InstrumentationConfig has been rolled out.
	// the rollout is needed to update the instrumentation done by the Pods webhook.
	WorkloadRolloutStatusConditionType = "WorkloadRollout"
)

// +kubebuilder:validation:Enum=EnabledSuccessfully;WaitingForRuntimeInspection;WaitingForNodeCollector;UnsupportedProgrammingLanguage;IgnoredContainer;NoAvailableAgent;UnsupportedRuntimeVersion;MissingDistroParameter;OtherAgentDetected
type AgentEnabledReason string

const (
	AgentEnabledReasonEnabledSuccessfully            AgentEnabledReason = "EnabledSuccessfully"
	AgentEnabledReasonWaitingForRuntimeInspection    AgentEnabledReason = "WaitingForRuntimeInspection"
	AgentEnabledReasonWaitingForNodeCollector        AgentEnabledReason = "WaitingForNodeCollector"
	AgentEnabledReasonUnsupportedProgrammingLanguage AgentEnabledReason = "UnsupportedProgrammingLanguage"
	AgentEnabledReasonIgnoredContainer               AgentEnabledReason = "IgnoredContainer"
	AgentEnabledReasonNoAvailableAgent               AgentEnabledReason = "NoAvailableAgent"
	AgentEnabledReasonUnsupportedRuntimeVersion      AgentEnabledReason = "UnsupportedRuntimeVersion"
	AgentEnabledReasonMissingDistroParameter         AgentEnabledReason = "MissingDistroParameter"
	AgentEnabledReasonOtherAgentDetected             AgentEnabledReason = "OtherAgentDetected"
)

// +kubebuilder:validation:Enum=RolloutTriggeredSuccessfully;FailedToPatch;PreviousRolloutOngoing
type WorkloadRolloutReason string

const (
	WorkloadRolloutReasonTriggeredSuccessfully  WorkloadRolloutReason = "RolloutTriggeredSuccessfully"
	WorkloadRolloutReasonFailedToPatch          WorkloadRolloutReason = "FailedToPatch"
	WorkloadRolloutReasonPreviousRolloutOngoing WorkloadRolloutReason = "PreviousRolloutOngoing"
)

// givin multiple reasons for not injecting an agent, this function returns the priority of the reason.
// which is - it allows choosing the most important reason to be displayed to the user in the aggregate status.
func AgentInjectionReasonPriority(reason AgentEnabledReason) int {
	switch reason {
	case AgentEnabledReasonEnabledSuccessfully:
		return 0
	case AgentEnabledReasonWaitingForRuntimeInspection:
		return 1
	case AgentEnabledReasonWaitingForNodeCollector:
		return 2
	case AgentEnabledReasonUnsupportedProgrammingLanguage:
		return 3
	case AgentEnabledReasonUnsupportedRuntimeVersion:
		return 4
	case AgentEnabledReasonIgnoredContainer:
		return 5
	case AgentEnabledReasonNoAvailableAgent:
		return 6
	case AgentEnabledReasonMissingDistroParameter:
		return 7
	case AgentEnabledReasonOtherAgentDetected:
		return 8
	default:
		return 9
	}
}

type OtherAgent struct {
	Name string `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true
type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProcessingState string

const (
	ProcessingStateFailed    ProcessingState = "Failed"    // Used when CRI fails to detect the runtime envs
	ProcessingStateSucceeded ProcessingState = "Succeeded" // Indicates that CRI successfully processed the runtime environments, even if no environments were detected.
	ProcessingStateSkipped   ProcessingState = "Skipped"   // Used when env originally come from manifest
)

// +kubebuilder:object:generate=true
type RuntimeDetailsByContainer struct {
	ContainerName  string                     `json:"containerName"`
	Language       common.ProgrammingLanguage `json:"language"`
	RuntimeVersion string                     `json:"runtimeVersion,omitempty"`
	EnvVars        []EnvVar                   `json:"envVars,omitempty"`
	OtherAgent     *OtherAgent                `json:"otherAgent,omitempty"`
	LibCType       *common.LibCType           `json:"libCType,omitempty"`

	// Stores the error message from the CRI runtime if returned to prevent instrumenting the container if an error exists.
	CriErrorMessage *string `json:"criErrorMessage,omitempty"`
	// Holds the environment variables retrieved from the container runtime.
	EnvFromContainerRuntime []EnvVar `json:"envFromContainerRuntime,omitempty"`
	// A temporary variable used during migration to track whether the new runtime detection process has been executed. If empty, it indicates the process has not yet been run. This field may be removed later.
	RuntimeUpdateState *ProcessingState `json:"runtimeUpdateState,omitempty"`
}

type InstrumentationConfigStatus struct {
	// Capture Runtime Details for the workloads that this CR applies to.
	RuntimeDetailsByContainer []RuntimeDetailsByContainer `json:"runtimeDetailsByContainer,omitempty"`

	// Represents the observations of a InstrumentationConfig's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`

	// The hash used to determine whether the associated workload needs to be rolled out.
	// This hash is calculated based on the containers config array and takes into account the
	// container name, Instrumented flag and the OTel distro name.
	WorkloadRolloutHash string `json:"workloadRolloutHash,omitempty"`
}

func (in *InstrumentationConfigStatus) GetRuntimeDetailsForContainer(container v1.Container) *RuntimeDetailsByContainer {
	for _, runtimeDetails := range in.RuntimeDetailsByContainer {
		if runtimeDetails.ContainerName == container.Name {
			return &runtimeDetails
		}
	}
	return nil
}

// ContainerAgentConfig is a configuration for a specific container in a workload.
type ContainerAgentConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`

	// boolean flag to indicate if the agent should be enabled for this container.
	AgentEnabled bool `json:"agentEnabled"`

	// An enum reason for the agent injection decision.
	AgentEnabledReason AgentEnabledReason `json:"agentEnabledReason,omitempty"`

	// free text message to provide more information about the instrumentation decision.
	// can be left empty if reason is self-explanatory.
	AgentEnabledMessage string `json:"agentEnabledMessage,omitempty"`

	// The name of the otel distribution to use for this container.
	// if the name is empty, this container should not be instrumented.
	OtelDistroName string `json:"otelDistroName,omitempty"`

	// Additional parameters to the distro that controls how it's being applied.
	// Keys are parameter names (like "libc") and values are the value to use for that parameter (glibc / musl)
	DistroParams map[string]string `json:"distroParams,omitempty"`
}

// Config for the OpenTelemeetry SDKs that should be applied to a workload.
// The workload is identified by the owner reference
type InstrumentationConfigSpec struct {
	// the service.name property is used to populate the `service.name` resource attribute in the telemetry generated by this workload
	ServiceName string `json:"serviceName,omitempty"`

	// determines if odigos should inject agents to pods of this workload.
	AgentInjectionEnabled bool `json:"agentInjectionEnabled"`

	// configuration for each instrumented container in the workload
	Containers []ContainerAgentConfig `json:"containers,omitempty"`

	// Configuration for the OpenTelemetry SDKs that this workload should use.
	// The SDKs are identified by the programming language they are written in.
	// TODO: consider adding more granular control over the SDKs, such as community/enterprise, native/ebpf.
	SdkConfigs []SdkConfig `json:"sdkConfigs,omitempty"`
}

func (in *InstrumentationConfigSpec) GetContainerAgentConfig(containerName string) *ContainerAgentConfig {
	for _, containerConfig := range in.Containers {
		if containerConfig.ContainerName == containerName {
			return &containerConfig
		}
	}
	return nil
}

type SdkConfig struct {

	// The language of the SDK being configured
	Language common.ProgrammingLanguage `json:"language"`

	// configurations for the instrumentation libraries the the SDK should use
	InstrumentationLibraryConfigs []InstrumentationLibraryConfig `json:"instrumentationLibraryConfigs,omitempty"`

	// HeadSamplingConfig is a set sampling rules.
	// This config currently only applies to root spans.
	// In the Future we might add another level of configuration base on the parent span (ParentBased Sampling)
	HeadSamplingConfig *HeadSamplingConfig `json:"headSamplerConfig,omitempty"`

	DefaultPayloadCollection *instrumentationrules.PayloadCollection `json:"payloadCollection,omitempty"`

	// default configuration for collecting code attributes, in case the instrumentation library does not provide a configuration.
	DefaultCodeAttributes *instrumentationrules.CodeAttributes `json:"codeAttributes,omitempty"`
}

// 'Operand' represents the attributes and values that an operator acts upon in an expression
type AttributeCondition struct {
	// attribute key (e.g. "url.path")
	Key string `json:"key"`
	// currently only string values are supported.
	Val string `json:"val"`
	// The operator to use to compare the attribute value.
	Operator Operator `json:"operator,omitempty"`
}

// +kubebuilder:validation:Enum=equals;notEquals;endWith;startWith
// +kubebuilder:default:=equals
type Operator string

const (
	Equals    Operator = "equals"
	NotEquals Operator = "notEquals"
	EndWith   Operator = "endWith"
	StartWith Operator = "startWith"
)

// AttributesAndSamplerRule is a set of AttributeCondition that are ANDed together.
// If all attribute conditions evaluate to true, the AND sampler evaluates to true,
// and the fraction is used to determine the sampling decision.
// If any of the attribute compare samplers evaluate to false,
// the fraction is not used and the rule is skipped.
// An "empty" AttributesAndSamplerRule with no attribute conditions is considered to always evaluate to true.
// and the fraction is used to determine the sampling decision.
// This entity is refered to a rule in Odigos terminology for head-sampling.
type AttributesAndSamplerRule struct {
	AttributeConditions []AttributeCondition `json:"attributeConditions"`
	// The fraction of spans to sample, in the range [0, 1].
	// If the fraction is 0, no spans are sampled.
	// If the fraction is 1, all spans are sampled.
	// +kubebuilder:default:=1
	Fraction float64 `json:"fraction"`
}

// HeadSamplingConfig is a set of attribute rules.
// The first attribute rule that evaluates to true is used to determine the sampling decision based on its fraction.
//
// If none of the rules evaluate to true, the fallback fraction is used to determine the sampling decision.
type HeadSamplingConfig struct {
	AttributesAndSamplerRules []AttributesAndSamplerRule `json:"attributesAndSamplerRules"`
	// Used as a fallback if all rules evaluate to false,
	// it may be empty - in this case the default value will be 1 - all spans are sampled.
	// it should be a float value in the range [0, 1] - the fraction of spans to sample.
	// a value of 0 means no spans are sampled if none of the rules evaluate to true.
	// +kubebuilder:default:=1
	FallbackFraction float64 `json:"fallbackFraction"`
}

type InstrumentationLibraryConfig struct {
	InstrumentationLibraryId InstrumentationLibraryId `json:"libraryId"`

	TraceConfig *InstrumentationLibraryConfigTraces `json:"traceConfig,omitempty"`

	PayloadCollection *instrumentationrules.PayloadCollection `json:"payloadCollection,omitempty"`

	// code attributes configuration for a specific library.
	// if not set, the default code attributes configuration for the workload will be used.
	// if set, but internal fields are empty, those fields will be used from the default configuration.
	CodeAttributes *instrumentationrules.CodeAttributes `json:"codeAttributes,omitempty"`
}

type InstrumentationLibraryId struct {
	// The name of the instrumentation library
	// - Node.js: The name of the npm package: `@opentelemetry/instrumentation-<name>`
	InstrumentationLibraryName string `json:"libraryName"`
	// SpanKind is only supported by Golang and will be ignored for any other SDK language.
	// In Go, SpanKind is used because the same instrumentation library can be utilized for different span kinds (e.g., client/server).
	SpanKind common.SpanKind `json:"spanKind,omitempty"`
}

type InstrumentationLibraryConfigTraces struct {
	// Whether the instrumentation library is enabled to record traces.
	// When false, it is expected that the instrumentation library does not produce any spans regardless of any other configuration.
	// When true, the instrumentation library should produce spans according to the other configuration options.
	// If not specified, the default value for this signal should be used (whether to enable libraries by default or not).
	Enabled *bool `json:"enabled,omitempty"`
}

// +kubebuilder:object:root=true

// InstrumentationConfigList contains a list of InstrumentationOption
type InstrumentationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstrumentationConfig{}, &InstrumentationConfigList{})
}

// Languages returns the set of languages that this configuration applies to
func (ic *InstrumentationConfig) Languages() map[common.ProgrammingLanguage]struct{} {
	langs := make(map[common.ProgrammingLanguage]struct{})
	for _, sdkConfig := range ic.Spec.SdkConfigs {
		langs[sdkConfig.Language] = struct{}{}
	}
	return langs
}
