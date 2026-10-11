package traces

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsTracesSpanRenamer(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.SpanRenamer != nil && distro.Traces.SpanRenamer.Supported
}

func CalculateSpanRenamerConfig(distro *distro.OtelDistro, agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage) *actions.SpanRenamerConfig {

	if !DistroSupportsTracesSpanRenamer(distro) {
		return nil
	}

	gotRenamingConfig := false
	scopeRulesMap := map[string]actions.SpanRenamerScopeRules{}

	for _, action := range *agentLevelActions {
		if action.Spec.SpanRenamer != nil {
			if action.Spec.SpanRenamer.ProgrammingLanguage != language {
				continue
			}
			scopeName := action.Spec.SpanRenamer.ScopeName
			for _, scopeRule := range action.Spec.SpanRenamer.RegexReplacements {
				if existing, ok := scopeRulesMap[scopeName]; ok {
					existing.RegexReplacements = append(existing.RegexReplacements, scopeRule)
					scopeRulesMap[scopeName] = existing
				} else {
					scopeRulesMap[scopeName] = actions.SpanRenamerScopeRules{
						ScopeName:         scopeName,
						RegexReplacements: []actions.SpanRenamerRegexReplacement{scopeRule},
					}
				}
				gotRenamingConfig = true
			}
		}
	}

	if !gotRenamingConfig {
		return nil
	}

	scopeRules := []actions.SpanRenamerScopeRules{}
	for _, scopeRule := range scopeRulesMap {
		scopeRules = append(scopeRules, scopeRule)
	}
	return &actions.SpanRenamerConfig{
		ScopeRules: scopeRules,
	}
}
