package matchers

import (
	"strings"

	"github.com/odigos-io/odigos/common/urltemplate"
)

// given a non-empty http method extracted from span, and a non-empty http method from rule, will attempt to match it.
// the matching is case-insensitive.
// will return true if there is a match.
func compareHttpMethod(spanMethod string, ruleMethod string) bool {
	return strings.EqualFold(ruleMethod, spanMethod)
}

// compare the http route attribute to the path rule.
// Matching is segment-based (same as url templatization): static segments must equal;
// "*" and "{name}" match any single segment. Exact requires the same segment count;
// prefix matches when the path starts with the rule segments (PathRule.Prefix).
func compareHttpRoute(spanRoute string, rule urltemplate.PathRule) bool {
	if rule.Empty() {
		return true
	}
	return rule.IsPathMatching(spanRoute)
}
