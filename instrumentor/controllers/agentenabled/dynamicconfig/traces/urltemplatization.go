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

func dedupeStatusCodes(codes []int) []int {

	// short circuit if there are no or only one code
	if len(codes) <= 1 {
		return codes
	}

	seen := make(map[int]struct{}, len(codes))
	result := []int{}
	for _, code := range codes {
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

func mergeDefaultTemplatizationSkipPolicyConfigs(c1 *actions.DefaultTemplatizationSkipPolicyConfig, c2 *actions.DefaultTemplatizationSkipPolicyConfig) *actions.DefaultTemplatizationSkipPolicyConfig {
	if c1 == nil {
		return c2
	}
	if c2 == nil {
		return c1
	}

	// both are not nil, merge them.
	skipForNonSuccessCodes := c1.SkipForNonSuccessCodes || c2.SkipForNonSuccessCodes
	if skipForNonSuccessCodes {
		return &actions.DefaultTemplatizationSkipPolicyConfig{
			SkipForNonSuccessCodes: skipForNonSuccessCodes,
		}
	}

	return &actions.DefaultTemplatizationSkipPolicyConfig{
		SkipForNonSuccessCodes: false,
		StatusCodes:            append(c1.StatusCodes, c2.StatusCodes...),
	}
}

func mergeDefaultTemplatizationConfigs(c1 *actions.DefaultTemplatizationConfig, c2 *actions.DefaultTemplatizationConfig) *actions.DefaultTemplatizationConfig {
	if c1 == nil {
		return c2
	}
	if c2 == nil {
		return c1
	}

	// both are not nil, merge them.
	disabled := c1.Disabled || c2.Disabled
	if disabled {
		return &actions.DefaultTemplatizationConfig{
			Disabled: disabled,
		}
	}

	return &actions.DefaultTemplatizationConfig{
		Disabled:   disabled,
		SkipPolicy: mergeDefaultTemplatizationSkipPolicyConfigs(c1.SkipPolicy, c2.SkipPolicy),
	}
}

// CalculateUrlTemplatizationConfig filters template rules to only include those relevant to the container.
// A rule group is applied if its SourcesScope matches (empty scope = global, applies to all).
func CalculateUrlTemplatizationConfig(agentLevelActions *[]odigosv1.Action, containerName string, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *actions.UrlTemplatizationConfig {
	var rules []string

	// if at least one rule group or default templatization config matches, the container participates.
	participating := false

	// the combined default templatization config from all actions.
	// for the default templatization to take effect, at least one default templatization config must be set and match the container.
	// one can set a rule to apply default templatization on the entire cluster, or add more granular configs for specific scopes.
	// if this is nil, the default templatization will not be applied.
	var configForDefaultTemplatization *actions.DefaultTemplatizationConfig

	for _, action := range *agentLevelActions {
		// Safety check: actions were already filtered to only include template actions.
		if action.Spec.URLTemplatization == nil {
			continue
		}

		if action.Spec.URLTemplatization.DefaultTemplatizations != nil {
			for _, defaultTemplatization := range action.Spec.URLTemplatization.DefaultTemplatizations {
				participating = true
				if scope.SourceScopeMatchesContainer(defaultTemplatization.SourcesScopes, pw, language) {
					configForDefaultTemplatization = mergeDefaultTemplatizationConfigs(configForDefaultTemplatization, &defaultTemplatization.Config)
				}
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

	// replace disabled with nil to align with the source api conventions.
	if configForDefaultTemplatization != nil && configForDefaultTemplatization.Disabled {
		configForDefaultTemplatization = nil
	}

	// no templatization, return nil to disable it entirely.
	if configForDefaultTemplatization == nil && len(rules) == 0 {
		return nil
	}

	if configForDefaultTemplatization != nil && configForDefaultTemplatization.SkipPolicy != nil {
		configForDefaultTemplatization.SkipPolicy.StatusCodes = dedupeStatusCodes(
			configForDefaultTemplatization.SkipPolicy.StatusCodes,
		)
	}

	return &actions.UrlTemplatizationConfig{
		TemplatizationRules:   rules,
		DefaultTemplatization: configForDefaultTemplatization,
	}
}
