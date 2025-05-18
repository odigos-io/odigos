package v__internal

import (
	"github.com/odigos-io/odigos/api/hub/odigos/v__internal/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:skipversion
// +kubebuilder:storageversion
// +kubebuilder:object:root=true
type InstrumentationConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstrumentationConfigSpec   `json:"spec,omitempty"`
	Status InstrumentationConfigStatus `json:"status,omitempty"`
}

type OtherAgent struct {
	Name string `json:"name,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ProcessingState string

type RuntimeDetailsByContainer struct {
	ContainerName           string                     `json:"containerName"`
	Language                common.ProgrammingLanguage `json:"language"`
	RuntimeVersion          string                     `json:"runtimeVersion,omitempty"`
	EnvVars                 []EnvVar                   `json:"envVars,omitempty"`
	OtherAgent              *OtherAgent                `json:"otherAgent,omitempty"`
	LibCType                *common.LibCType           `json:"libCType,omitempty"`
	SecureExecutionMode     *bool                      `json:"secureExecutionMode,omitempty"`
	CriErrorMessage         *string                    `json:"criErrorMessage,omitempty"`
	EnvFromContainerRuntime []EnvVar                   `json:"envFromContainerRuntime,omitempty"`
	RuntimeUpdateState      *ProcessingState           `json:"runtimeUpdateState,omitempty"`
}

type InstrumentationConfigStatus struct {
	RuntimeDetailsByContainer []RuntimeDetailsByContainer `json:"runtimeDetailsByContainer,omitempty"`
	Conditions                []metav1.Condition          `json:"conditions,omitempty" patchStrategy:"merge" protobuf:"bytes,1,rep,name=conditions"`
	WorkloadRolloutHash       string                      `json:"workloadRolloutHash,omitempty"`
}

type AgentEnabledReason string

type ContainerAgentConfig struct {
	ContainerName       string             `json:"containerName"`
	AgentEnabled        bool               `json:"agentEnabled"`
	AgentEnabledReason  AgentEnabledReason `json:"agentEnabledReason,omitempty"`
	AgentEnabledMessage string             `json:"agentEnabledMessage,omitempty"`
	OtelDistroName      string             `json:"otelDistroName,omitempty"`
	DistroParams        map[string]string  `json:"distroParams,omitempty"`
}

type InstrumentationConfigSpec struct {
	ServiceName           string                 `json:"serviceName,omitempty"`
	AgentInjectionEnabled bool                   `json:"agentInjectionEnabled"`
	Containers            []ContainerAgentConfig `json:"containers,omitempty"`
	AgentsMetaHash        string                 `json:"agentsMetaHash,omitempty"`
	SdkConfigs            []SdkConfig            `json:"sdkConfigs,omitempty"`
}

type SdkConfig struct {
	Language                      common.ProgrammingLanguage                  `json:"language"`
	InstrumentationLibraryConfigs []InstrumentationLibraryConfig              `json:"instrumentationLibraryConfigs,omitempty"`
	HeadSamplingConfig            *HeadSamplingConfig                         `json:"headSamplerConfig,omitempty"`
	DefaultPayloadCollection      *instrumentationrules.PayloadCollection     `json:"payloadCollection,omitempty"`
	DefaultCodeAttributes         *instrumentationrules.CodeAttributes        `json:"codeAttributes,omitempty"`
	DefaultHeadersCollection      *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`
}

type AttributeCondition struct {
	Key      string   `json:"key"`
	Val      string   `json:"val"`
	Operator Operator `json:"operator,omitempty"`
}

type Operator string

type AttributesAndSamplerRule struct {
	AttributeConditions []AttributeCondition `json:"attributeConditions"`
	Fraction            float64              `json:"fraction"`
}

type HeadSamplingConfig struct {
	AttributesAndSamplerRules []AttributesAndSamplerRule `json:"attributesAndSamplerRules"`
	FallbackFraction          float64                    `json:"fallbackFraction"`
}

type InstrumentationLibraryConfig struct {
	InstrumentationLibraryId InstrumentationLibraryId                    `json:"libraryId"`
	TraceConfig              *InstrumentationLibraryConfigTraces         `json:"traceConfig,omitempty"`
	PayloadCollection        *instrumentationrules.PayloadCollection     `json:"payloadCollection,omitempty"`
	CodeAttributes           *instrumentationrules.CodeAttributes        `json:"codeAttributes,omitempty"`
	HeadersCollection        *instrumentationrules.HttpHeadersCollection `json:"headersCollection,omitempty"`
}

type InstrumentationLibraryId struct {
	InstrumentationLibraryName string          `json:"libraryName"`
	SpanKind                   common.SpanKind `json:"spanKind,omitempty"`
}

type InstrumentationLibraryConfigTraces struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// +kubebuilder:object:root=true
type InstrumentationConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstrumentationConfig `json:"items"`
}

func (*InstrumentationConfig) Hub() {}
