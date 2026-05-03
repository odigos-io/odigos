package traces

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

func DistroSupportsTracesUrlTemplatization(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.UrlTemplatization != nil && distro.Traces.UrlTemplatization.Supported
}

// CalculateUrlTemplatizationConfig filters template rules to only include those relevant to the container.
// A rule group is applied if its SourcesScope matches (empty scope = global, applies to all).
func CalculateUrlTemplatizationConfig(agentLevelActions *[]odigosv1.Action, containerName string, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *commonapi.UrlTemplatizationConfig {
	var rules []string
	participating := false

	for _, action := range *agentLevelActions {
		// Safety check: actions were already filtered to only include template actions.
		if action.Spec.URLTemplatization == nil {
			continue
		}

		for _, rulesGroup := range action.Spec.URLTemplatization.TemplatizationRulesGroups {
			if scope.AnySourceScopeMatchesContainer(rulesGroup.SourcesScope, pw, containerName, language) {
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

	return &commonapi.UrlTemplatizationConfig{
		TemplatizationRules: rules,
	}
}
