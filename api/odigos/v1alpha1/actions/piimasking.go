package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

const ActionNamePiiMasking = "PiiMasking"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PiiMaskingConfig struct {

	// the scope of services for which this masking config will be applied.
	// if empty, the provided config will be applied to all sources.
	Scopes *k8sconsts.SourcesScopes `json:"scopes,omitempty"`

	actionsapi.PiiMaskingConfig `json:",inline"`
}

func (PiiMaskingConfig) ProcessorType() string {
	return consts.OdigosPiiMaskingProcessorType
}

func (PiiMaskingConfig) OrderHint() int {
	return 1
}

func (PiiMaskingConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
