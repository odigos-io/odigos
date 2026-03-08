package matchers

import (
	"strings"

	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv_1_4_0 "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func getHttpMethod(span ptrace.Span) (string, bool) {
	httpMethod, found := span.Attributes().Get(string(semconv.HTTPRequestMethodKey))
	if found {
		return httpMethod.Str(), true
	}
	httpMethod, found = span.Attributes().Get(string(semconv_1_4_0.HTTPMethodKey))
	if found {
		return httpMethod.Str(), true
	}
	return "", false
}

// given a non-empty http method extracted from span, and a non-empty http method from rule, will attempt to match it.
// the matching is case-insensitive.
// will return true if there is a match.
func compareHttpMethod(spanMethod string, ruleMethod string) bool {
	return strings.EqualFold(ruleMethod, spanMethod)
}

func getHttpRoute(span ptrace.Span) (string, bool) {
	httpRoute, found := span.Attributes().Get(string(semconv.HTTPRouteKey))
	if found {
		return httpRoute.Str(), true
	}
	return "", false
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

// given a span, will attempt to match it to a route rules based on:
// - http.route attribute (if present)
// - url.path attribute (if present)
// - old http.target attribute for agents not yet migrated to the new semconv (if present)
// if no attribute is found to match the rule, it will return false (no match).
// route matching is based on exact match and prefix match.
func matchHttpRoute(span ptrace.Span, ruleRouteExact string, ruleRoutePrefix string) bool {
	if ruleRouteExact == "" && ruleRoutePrefix == "" { // (should have been checked by caller, but just in case.)
		// unset means match any route
		return true
	}

	httpRoute, found := getHttpRoute(span)
	if found {
		return compareHttpRoute(httpRoute, ruleRouteExact, ruleRoutePrefix)
	}

	httpPath, found := getHttpServerPath(span)
	if found {
		return comparePathToHttpRoute(httpPath, ruleRouteExact, ruleRoutePrefix)
	}

	return false // no attribute found and the rule requires a match, so no match.
}

func getHttpServerPath(span ptrace.Span) (string, bool) {
	httpPath, found := span.Attributes().Get(string(semconv.URLPathKey))
	if found {
		return httpPath.Str(), true
	}
	// fallback to the old semconv http.target attribute.
	// it is not templated, and also contains the query string, which should be fixed.
	httpPath, found = span.Attributes().Get(string(semconv_1_4_0.HTTPTargetKey))
	if found {
		return httpPath.Str(), true
	}
	return "", false
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

func getServerAddress(span ptrace.Span) (string, bool) {
	serverAddress, found := span.Attributes().Get(string(semconv.ServerAddressKey))
	if found {
		return serverAddress.Str(), true
	}
	return "", false
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
