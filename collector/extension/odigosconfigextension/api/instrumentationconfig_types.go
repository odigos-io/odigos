// Types in this file are copied from:
//   github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationconfig_types.go
// Keep in sync when the upstream API changes.

package api

// InstrumentationConfigSpec is a subset of InstrumentationConfig.spec used to unmarshal
// the spec (e.g. from unstructured) for the fields needed by the extension.
// Extra spec fields in the CR are ignored when unmarshaling.
type InstrumentationConfigSpec struct {
	Containers              []ContainerAgentSpec       `json:"containers,omitempty"`
	WorkloadCollectorConfig []ContainerCollectorConfig `json:"workloadCollectorConfig,omitempty"`
}

// ContainerAgentSpec is a subset of ContainerAgentConfig used to unmarshal container
// entries from InstrumentationConfig.spec.containers.
type ContainerAgentSpec struct {
	ContainerName string           `json:"containerName"`
	Traces        *AgentTracesSpec `json:"traces,omitempty"`
}

// AgentTracesSpec is a subset of AgentTracesConfig (traces config per container).
type AgentTracesSpec struct {
	HeadSampling *HeadSamplingConfig `json:"headSampling,omitempty"`
}

// HeadSamplingConfig holds head-sampling rules for a workload or container.
type HeadSamplingConfig struct {
	AttributesAndSamplerRules []AttributesAndSamplerRule `json:"attributesAndSamplerRules,omitempty"`
	FallbackFraction          float64                    `json:"fallbackFraction"`
}

// AttributesAndSamplerRule is a rule that matches span attributes and applies a sampling fraction.
type AttributesAndSamplerRule struct {
	AttributeConditions []AttributeCondition `json:"attributeConditions"`
	Fraction            float64              `json:"fraction"`
}

// AttributeCondition compares an attribute key to a value.
type AttributeCondition struct {
	Key      string `json:"key"`
	Val      string `json:"val"`
	Operator string `json:"operator,omitempty"`
}

// ContainerCollectorConfig is a configuration for a specific container in a workload.
type ContainerCollectorConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`
	// The sampling configuration relevant for the collector (tail sampling).
	TailSampling *SamplingCollectorConfig `json:"samplingCollectorConfig,omitempty"`
}

// SamplingCollectorConfig holds tail-sampling config for a container.
type SamplingCollectorConfig struct {
	NoisyOperations          []NoisyOperations         `json:"noisyOperations,omitempty"`
	HighlyRelevantOperations []HighlyRelevantOperation `json:"highlyRelevantOperations,omitempty"`
	CostReductionRules       []CostReductionRule       `json:"costReductionRules,omitempty"`
}
