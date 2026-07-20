package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

const ActionNameDbQueryTemplatization = "DbQueryTemplatization"

// DbQueryTemplatizationConfig is the action config for templatizing database query text
// (e.g. replacing SQL literals with placeholders to reduce cardinality).
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type DbQueryTemplatizationConfig struct {
	// the scope of services for which this templatization config will be applied.
	// if empty, the provided config will be applied to all sources.
	Scopes *k8sconsts.SourcesScopes `json:"scopes,omitempty"`

	actionsapi.DbQueryTemplatizationConfig `json:",inline"`
}

func (DbQueryTemplatizationConfig) ProcessorType() string {
	return consts.OdigosSQLQueryProcessorType
}

func (DbQueryTemplatizationConfig) OrderHint() int {
	return 1
}

func (DbQueryTemplatizationConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
