package traces

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

func DistroSupportsTracesUrlTemplatization(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.UrlTemplatization != nil && distro.Traces.UrlTemplatization.Supported
}

// CalculateUrlTemplatizationConfig filters template rules to only include those relevant to the container.
// A rule group is applied if its SourcesScope matches (empty scope = global, applies to all).
func CalculateUrlTemplatizationConfig(agentLevelActions *[]odigosv1.Action, containerName string, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *actions.UrlTemplatizationConfig {
	var templates []string

	// if at least one rule group matches, the container participates.
	participating := false

	// the combined default templatization config from all actions.
	// one can set a rule to apply default templatization on the entire cluster, or add more granular configs for specific scopes.
	// if this is nil (no specific config), the default templatization will be applied.
	var configForDefaultTemplatization *actions.DefaultTemplatizationConfig

	for _, action := range *agentLevelActions {
		// Safety check: actions were already filtered to only include template actions.
		if action.Spec.URLTemplatization == nil {
			continue
		}

		if action.Spec.URLTemplatization.Default != nil {
			for _, defaultTemplatization := range action.Spec.URLTemplatization.Default {
				if scope.SourceScopeMatchesContainer(defaultTemplatization.Scopes, pw, language) {
					configForDefaultTemplatization = actions.MergeDefaultTemplatizationConfigs(configForDefaultTemplatization, &defaultTemplatization.DefaultTemplatizationConfig)
				}
			}
		}

		for _, rules := range action.Spec.URLTemplatization.Rules {
			if scope.SourceScopeMatchesContainer(rules.Scopes, pw, language) {
				participating = true
				templates = append(templates, rules.Templates...)
				// DocumentedTemplates carry the same template strings as Templates, with
				// example paths and notes attached. They must be applied the same way.
				for _, documented := range rules.DocumentedTemplates {
					templates = append(templates, documented.Template)
				}
			}
		}
	}

	return actions.BuildUrlTemplatizationConfig(templates, configForDefaultTemplatization, participating)
}
