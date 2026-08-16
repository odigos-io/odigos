package urltemplate

import "strings"

// IsPathMatching reports whether path matches this rule.
// A leading '/' on path is ignored. All-static rules compare StaticPath as a string;
// otherwise the path is split and matched segment-by-segment.
func (r PathRule) IsPathMatching(path string) bool {
	if path != "" && path[0] == '/' {
		path = path[1:]
	}
	if r.AllStatic {
		return r.matchStaticPath(path)
	}
	var pathSegments []string
	if path == "" {
		pathSegments = []string{""}
	} else {
		pathSegments = strings.Split(path, "/")
	}
	return r.isPathSegmentsMatching(pathSegments)
}

// IsPathSegmentsMatching reports whether pathSegments matches this rule.
// Static segments must equal; wildcard and templated segments match any single segment.
// Exact requires the same number of segments; prefix matches when the path starts with the rule.
func (r PathRule) IsPathSegmentsMatching(pathSegments []string) bool {
	if r.AllStatic {
		return r.matchStaticPath(strings.Join(pathSegments, "/"))
	}
	return r.isPathSegmentsMatching(pathSegments)
}

func (r PathRule) isPathSegmentsMatching(pathSegments []string) bool {
	if r.Prefix {
		// prefix match, so the path must have at least as many segments as the rule
		if len(r.Segments) > len(pathSegments) {
			return false
		}
	} else {
		// exact match, so the number of segments must be the same
		if len(r.Segments) != len(pathSegments) {
			return false
		}
	}
	return r.matchSegments(pathSegments)
}

func (r PathRule) matchStaticPath(path string) bool {
	if !r.Prefix {
		return path == r.StaticPath
	}
	if r.StaticPath == "" {
		return true
	}
	if !strings.HasPrefix(path, r.StaticPath) {
		return false
	}
	return len(path) == len(r.StaticPath) || path[len(r.StaticPath)] == '/'
}

func (r PathRule) matchSegments(pathSegments []string) bool {
	for i, ruleSegment := range r.Segments {
		if ruleSegment.Wildcard || ruleSegment.TemplateName != "" {
			continue
		}
		if ruleSegment.StaticString != pathSegments[i] {
			return false
		}
	}
	return true
}
