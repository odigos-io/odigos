package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

const ActionNameInferDbAttributes = "InferDbAttributes"

// InferDbAttributesConfig is the action config for parsing database query text
// and adding attributes such as db.operation.name and db.collection.name.
// Configuration options will be added later.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type InferDbAttributesConfig struct {
	// the scope of services for which this config will be applied.
	// if empty, the provided config will be applied to all sources.
	Scopes *k8sconsts.SourcesScopes `json:"scopes,omitempty"`

	actionsapi.InferDbAttributesConfig `json:",inline"`
}

func (InferDbAttributesConfig) ProcessorType() string {
	return consts.OdigosSQLQueryProcessorType
}

// OrderHint is 1 so SQL query processing runs before spans reach the spanmetrics connector on the data-collector.
func (InferDbAttributesConfig) OrderHint() int {
	return 1
}

// CollectorRoles satisfies ActionConfig for generic action-backed Processor CRs.
// The shared SQL-query Processor uses SharedProcessorCollectorRoles(spanMetricsEnabled) instead.
func (InferDbAttributesConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}

// SharedProcessorCollectorRoles returns where the shared SQL-query Processor should run.
// When span metrics are enabled on the node collectors group, the processor must run on the node
// collector so inferred attributes / span names are set before span metrics record them.
func (InferDbAttributesConfig) SharedProcessorCollectorRoles(spanMetricsEnabled bool) []k8sconsts.CollectorRole {
	if spanMetricsEnabled {
		return []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleNodeCollector}
	}
	return []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleClusterGateway}
}
