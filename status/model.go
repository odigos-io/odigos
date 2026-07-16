package status

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// OdigosStatus is the root object for an Odigos status definition.
type OdigosStatus struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name string `yaml:"name"`
}

type Spec struct {
	Type    string   `yaml:"type"`
	Reasons []Reason `yaml:"reasons"`
}

type Reason struct {
	Name                 string                 `yaml:"name"`
	Message              string                 `yaml:"message"`
	TechnicalDescription string                 `yaml:"technicalDescription,omitempty"`
	K8sConditionStatus   metav1.ConditionStatus `yaml:"k8sConditionStatus,omitempty"`
	OdigosSeverity       OdigosSeverity         `yaml:"odigosSeverity"`
	ActionItems          []ActionItem           `yaml:"actionItems,omitempty"`
}

type ActionItem struct {
	Type           ActionItemType `yaml:"type"`
	UserFacingText string         `yaml:"userFacingText"`
}

type ActionItemType string

const (
	ActionItemTypeRolloutWorkload ActionItemType = "RolloutWorkload"
)
