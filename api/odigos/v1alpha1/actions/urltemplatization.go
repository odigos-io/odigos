package actions

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/common/consts"
)

const ActionNameURLTemplatization = "URLTemplatization"

// UrlTemplatizationRule is a group of rules that share the same target spans.
// If SourcesScope is empty, the rules apply to all sources (global).
// If set, rules apply to selected scope.
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type UrlTemplatizationRule struct {
	// SourcesScope selects which sources (workloads / containers / languages) the rules apply to.
	// Empty list means "all sources" (global rules).
	Scopes *k8sconsts.SourcesScopes `json:"scopes,omitempty"`

	// the rules that will be applied to the spans matching the above filters.
	Templates []string `json:"templates,omitempty"`
}

// URLTemplatizationDefaultTemplatizationGroup is a group of services for which default templatization will be applied.
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type URLTemplatizationDefaultTemplatizationGroup struct {
	// the scope of services for which this templatization config will be applied.
	// if empty, the provided config will be applied to all sources.
	Scopes *k8sconsts.SourcesScopes `json:"scopes,omitempty"`

	// configurations for default templatization.
	// default templatization is applied on a single http span if none of the custom templatization rules matched.
	actionsapi.DefaultTemplatizationConfig `json:",inline"`
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
	Rules []UrlTemplatizationRule `json:"rules,omitempty"`

	// configurations for default templatization, on groups of services.
	// default templatization is applied on a single http span if none of the custom templatization rules matched.
	Default []URLTemplatizationDefaultTemplatizationGroup `json:"default,omitempty"`
}

func (URLTemplatizationConfig) ProcessorType() string {
	return consts.OdigosURLTemplateProcessorType
}

// OrderHint is 1 so URL templatization runs before spans reach the spanmetrics connector on the data-collector.
func (URLTemplatizationConfig) OrderHint() int {
	return 1
}

// CollectorRoles satisfies ActionConfig for generic action-backed Processor CRs.
// The shared URL-templatization Processor uses SharedProcessorCollectorRoles(spanMetricsEnabled) instead.
func (URLTemplatizationConfig) CollectorRoles() []k8sconsts.CollectorRole {
	return []k8sconsts.CollectorRole{
		k8sconsts.CollectorsRoleClusterGateway,
	}
}

// SharedProcessorCollectorRoles returns where the shared URL-templatization Processor should run.
// When span metrics are enabled on the node collectors group, the processor must run on the node
// collector so routes are templated before span metrics record span names and http.route.
func (URLTemplatizationConfig) SharedProcessorCollectorRoles(spanMetricsEnabled bool) []k8sconsts.CollectorRole {
	if spanMetricsEnabled {
		return []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleNodeCollector}
	}
	return []k8sconsts.CollectorRole{k8sconsts.CollectorsRoleClusterGateway}
}
