package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
)

const ActionNamePiiMasking = "PiiMasking"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type PiiMaskingConfig struct {
	actionsapi.PiiMaskingConfig `json:",inline"`
}

func (PiiMaskingConfig) ProcessorType() string {
	return "redaction"
}

func (PiiMaskingConfig) OrderHint() int {
	return 1
}

func (PiiMaskingConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
