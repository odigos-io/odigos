package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
)

const ActionNameURLTemplatization = "URLTemplatization"

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type URLTemplatizationRule struct {

	// this is the instructions on how to match and templatize a url for this rule.
	Template string `json:"template"`

	// user can populate examples of the urls that were observed.
	// when someone review this rule in the future, this can be helpful to understand and maintain it.
	// this field is optional and can be kept empty.
	Examples []string `json:"examples,omitempty"`

	// notes about why this rule was added and what it's purpose is.
	// only for human consumption and maintenance purposes. not used by the system.
	Notes string `json:"notes,omitempty"`
}

// UrlTemplatizationRulesGroup is a group of rules that share the same target spans.
// If SourcesScope is empty, the rules apply to all sources (global).
// If set, rules apply to selected scope.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type UrlTemplatizationRulesGroup struct {
	// SourcesScope selects which sources (workloads / containers / languages) the rules apply to.
	// Empty list means "all sources" (global rules).
	SourcesScope []k8sconsts.SourcesScope `json:"sourcesScope,omitempty"`

	// the rules that will be applied to the spans matching the above filters.
	TemplatizationRules []URLTemplatizationRule `json:"templatizationRules,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type URLTemplatizationConfig struct {

	// list here all the groups of rules that will be applied to the spans.
	// each group targets a specific set of spans that share the same filters.
	// for example, one can set up 3 groups in the action:
	// 1. some rules for java spans
	// 2. some rules for deployment foo in namespace default
	// 3. rules without filters that will be applied to all spans.
	TemplatizationRulesGroups []UrlTemplatizationRulesGroup `json:"templatizationRulesGroups"`
}

func (URLTemplatizationConfig) ProcessorType() string {
	return "odigosurltemplate"
}

func (URLTemplatizationConfig) OrderHint() int {
	return 1
}

func (URLTemplatizationConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}
