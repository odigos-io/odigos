package matchers

import (
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

func getHttpRoute(span ptrace.Span) (string, bool) {
	httpRoute, found := span.Attributes().Get(string(semconv.HTTPRouteKey))
	if found {
		return httpRoute.Str(), true
	}
	return "", false
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

func getServerAddress(span ptrace.Span) (string, bool) {
	serverAddress, found := span.Attributes().Get(string(semconv.ServerAddressKey))
	if found {
		return serverAddress.Str(), true
	}
	return "", false
}
