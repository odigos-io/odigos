package api

// ContainerCollectorConfig is a configuration for a specific container in a workload.
// +kubebuilder:object:generate=true
type ContainerCollectorConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`

	// The sampling configuration that relevant for the collector (tailsampling).
	TailSampling *SamplingCollectorConfig `json:"samplingCollectorConfig,omitempty"`

	UrlTemplatization *UrlTemplatizationConfig `json:"urlTemplatization,omitempty"`

	// Later we can add here any relevant collector configuration in the scope of the container.
	// e.g url-templatization
}

// noisy operation configuration used by the instrumentation config.
// it is similar to the NoisyOperation struct, but includes a rule id and excludes irrelevant fields
// the original struct cannot be used as the id property is internal and should not appear in user-facing API.
// +kubebuilder:object:generate=true
type WorkloadNoisyOperation struct {
	Id               string                        `json:"id"`
	Operation        *HeadSamplingOperationMatcher `json:"operation,omitempty"`
	PercentageAtMost *float64                      `json:"percentageAtMost,omitempty"`
}

// highly relevant operation configuration used by the instrumentation config.
// it is similar to the HighlyRelevantOperation struct, but includes a rule id and excludes irrelevant fields
// the original struct cannot be used as the id property is internal and should not appear in user-facing API.
// +kubebuilder:object:generate=true
type WorkloadHighlyRelevantOperation struct {
	Id                string                        `json:"id"`
	Error             bool                          `json:"error,omitempty"`
	DurationAtLeastMs *int                          `json:"durationAtLeastMs,omitempty"`
	Operation         *TailSamplingOperationMatcher `json:"operation,omitempty"`
	PercentageAtLeast *float64                      `json:"percentageAtLeast,omitempty"`
}

// cost reduction rule configuration used by the instrumentation config.
// it is similar to the CostReductionRule struct, but includes a rule id and excludes irrelevant fields
// the original struct cannot be used as the id property is internal and should not appear in user-facing API.
// +kubebuilder:object:generate=true
type WorkloadCostReductionRule struct {
	Id               string                        `json:"id"`
	Operation        *TailSamplingOperationMatcher `json:"operation,omitempty"`
	PercentageAtMost float64                       `json:"percentageAtMost"`
}

// +kubebuilder:object:generate=true
type SamplingCollectorConfig struct {
	NoisyOperations          []WorkloadNoisyOperation          `json:"noisyOperations,omitempty"`
	HighlyRelevantOperations []WorkloadHighlyRelevantOperation `json:"highlyRelevantOperations,omitempty"`
	CostReductionRules       []WorkloadCostReductionRule       `json:"costReductionRules,omitempty"`
}

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Rule is the template rule to be applied to URLs
	Rules []string `json:"templatizationRules,omitempty"`
}
