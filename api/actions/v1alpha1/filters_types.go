package v1alpha1

import "github.com/odigos-io/odigos/api/k8sconsts"

const ActionNameFilters = "Filters"

// FiltersConfig defines the configuration for the Filters action.
type FiltersConfig struct {
	// Attributes is a map of attribute keys and values to filter on.
	Attributes map[string]string `json:"attributes,omitempty"`
	// ResourceAttributes is a map of resource attribute keys and values to filter on.
	ResourceAttributes map[string]string `json:"resource_attributes,omitempty"`
}

func (FiltersConfig) ProcessorType() string {
	return "filter"
}

func (FiltersConfig) OrderHint() int {
	return 1
}

func (FiltersConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
