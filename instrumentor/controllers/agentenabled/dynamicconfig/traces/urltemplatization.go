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
	var rules []string
	participating := false
	avoidDefaultTemplatizationOnError := false

	for _, action := range *agentLevelActions {
		// Safety check: actions were already filtered to only include template actions.
		if action.Spec.URLTemplatization == nil {
			continue
		}

		if action.Spec.URLTemplatization.AvoidDefaultTemplatizationOnError != nil {
			if scope.SourceScopeMatchesContainer(action.Spec.URLTemplatization.AvoidDefaultTemplatizationOnError.SourcesScopes, pw, language) {
				avoidDefaultTemplatizationOnError = true
			}
		}

		for _, rulesGroup := range action.Spec.URLTemplatization.TemplatizationRulesGroups {
			if scope.SourceScopeMatchesContainer(rulesGroup.SourcesScopes, pw, language) {
				participating = true
				for _, rule := range rulesGroup.TemplatizationRules {
					rules = append(rules, rule.Template)
				}
			}
		}
	}

	// container can participate in templatization and have no rule.
	// if at least one rule group matches, the container participates.
	if !participating {
		return nil
	}

	return &actions.UrlTemplatizationConfig{
		TemplatizationRules:               rules,
		AvoidDefaultTemplatizationOnError: avoidDefaultTemplatizationOnError,
	}
}
