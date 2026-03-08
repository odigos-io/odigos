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

func comparePathToHttpRoute(path string, routeExactMatch string, routePrefix string) bool {
	if routeExactMatch != "" {
		// todo: we should do templated comparison here.
		return routeExactMatch == path
	}
	if routePrefix != "" {
		// todo: we should do templated comparison here.
		return strings.HasPrefix(path, routePrefix)
	}
	return false
}
