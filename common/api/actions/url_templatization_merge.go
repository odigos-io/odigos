package actions

import (
	"cmp"
	"slices"
	"strings"
)

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

type segmentedUrlTemplate struct {
	template string
	segments []string
	ranks    []int
}

func dedupeAndSortTemplates(templates []string) []string {
	if len(templates) <= 1 {
		return templates
	}

	seen := make(map[string]struct{}, len(templates))
	segmented := make([]segmentedUrlTemplate, 0, len(templates))
	for _, template := range templates {
		if _, ok := seen[template]; ok {
			continue
		}
		seen[template] = struct{}{}
		parts := strings.Split(template, "/")
		ranks := make([]int, len(parts))
		for i, part := range parts {
			ranks[i] = urlTemplateSegmentRank(part)
		}
		segmented = append(segmented, segmentedUrlTemplate{
			template: template,
			segments: parts,
			ranks:    ranks,
		})
	}
	slices.SortFunc(segmented, compareUrlTemplates)

	result := make([]string, len(segmented))
	for i, item := range segmented {
		result[i] = item.template
	}
	return result
}

// compareUrlTemplates orders rules segment by segment so a more specific rule
// sorts before a less specific one at the first differing segment: static
// literals, then "*", then "{...}" templates. Same-kind segments are ordered
// lexicographically, and a shorter rule sorts before a longer one that extends it.
func compareUrlTemplates(a, b segmentedUrlTemplate) int {
	n := min(len(a.segments), len(b.segments))
	for i := 0; i < n; i++ {
		if c := cmp.Compare(a.ranks[i], b.ranks[i]); c != 0 {
			return c
		}
		if c := strings.Compare(a.segments[i], b.segments[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(a.segments), len(b.segments))
}

// urlTemplateSegmentRank ranks a segment from most to least specific.
func urlTemplateSegmentRank(segment string) int {
	switch {
	case segment == "*":
		return 1
	case strings.HasPrefix(segment, "{"):
		return 2
	default:
		return 0
	}
}

func mergeDefaultTemplatizationSkipPolicyConfigs(c1, c2 *DefaultTemplatizationSkipPolicyConfig) *DefaultTemplatizationSkipPolicyConfig {
	if c1 == nil {
		return c2
	}
	if c2 == nil {
		return c1
	}

	// both are not nil, merge them.
	skipForNonSuccessCodes := c1.SkipForNonSuccessCodes || c2.SkipForNonSuccessCodes
	if skipForNonSuccessCodes {
		return &DefaultTemplatizationSkipPolicyConfig{
			SkipForNonSuccessCodes: skipForNonSuccessCodes,
		}
	}

	return &DefaultTemplatizationSkipPolicyConfig{
		SkipForNonSuccessCodes: false,
		SkipHttpStatusCodes:    append(c1.SkipHttpStatusCodes, c2.SkipHttpStatusCodes...),
	}
}

// MergeDefaultTemplatizationConfigs combines the default templatization configs of two actions.
// Disabled wins over any other config, and skip policies are merged.
func MergeDefaultTemplatizationConfigs(c1, c2 *DefaultTemplatizationConfig) *DefaultTemplatizationConfig {
	if c1 == nil {
		return c2
	}
	if c2 == nil {
		return c1
	}

	// both are not nil, merge them.
	disabled := c1.Disabled || c2.Disabled
	if disabled {
		return &DefaultTemplatizationConfig{
			Disabled: disabled,
		}
	}

	return &DefaultTemplatizationConfig{
		Disabled:   disabled,
		SkipPolicy: mergeDefaultTemplatizationSkipPolicyConfigs(c1.SkipPolicy, c2.SkipPolicy),
	}
}

// BuildUrlTemplatizationConfig shapes the templates and default templatization config merged from
// all relevant actions into the config consumed by the url templatization processor.
// rulesMatched reports whether at least one rule group applies; a nil defaultConfig means default
// templatization is enabled. Returns nil when templatization should be disabled entirely.
func BuildUrlTemplatizationConfig(templates []string, defaultConfig *DefaultTemplatizationConfig, rulesMatched bool) *UrlTemplatizationConfig {
	// if at least one rule group or default templatization config matches, the source participates.
	participating := rulesMatched

	// if not explicitly disabled, the default templatization will be applied.
	// set it to empty config to align with the common api conventions.
	if defaultConfig == nil {
		defaultConfig = &DefaultTemplatizationConfig{}
	}

	// if not explicitly disabled, the source participates in templatization.
	if !defaultConfig.Disabled {
		participating = true
	}

	// if not participating, return nil to disable it entirely.
	if !participating {
		return nil
	}

	// if multiple actions set a skip policy status codes, dedupe them to keep just one of each status code.
	if defaultConfig.SkipPolicy != nil {
		defaultConfig.SkipPolicy.SkipHttpStatusCodes = dedupeStatusCodes(
			defaultConfig.SkipPolicy.SkipHttpStatusCodes,
		)
	}

	templates = dedupeAndSortTemplates(templates)

	// align default templatization config with the common api conventions.
	// e.g: if disabled, set default templatization to nil
	defaultTemplatization := defaultConfig
	if defaultConfig.Disabled {
		defaultTemplatization = nil
	}

	return &UrlTemplatizationConfig{
		Templates: templates,
		Default:   defaultTemplatization,
	}
}
