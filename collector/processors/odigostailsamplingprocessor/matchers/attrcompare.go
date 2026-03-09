package matchers

import "strings"

// given a non-empty http method extracted from span, and a non-empty http method from rule, will attempt to match it.
// the matching is case-insensitive.
// will return true if there is a match.
func compareHttpMethod(spanMethod string, ruleMethod string) bool {
	return strings.EqualFold(ruleMethod, spanMethod)
}

// compare the http route attribute to the rule route(s).
// will return true if there is a match.
func compareHttpRoute(spanRoute string, ruleRouteExact string, ruleRoutePrefix string) bool {
	if ruleRouteExact != "" {
		return ruleRouteExact == spanRoute
	}
	if ruleRoutePrefix != "" {
		return strings.HasPrefix(spanRoute, ruleRoutePrefix)
	}
	// both options are unset, so we consider this a match.
	return true
}

func comparePathToTemplate(path string, exactMatch string, prefix string) bool {
	if exactMatch != "" {
		// todo: we should do templated comparison here.
		return exactMatch == path
	}
	if prefix != "" {
		// todo: we should do templated comparison here.
		return strings.HasPrefix(path, prefix)
	}
	return false
}
