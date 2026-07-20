package api

import (
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/api/sampling"
)

// ContainerCollectorConfig is a configuration for a specific container in a workload.
// +kubebuilder:object:generate=true
type ContainerCollectorConfig struct {
	// The name of the container to which this configuration applies.
	ContainerName string `json:"containerName"`

	// The sampling configuration that relevant for the collector (tailsampling).
	TailSampling *sampling.TailSamplingSourceConfig `json:"samplingCollectorConfig,omitempty"`

	UrlTemplatization *actions.UrlTemplatizationConfig `json:"urlTemplatization,omitempty"`

	DbQueryTemplatization *actions.DbQueryTemplatizationConfig `json:"dbQueryTemplatization,omitempty"`

	InferDbAttributes *actions.InferDbAttributesConfig `json:"inferDbAttributes,omitempty"`
}
