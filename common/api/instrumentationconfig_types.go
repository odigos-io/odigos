package api

import (
	"github.com/odigos-io/odigos/common/api/sampling"
)

// ContainerCollectorConfig is a configuration for a specific container in a workload.
// +kubebuilder:object:generate=true
type ContainerCollectorConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`

	// The sampling configuration that relevant for the collector (tailsampling).
	TailSampling *sampling.TailSamplingSourceConfig `json:"samplingCollectorConfig,omitempty"`

	UrlTemplatization *UrlTemplatizationConfig `json:"urlTemplatization,omitempty"`

	// Later we can add here any relevant collector configuration in the scope of the container.
	// e.g url-templatization
}

// +kubebuilder:object:generate=true
type UrlTemplatizationConfig struct {
	// Template rules to apply to URLs
	TemplatizationRules []string `json:"templatizationRules,omitempty"`
}
