package traces

import (
	"slices"

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
	slices.Sort(result)
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
		SkipHttpStatusCodes:    append(c1.SkipHttpStatusCodes, c2.SkipHttpStatusCodes...),
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
	var templates []string

	// if at least one rule group or default templatization config matches, the container participates.
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
					configForDefaultTemplatization = mergeDefaultTemplatizationConfigs(configForDefaultTemplatization, &defaultTemplatization.DefaultTemplatizationConfig)
				}
			}
		}

		for _, rules := range action.Spec.URLTemplatization.Rules {
			if scope.SourceScopeMatchesContainer(rules.Scopes, pw, language) {
				participating = true
				templates = append(templates, rules.Templates...)
			}
		}
	}

	// if not explicitly disabled, the default templatization will be applied.
	// set it to empty config to align with the common api conventions.
	if configForDefaultTemplatization == nil {
		configForDefaultTemplatization = &actions.DefaultTemplatizationConfig{}
	}

	// if not explicitly disabled, the container participates in templatization.
	if !configForDefaultTemplatization.Disabled {
		participating = true
	}

	// if not participating, return nil to disable it entirely for this container.
	if !participating {
		return nil
	}

	// if multiple actions set a skip policy status codes, dedupe them to keep just one of each status code.
	if configForDefaultTemplatization.SkipPolicy != nil {
		configForDefaultTemplatization.SkipPolicy.SkipHttpStatusCodes = dedupeStatusCodes(
			configForDefaultTemplatization.SkipPolicy.SkipHttpStatusCodes,
		)
	}

	// align default templatization config with the common api conventions.
	// e.g: if disabled, set default templatization to nil
	defaultTemplatization := configForDefaultTemplatization
	if configForDefaultTemplatization.Disabled {
		defaultTemplatization = nil
	}

	return &actions.UrlTemplatizationConfig{
		Templates: templates,
		Default:   defaultTemplatization,
	}
}
