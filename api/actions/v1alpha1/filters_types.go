package v1alpha1

import "github.com/odigos-io/odigos/api/k8sconsts"

const ActionNameFilters = "AttributeBasedFilters"

// FiltersConfig defines the configuration for the Filters action.
type AttributeBasedFiltersConfig struct {
	// Attributes is a map of attribute keys and values to filter on.
	// This will drop any spans, metrics, data points, or log records that contain the given attribute key and value,
	// based on what signal the Action is configured for.
	// This applies to both resource attributes and telemetry attributes. Multiple rules are ORed together.
	// For example, if you want to drop all spans that contain the attribute "url.path" with the value "/api/v1/users", you can set the following:
	// attributes:
	//   url.path: /api/v1/users
	// The value can be a string literal or a regex pattern.
	// Note that for traces, this can lead to apparent "gaps" in a trace when matching spans are filtered.
	Attributes map[string]string `json:"attributes,omitempty"`
}

func (AttributeBasedFiltersConfig) ProcessorType() string {
	return "filter"
}

func (AttributeBasedFiltersConfig) OrderHint() int {
	return 1
}

func (AttributeBasedFiltersConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
