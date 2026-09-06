package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	"github.com/odigos-io/odigos/common/urltemplate"
)

// given a span, will attempt to match it to a route rule based on:
// - http.route attribute (if present)
// - url.path attribute (if present)
// - old http.target attribute for agents not yet migrated to the new semconv (if present)
// if no attribute is found to match the rule, it will return false (no match).
// exact vs prefix matching is recorded on the PathRule.
func matchHttpRoute(span ptrace.Span, rule urltemplate.PathRule) bool {
	if rule.Empty() { // (should have been checked by caller, but just in case.)
		// unset means match any route
		return true
	}

	httpRoute, found := getHttpRoute(span)
	if found {
		return compareHttpRoute(httpRoute, rule)
	}

	httpPath, found := getHttpServerPath(span)
	if found {
		return compareHttpRoute(httpPath, rule)
	}

	return false // no attribute found and the rule requires a match, so no match.
}

// given a span and a templated path rule, will attempt to match the span to the rule.
// will return true if there is a match.
// if the attribute is missing (and required on the rule), it will return false (no match).
func matchTemplatedPath(span ptrace.Span, rule urltemplate.PathRule) bool {
	if rule.Empty() { // (should have been checked by caller, but just in case.)
		// unset means match any path
		return true
	}

	urlTemplate, found := getHttpTemplatedPath(span)
	if found {
		// best case scenario (like if url templatization was run prior to the sampling)
		return compareHttpRoute(urlTemplate, rule)
	}

	// TODO: extract the path from either url.full or http.target attributes and compare to the rule.
	return false
}

// given a span and a non-empty server address, will attempt to match it to the span attributes.
// will return true if there is a match.
// if the attribute is missing (and requied on the rule), it will return false (no match).
func matchServerAddress(span ptrace.Span, ruleServerAddress string) bool {
	serverAddress, found := getServerAddress(span)
	if found {
		return serverAddress == ruleServerAddress
	}
	return false
}
