package traces

import (
	"sort"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// CalculatePiiMaskingConfig merges matching PiiMasking Actions for a container.
// Categories and custom masking rules are unioned across matching actions.
// Returns nil when no matching action contributes any masking rules.
func CalculatePiiMaskingConfig(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *actions.PiiMaskingConfig {
	seenCategories := make(map[actions.PiiCategory]struct{})
	seenFormats := make(map[string]struct{})
	seenRegexes := make(map[string]struct{})
	cfg := actions.PiiMaskingConfig{}

	for _, action := range *agentLevelActions {
		if action.Spec.PiiMasking == nil {
			continue
		}
		if !scope.SourceScopeMatchesContainer(action.Spec.PiiMasking.Scopes, pw, language) {
			continue
		}

		for _, category := range action.Spec.PiiMasking.PiiCategories {
			if _, ok := seenCategories[category]; ok {
				continue
			}
			seenCategories[category] = struct{}{}
			cfg.PiiCategories = append(cfg.PiiCategories, category)
		}

		for _, masking := range action.Spec.PiiMasking.CustomFormatMaskings {
			if masking.LookupKey == "" || masking.DataFormat == "" {
				continue
			}
			key := masking.LookupKey + "\x00" + string(masking.DataFormat)
			if _, ok := seenFormats[key]; ok {
				continue
			}
			seenFormats[key] = struct{}{}
			cfg.CustomFormatMaskings = append(cfg.CustomFormatMaskings, masking)
		}

		for _, masking := range action.Spec.PiiMasking.CustomRegexMaskings {
			if masking.Regex == "" {
				continue
			}
			if _, ok := seenRegexes[masking.Regex]; ok {
				continue
			}
			seenRegexes[masking.Regex] = struct{}{}
			cfg.CustomRegexMaskings = append(cfg.CustomRegexMaskings, masking)
		}
	}

	if len(cfg.PiiCategories) == 0 && len(cfg.CustomFormatMaskings) == 0 && len(cfg.CustomRegexMaskings) == 0 {
		return nil
	}

	sort.Slice(cfg.PiiCategories, func(i, j int) bool {
		return cfg.PiiCategories[i] < cfg.PiiCategories[j]
	})

	return &cfg
}
