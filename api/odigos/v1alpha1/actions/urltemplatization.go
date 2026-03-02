package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
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

// WorkloadFilter allows filtering rule groups by k8s workload.
// Kind and Name are AND'd together within a single filter entry.
// Multiple WorkloadFilter entries in a group are OR'd together.
// An empty WorkloadFilters list matches all workloads.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type WorkloadFilter struct {
	// +kubebuilder:validation:Enum=Deployment;StatefulSet;DaemonSet
	Kind *k8sconsts.WorkloadKind `json:"kind,omitempty"`
	Name string                  `json:"name,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
// UrlTemplatizationRulesGroup is a group of rules that share the same target spans.
// All set filters are AND'd together; an unset filter matches everything.
// FilterK8sNamespace is an AND gate (empty = all namespaces).
// WorkloadFilters list is an OR gate (empty list = all workloads; each entry ANDs kind + name).
// FilterK8sWorkloadKind and FilterK8sWorkloadName are deprecated scalar equivalents of WorkloadFilters,
// kept for backward compatibility with existing CRs; they are ORed with WorkloadFilters entries.
// If no filters are set at all, the rules will be applied to all spans.
type UrlTemplatizationRulesGroup struct {
	FilterProgrammingLanguage *common.ProgrammingLanguage `json:"filterProgrammingLanguage,omitempty"`
	FilterK8sNamespace        string                      `json:"filterK8sNamespace,omitempty"`
	FilterK8sWorkloadKind     *k8sconsts.WorkloadKind     `json:"filterK8sWorkloadKind,omitempty"`
	FilterK8sWorkloadName     string                      `json:"filterK8sWorkloadName,omitempty"`

	// WorkloadFilters is a list of workload (kind, name) pairs this group targets.
	// Each entry is ORed: the group matches if the workload matches any entry.
	// Within an entry, set fields are ANDed (both kind and name must match if both are set).
	// If the list is empty, the group matches all workloads (subject to FilterK8sNamespace).
	WorkloadFilters []WorkloadFilter `json:"workloadFilters,omitempty"`
	// the rules that will be applied to the spans matching the above filters.
	TemplatizationRules []URLTemplatizationRule `json:"templatizationRules,omitempty"`

	// user can add notes about this group for future maintenance purposes. not used by the system.
	// it can record why this group was added and what endpoints it's targeting.
	Notes string `json:"notes,omitempty"`
}

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type URLTemplatizationConfig struct {

	// list here all the groups of rules that will be applied to the spans.
	// each group targets a specific set of spans that share the same filters.
	// for example, one can set up multiple groups in the action:
	// 1. some rules for deployment foo in namespace default
	// 2. rules without filters that will be applied to all spans.
	// +kubebuilder:validation:MinItems=1
	TemplatizationRulesGroups []UrlTemplatizationRulesGroup `json:"templatizationRulesGroups"`
}

func (URLTemplatizationConfig) ProcessorType() string {
	return "odigosurltemplate"
}

func (URLTemplatizationConfig) OrderHint() int {
	return 1
}

func (URLTemplatizationConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleNodeCollector}
}
