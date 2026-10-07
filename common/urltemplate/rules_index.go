package urltemplate

// AppendRuleBySegmentCount appends rule to m keyed by len(rule.Segments).
// Used for exact (non-prefix) rules so matching can look up only the bucket
// for a path's segment count.
func AppendRuleBySegmentCount(m map[int][]PathRule, rule PathRule) {
	n := len(rule.Segments)
	m[n] = append(m[n], rule)
}

// ParseRulesBySegmentCount parses exact-match (non-prefix) rule strings into a
// map keyed by segment count. Invalid rules are skipped.
func ParseRulesBySegmentCount(ruleStrings []string) map[int][]PathRule {
	parsed := make(map[int][]PathRule)
	for _, s := range ruleStrings {
		rule, err := ParseUserInputRuleString(s, false)
		if err != nil {
			continue
		}
		AppendRuleBySegmentCount(parsed, rule)
	}
	return parsed
}

// PathMatchesAnyRule reports whether path matches any exact rule in
// rulesBySegmentCount. Only the bucket for the path's segment count is checked.
func PathMatchesAnyRule(path string, rulesBySegmentCount map[int][]PathRule) bool {
	if len(rulesBySegmentCount) == 0 {
		return false
	}
	segments, _ := SplitPath(path)
	_, found := FindMatchingRule(segments, rulesBySegmentCount)
	return found
}

// FindMatchingRule returns the first exact rule in rulesBySegmentCount that
// matches pathSegments. Only the bucket for len(pathSegments) is checked.
func FindMatchingRule(pathSegments []string, rulesBySegmentCount map[int][]PathRule) (PathRule, bool) {
	for _, rule := range rulesBySegmentCount[len(pathSegments)] {
		if rule.IsPathSegmentsMatching(pathSegments) {
			return rule, true
		}
	}
	return PathRule{}, false
}
