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
// +kubebuilder:metadata:labels=odigos.io/system-object=true

// InstrumentationConfig is the Schema for the instrumentationconfig API
type InstrumentationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationConfigSpec   `json:"spec,omitempty"`
	Status InstrumentationConfigStatus `json:"status,omitempty"`
}

// conditions for the InstrumentationConfigStatus
const (
	// Define a status condition type that describes why the workload is marked for instrumentation or not.
	MarkedForInstrumentationStatusConditionType = "MarkedForInstrumentation"
	// Describe the runtime detection status of this workload.
	RuntimeDetectionStatusConditionType = "RuntimeDetection"
	// this const is the Type field in the conditions of the InstrumentationConfigStatus.
	AgentEnabledStatusConditionType = "AgentEnabled"
	// reports whether the workload associated with the InstrumentationConfig has been rolled out.
	// the rollout is needed to update the instrumentation done by the Pods webhook.
	WorkloadRolloutStatusConditionType = "WorkloadRollout"
)

func StatusConditionTypeLogicalOrder(condType string) int {
	switch condType {
	case MarkedForInstrumentationStatusConditionType:
		return 1
	case RuntimeDetectionStatusConditionType:
		return 2
	case AgentEnabledStatusConditionType:
		return 3
	case WorkloadRolloutStatusConditionType:
		return 4
	default:
		return 5
	}
}

// +kubebuilder:validation:Enum=WorkloadSource;NamespaceSource;WorkloadSourceDisabled;NoSource;RetirableError
type MarkedForInstrumentationReason string

const (
	// denotes that the workload is instrumented because of a source CR exists for this workload.
	// and the source is not disabled..
	MarkedForInstrumentationReasonWorkloadSource MarkedForInstrumentationReason = "WorkloadSource"

	// denotes that the workload does not have a source CR, but the namespace has a source CR,
	// so the workload is instrumented as inherited from the namespace.
	MarkedForInstrumentationReasonNamespaceSource MarkedForInstrumentationReason = "NamespaceSource"

	// the source object for workload exists, and it is disabled, thus uninstrumented.
	MarkedForInstrumentationReasonWorkloadSourceDisabled MarkedForInstrumentationReason = "WorkloadSourceDisabled"

	// this workload is not instrumented because no source CR exists for it or its namespace.
	MarkedForInstrumentationReasonNoSource MarkedForInstrumentationReason = "NoSource"

	// cannot determine the reason for the instrumentation due to a possible transient error.
	MarkedForInstrumentationReasonError MarkedForInstrumentationReason = "RetirableError"
)

// +kubebuilder:validation:Enum=NotMakredForInstrumentation;DetectedSuccessfully;WaitingForDetection;NoRunningPods;Error
type RuntimeDetectionReason string

const (
	// when the runtime detection process is successful and runtime details are available for instrumentation.
	RuntimeDetectionReasonDetectedSuccessfully RuntimeDetectionReason = "DetectedSuccessfully"
	// when the runtime detection process is still ongoing and the runtime details are not yet available.
	// this status should be visible only for a short period of time until the detection process is completed by one odiglet.
	RuntimeDetectionReasonWaitingForDetection RuntimeDetectionReason = "WaitingForDetection"
	// when the runtime detection process is not yet started because there are no running pods for this workload.
	// runtime detection requires at least one running pod to inspect the runtime details from.
	RuntimeDetectionReasonNoRunningPods RuntimeDetectionReason = "NoRunningPods"
	// error occurred during the runtime detection process.
	RuntimeDetectionReasonError RuntimeDetectionReason = "Error"
)

// +kubebuilder:validation:Enum=EnabledSuccessfully;WaitingForRuntimeInspection;WaitingForNodeCollector;IgnoredContainer;NoCollectedSignals;InjectionConflict;UnsupportedProgrammingLanguage;NoAvailableAgent;UnsupportedRuntimeVersion;MissingDistroParameter;OtherAgentDetected;RuntimeDetailsUnavailable;CrashLoopBackOff;ImagePullBackOff
type AgentEnabledReason string

const (
	AgentEnabledReasonEnabledSuccessfully            AgentEnabledReason = "EnabledSuccessfully"
	AgentEnabledReasonWaitingForRuntimeInspection    AgentEnabledReason = "WaitingForRuntimeInspection"
	AgentEnabledReasonWaitingForNodeCollector        AgentEnabledReason = "WaitingForNodeCollector"
	AgentEnabledReasonIgnoredContainer               AgentEnabledReason = "IgnoredContainer"
	AgentEnabledReasonNoCollectedSignals             AgentEnabledReason = "NoCollectedSignals"
	AgentEnabledReasonUnsupportedProgrammingLanguage AgentEnabledReason = "UnsupportedProgrammingLanguage"
	AgentEnabledReasonNoAvailableAgent               AgentEnabledReason = "NoAvailableAgent"
	AgentEnabledReasonInjectionConflict              AgentEnabledReason = "InjectionConflict"
	AgentEnabledReasonUnsupportedRuntimeVersion      AgentEnabledReason = "UnsupportedRuntimeVersion"
	AgentEnabledReasonMissingDistroParameter         AgentEnabledReason = "MissingDistroParameter"
	AgentEnabledReasonOtherAgentDetected             AgentEnabledReason = "OtherAgentDetected"
	// if the source cannot be instrumented because there are no running pods,
	// we want to show this reason to the user so it's not a spinner
	AgentEnabledReasonRuntimeDetailsUnavailable AgentEnabledReason = "RuntimeDetailsUnavailable"
	// used for the rollback feature, when an application was instrumented and it caused a CrashLoopBackOff
	// We're marking it as that and rolling back the instrumentation
	AgentEnabledReasonCrashLoopBackOff AgentEnabledReason = "CrashLoopBackOff"
	// used for the rollback feature, when an application was instrumented and it caused an ImagePullBackOff
	// We're marking it as that and rolling back the instrumentation
	AgentEnabledReasonImagePullBackOff AgentEnabledReason = "ImagePullBackOff"
)

// +kubebuilder:validation:Enum=RolloutTriggeredSuccessfully;FailedToPatch;PreviousRolloutOngoing;Disabled;WaitingForRestart;WorkloadNotSupporting
type WorkloadRolloutReason string

const (
	WorkloadRolloutReasonTriggeredSuccessfully  WorkloadRolloutReason = "RolloutTriggeredSuccessfully"
	WorkloadRolloutReasonFailedToPatch          WorkloadRolloutReason = "FailedToPatch"
	WorkloadRolloutReasonPreviousRolloutOngoing WorkloadRolloutReason = "PreviousRolloutOngoing"
	WorkloadRolloutReasonDisabled               WorkloadRolloutReason = "Disabled"
	WorkloadRolloutReasonNotRequired            WorkloadRolloutReason = "NotRequired"
	WorkloadRolloutReasonWaitingForRestart      WorkloadRolloutReason = "WaitingForRestart"
	WorkloadRolloutReasonWorkloadNotSupporting  WorkloadRolloutReason = "WorkloadNotSupporting"
)

const (
	// K8s workload status conditions, not set in the InstrumentationConfigStatus but used in the frontend to display the status.
	K8sWorkloadRolloutReasonFailedCreate = "FailedCreate"
)

// givin multiple reasons for not injecting an agent, this function returns the priority of the reason.
// which is - it allows choosing the most important reason to be displayed to the user in the aggregate status.
func AgentInjectionReasonPriority(reason AgentEnabledReason) int {
	switch reason {
	case AgentEnabledReasonEnabledSuccessfully:
		return 0
	case AgentEnabledReasonRuntimeDetailsUnavailable:
		return 10
	case AgentEnabledReasonWaitingForRuntimeInspection:
		return 20
	case AgentEnabledReasonWaitingForNodeCollector:
		return 30
	case AgentEnabledReasonIgnoredContainer:
		return 40
	case AgentEnabledReasonNoCollectedSignals:
		return 45
	case AgentEnabledReasonInjectionConflict:
		return 48
	case AgentEnabledReasonUnsupportedProgrammingLanguage:
		return 50
	case AgentEnabledReasonUnsupportedRuntimeVersion:
		return 60
	case AgentEnabledReasonNoAvailableAgent:
		return 70
	case AgentEnabledReasonMissingDistroParameter:
		return 80
	case AgentEnabledReasonOtherAgentDetected:
		return 90
	case AgentEnabledReasonCrashLoopBackOff, AgentEnabledReasonImagePullBackOff:
		return 95
	default:
		return 100
	}
}

// some conditions with "status: false" should be considered as "disabled" status.
// this function returns true if the reason is considered as "disabled" status.
func IsReasonStatusDisabled(reason string) bool {
	switch reason {
	// Agent-related reasons
	case string(AgentEnabledReasonUnsupportedProgrammingLanguage),
		string(AgentEnabledReasonUnsupportedRuntimeVersion),
		string(RuntimeDetectionReasonNoRunningPods),
		string(AgentEnabledReasonNoCollectedSignals),
		string(AgentEnabledReasonIgnoredContainer),
		string(AgentEnabledReasonNoAvailableAgent),
		string(AgentEnabledReasonInjectionConflict),
		string(AgentEnabledReasonOtherAgentDetected),
		string(AgentEnabledReasonCrashLoopBackOff),
		string(AgentEnabledReasonImagePullBackOff),
		string(AgentEnabledReasonRuntimeDetailsUnavailable):

		return true

	// rollout-related reasons
	case string(K8sWorkloadRolloutReasonFailedCreate):

		return true

	default:
		return false
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
	// Indicates whether the target process is running is secure-execution mode.
	// nil means we were unable to determine the secure-execution mode.
	SecureExecutionMode *bool `json:"secureExecutionMode,omitempty"`

	// CriErrorMessage is set if the value in EnvFromContainerRuntime was not computed correctly and cannot be used safely.
	// Sometimes, even if CRI check failed, it is possible to tell that relevant env vars are not coming from container runtime.
	// Thus, this field is set only when there is:
	// - Actual CRI check failed
	// - The observed environment variables might come from container runtime
	CriErrorMessage *string `json:"criErrorMessage,omitempty"`
	// Holds the environment variables retrieved from the container runtime.
	EnvFromContainerRuntime []EnvVar `json:"envFromContainerRuntime,omitempty"`
	// A temporary variable used during migration to track whether the new runtime detection process has been executed. If empty, it indicates the process has not yet been run. This field may be removed later.
	RuntimeUpdateState *ProcessingState `json:"runtimeUpdateState,omitempty"`
}

// represents the status of odigos MANIFEST injection to existing pods template.
// each pod can be in one of 3 states: injected and up-to-date, injected and out-of-date, or not injected.
// actual agent in pod container can be injected or not regardless of the pod manifest injection status.
// status only shows if pods exists from each category and not count, to avoid frequent (noisy/expensive) updates in steady state
type PodsManifestInjectionStatus struct {
	// if the source has any pods which are using the latest agent deployment setup.
	HasInjectedUpToDatePods bool `json:"hasInjectedUpToDatePods,omitempty"`

	// if the source has any pods which are using an outdated agent deployment setup.
	HasInjectedOutOfDatePods bool `json:"hasInjectedOutOfDatePods,omitempty"`

	// if the source has any pods which do not have the odigos agent injected.
	HasUninjectedPods bool `json:"hasUninjectedPods,omitempty"`
}

type InstrumentationConfigStatus struct {
	// Capture Runtime Details for the workloads that this CR applies to.
	RuntimeDetailsByContainer []RuntimeDetailsByContainer `json:"runtimeDetailsByContainer,omitempty"`

	// Represents the observations of a InstrumentationConfig's current state.
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`

	// This hash is recorded only after the rollout took place.
	// it allows us to determine if the workload needs to be rollout based on previous rollout and the current config.
	// if this field is different than the spec.AgentsDeploymentHash it means rollout is needed or not yet updated.
	WorkloadRolloutHash string `json:"workloadRolloutHash,omitempty"`

	// Check if rollback happened to an application
	RollbackOccurred bool `json:"rollbackOccurred,omitempty"`
	// This time recorded only after the rollout took place.
	// This allows us to determine whether a crashing application should be rolled back or not
	InstrumentationTime *metav1.Time `json:"instrumentationTime,omitempty"`

	// Represents the status of odigos MANIFEST injection to existing pods template.
	PodsManifestInjectionStatus *PodsManifestInjectionStatus `json:"podsManifestInjectionStatus,omitempty"`
}

func (in *InstrumentationConfigStatus) GetRuntimeDetailsForContainer(container v1.Container) *RuntimeDetailsByContainer {
	for _, runtimeDetails := range in.RuntimeDetailsByContainer {
		if runtimeDetails.ContainerName == container.Name {
			return &runtimeDetails
		}
	}
	return nil
}

// random id generator is the default, and most common.
// it creates span ids and trace ids using random bytes.
// It has no configuration.
type IdGeneratorRandomConfig struct{}

// trace id includes timestamp, source id byte, and random number bytes.
// this id generator can be leveraged by databases to do efficient indexing.
type IdGeneratorTimedWallConfig struct {
	// sourceId is a number between 0-255 (8 bits) written into the 8th byte of the trace id.
	// if timedWall is specified, the sourceId is required.
	SourceId uint8 `json:"sourceId"`
}

// id generator configuration for the traces
type IdGeneratorConfig struct {
	Random    *IdGeneratorRandomConfig    `json:"random,omitempty"`
	TimedWall *IdGeneratorTimedWallConfig `json:"timedWall,omitempty"`
}

type AgentSpanMetricsConfig struct {
	// additional dimensions to add for the span metrics.
	// for example, if you add `http.method` to the dimensions,
	// then the span metrics data points will include the `http.method` in the attributes,
	// and different values of `http.method` will be aggregated into different time series.
	Dimensions []string `json:"dimensions,omitempty"`

	// time interval in miliseconds for flushing the span metrics.
	// defaults: 60000 (60 seconds, 1 minute)
	IntervalMs int `json:"intervalMs,omitempty"`

	// explicit buckets list for the histogram metrics in ms
	HistogramBucketsMs []int `json:"histogramBucketsMs,omitempty"`
}

type UrlTemplatizationConfig struct {
	// Rule is the template rule to be applied to URLs
	Rules []string `json:"templatizationRules,omitempty"`
}

// HeadersCollectionConfig represents configuration for HTTP headers collection.
type HeadersCollectionConfig struct {
	// Limit HTTP headers collection to specific header keys.
	// if unset, no HTTP headers will be collected.
	// HTTP headers cannot be collected as wildcard, to avoid leaking sensitive information.
	HttpHeaderKeys []string `json:"httpHeaderKeys,omitempty"`
}

// all "traces" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that trace collection is enabled for this container.
type AgentTracesConfig struct {
	// id generator configuration for the traces.
	// if not specified, the default random id generator will be used.
	IdGenerator *IdGeneratorConfig `json:"idGenerator,omitempty"`

	// A list of URL templatization configurations to be applied to the traces.
	UrlTemplatization *UrlTemplatizationConfig `json:"urlTemplatization,omitempty"`

	// Configuration for headers collection. If not specified, no headers will be collected.
	HeadersCollection *HeadersCollectionConfig `json:"headersCollection,omitempty"`

	// HeadSamplingConfig is a set sampling rules.
	// This config currently only applies to root spans.
	// In the Future we might add another level of configuration base on the parent span (ParentBased Sampling)
	HeadSampling *HeadSamplingConfig `json:"headSampling,omitempty"`
}

// all "metrics" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that metrics collection is enabled for this container.
type AgentMetricsConfig struct {

	// if not nil, it means agent should report span metrics,
	// calculated directly in the agent.
	// this is most accurate as it includes any sampled spans,
	// and is not affected if spans are dropped anywhere in the pipeline.
	SpanMetrics *AgentSpanMetricsConfig `json:"spanMetrics,omitempty"`
}

// all "logs" related configuration for an agent running on any process in a specific container.
// The presence of this struct (as opposed to nil) means that logs collection is enabled for this container.
type AgentLogsConfig struct{}

// ContainerAgentConfig is a configuration for a specific container in a workload.
type ContainerAgentConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`

	// boolean flag to indicate if the agent should be enabled for this container.
	AgentEnabled bool `json:"agentEnabled"`

	// set to true if the agent in this container requires pod manifest injection to be enabled.
	PodManifestInjectionRequired bool `json:"podManifestInjectionRequired,omitempty"`

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

	// What method to use for injecting the agent environment variables (just those covered by the loader (PYTHONPATH, JAVA_TOOLS_OPTIONS, NODE_OPTIONS))
	// Can be either "loader" or "pod-manifest".
	// The injection should still check the actual values in the container manifest before injecting.
	// Nil means that this container should not have env injection (agent should not be injected, or distro does not specify "loader" injection envs).
	EnvInjectionMethod *common.EnvInjectionDecision `json:"envInjectionMethod,omitempty"`

	// Each enabled signal must be set with a non-nil value (even if the config content is empty).
	// nil means that the signal is disabled and should not be instrumented/collected by the agent.
	Traces  *AgentTracesConfig  `json:"traces,omitempty"`
	Metrics *AgentMetricsConfig `json:"metrics,omitempty"`
	Logs    *AgentLogsConfig    `json:"logs,omitempty"`
}

// Config for the OpenTelemeetry SDKs that should be applied to a workload.
// The workload is identified by the owner reference
type InstrumentationConfigSpec struct {
	// the service.name property is used to populate the `service.name` resource attribute in the telemetry generated by this workload
	ServiceName string `json:"serviceName,omitempty"`

	// determines if odigos should inject agents to pods of this workload.
	AgentInjectionEnabled bool `json:"agentInjectionEnabled"`

	// true if at least one container in this workload requires pod manifest injection
	// to enable agent injection.
	PodManifestInjectionRequired bool `json:"podManifestInjectionRequired,omitempty"`

	// configuration for each instrumented container in the workload
	Containers []ContainerAgentConfig `json:"containers,omitempty"`

	// will always list all containers of this workload by name,
	// and override data in case it is configured on the source.
	// this peoperty can be used to know all container names in the workload, even if other controllers did not yet run.
	ContainersOverrides []ContainerOverride `json:"containersOverrides,omitempty"`
	// An hash of the containers overrides, used to determine if the overrides have changed for event filtering.
	// this is updated only when the overrides are changed, and not when some other change in the source occurs.
	ContainerOverridesHash string `json:"containerOverridesHash,omitempty"`

	// this hash is used to determine the deployment of the agents.
	// e.g. when the distro for container changes, or it's compatibility version,
	// or something else that requires rollout, the hash change will indicate that.
	// if the hash is empty, it means that no agent should be enabled in any pod container.
	AgentsMetaHash string `json:"agentsMetaHash,omitempty"`

	// The last time at which the agents meta hash value was changed.
	// Pods created before this time may not be in alignment with the AgentsMetaHash.
	// e.g. can lack the odigos label, or have a different value.
	AgentsMetaHashChangedTime *metav1.Time `json:"agentsMetaHashChangedTime,omitempty"`

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

	// default configuration for collecting http headers, in case the instrumentation library does not provide a configuration.
	DefaultHeadersCollection *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`

	// default configuration for library tracing.
	DefaultTraceConfig *instrumentationrules.TraceConfig `json:"traceConfig,omitempty"`

	// list of the custom instrumentation probes the SDK should use.
	CustomInstrumentations *instrumentationrules.CustomInstrumentations `json:"customInstrumentations,omitempty"`
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
	AttributesAndSamplerRules []AttributesAndSamplerRule `json:"attributesAndSamplerRules,omitempty"`
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

	HeadersCollection *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`
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

// RuntimeDetailsByContainer will return a map containing runtime details for each container name present in the instrumented workload.
// The keys are container names. Each value can be nil in case we have no runtime details for this container
// from automatic runtime detection or overrides.
// For each container, if an override is present, it will be taken into account before the automatic detection results.
func (ic *InstrumentationConfig) RuntimeDetailsByContainer() map[string]*RuntimeDetailsByContainer {
	detailsByContainer := make(map[string]*RuntimeDetailsByContainer)

	// ContainersOverrides will always list all containers of the workloads, so we can use it to iterate.
	for i := range ic.Spec.ContainersOverrides {
		containerName := ic.Spec.ContainersOverrides[i].ContainerName
		var containerRuntimeDetails *RuntimeDetailsByContainer
		// always take the override if it exists, before taking the automatic runtime detection.
		if ic.Spec.ContainersOverrides[i].RuntimeInfo != nil {
			containerRuntimeDetails = ic.Spec.ContainersOverrides[i].RuntimeInfo
		} else {
			// find this container by name in the automatic runtime detection
			for j := range ic.Status.RuntimeDetailsByContainer {
				if ic.Status.RuntimeDetailsByContainer[j].ContainerName == containerName {
					containerRuntimeDetails = &ic.Status.RuntimeDetailsByContainer[j]
					break
				}
			}
		}
		detailsByContainer[containerName] = containerRuntimeDetails
	}

	return detailsByContainer
}

func (ic *InstrumentationConfig) GetOverridesForContainer(containerName string) *ContainerOverride {
	// list of containers is short (usually one, but can realistically be as much as 5) so linear search is fine.
	for i := range ic.Spec.ContainersOverrides {
		if ic.Spec.ContainersOverrides[i].ContainerName == containerName {
			return &ic.Spec.ContainersOverrides[i]
		}
	}
	return nil
}
