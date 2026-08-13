package urltemplate

// Match reports whether pathSegments matches this rule exactly (same number of segments).
// Static segments must equal; wildcard and templated segments match any single segment.
func (r PathRule) Match(pathSegments []string) bool {
	if len(r) != len(pathSegments) {
		return false
	}
	return r.matchSegments(pathSegments)
}

// MatchPrefix reports whether pathSegments starts with this rule.
func (r PathRule) MatchPrefix(pathSegments []string) bool {
	if len(r) == 0 || len(r) > len(pathSegments) {
		return false
	}
	return r.matchSegments(pathSegments[:len(r)])
}

func (r PathRule) matchSegments(pathSegments []string) bool {
	for i, ruleSegment := range r {
		if ruleSegment.Wildcard || ruleSegment.TemplateName != "" {
			continue
		}
		if ruleSegment.StaticString != pathSegments[i] {
			return false
		}
	}
	return true
}
