package status

import (
	"text/template"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// OdigosStatus is the root object for an Odigos status definition.
type OdigosStatus struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type Metadata struct {
	Name          string `yaml:"name"`
	OwnerResource string `yaml:"ownerResource,omitempty"`
	Scope         string `yaml:"scope,omitempty"`
	Component     string `yaml:"component,omitempty"`
}

type Spec struct {
	Type    string   `yaml:"type"`
	Docs    Docs     `yaml:"docs,omitempty"`
	Reasons []Reason `yaml:"reasons"`
}

type Docs struct {
	Title       string     `yaml:"title"`
	Summary     string     `yaml:"summary"`
	Description string     `yaml:"description,omitempty"`
	States      []StateDoc `yaml:"states,omitempty"`
}

// StateDoc documents a reason state value (e.g. enabled / disabled) for a status type.
type StateDoc struct {
	State   string `yaml:"state"`
	Summary string `yaml:"summary"`
}

// Reason is a concrete status reason. When a YAML reason defines states, the
// generator expands each state into its own Reason with State set and
// state-specific Title/Message/Summary applied.
type Reason struct {
	Name               string                 `yaml:"name"`
	Title              string                 `yaml:"title,omitempty"`
	Summary            string                 `yaml:"summary"`
	Description        string                 `yaml:"description,omitempty"`
	Message            string                 `yaml:"message,omitempty"`
	State              string                 `yaml:"state,omitempty"`
	K8sConditionStatus metav1.ConditionStatus `yaml:"k8sConditionStatus,omitempty"`
	OdigosSeverity     OdigosSeverity         `yaml:"odigosSeverity"`
	ActionItems        []ActionItem           `yaml:"actionItems,omitempty"`
	States             []ReasonState          `yaml:"states,omitempty"`

	// Template is a pre-parsed Message template for runtime rendering.
	// It is not loaded from YAML; generated reasons set it via WithMessageTemplate.
	Template *template.Template `yaml:"-"`
}

type ReasonState struct {
	State   string `yaml:"state"`
	Title   string `yaml:"title,omitempty"`
	Message string `yaml:"message,omitempty"`
	Summary string `yaml:"summary,omitempty"`
}

type ActionItem struct {
	Type       ActionItemType `yaml:"type"`
	ButtonText string         `yaml:"buttonText"`
}

type ActionItemType string

const (
	ActionItemTypeRolloutWorkload ActionItemType = "RolloutWorkload"
)
