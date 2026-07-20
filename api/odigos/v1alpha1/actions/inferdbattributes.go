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

func (InferDbAttributesConfig) OrderHint() int {
	return 1
}

func (InferDbAttributesConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
