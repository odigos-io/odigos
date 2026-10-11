package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

const ActionNameExtractAttribute = "ExtractAttribute"

// ExtractAttributeConfig is the action config for the odigosextractattribute processor.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type ExtractAttributeConfig struct {
	actionsapi.ExtractAttributeConfig `json:",inline"`
}

func (ExtractAttributeConfig) ProcessorType() string {
	return consts.OdigosExtractAttributeProcessorType
}

// OrderHint is 2 so extraction runs after K8sAttributes (0) and before downstream
// transforms that may consume the extracted attributes.
func (ExtractAttributeConfig) OrderHint() int {
	return 2
}

func (ExtractAttributeConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
