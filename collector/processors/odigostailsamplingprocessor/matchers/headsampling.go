package matchers

import (
	commonapisanpling "github.com/odigos-io/odigos/common/api/sampling"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func HeadSamplingOperationMatcher(operation *commonapisanpling.HeadSamplingOperationMatcher, span ptrace.Span) bool {
	if operation == nil {
		// if operation is not specified, it will match any operation.
		return true
	}
	if operation.HttpServer != nil {
		return headSamplingOperationHttpServerMatcher(operation.HttpServer, span)
	}
	if operation.HttpClient != nil {
		return headSamplingOperationHttpClientMatcher(operation.HttpClient, span)
	}
	// no operation type specified, match any.
	return true
}

func headSamplingOperationHttpServerMatcher(operation *commonapisanpling.HeadSamplingHttpServerOperationMatcher, span ptrace.Span) bool {

	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	if !found {
		return false
	}

	if operation.Method != "" && !compareHttpMethod(httpMethod, operation.Method) {
		return false
	}

	if (operation.Route != "" || operation.RoutePrefix != "") && !matchHttpRoute(span, operation.Route, operation.RoutePrefix) {
		return false
	}

	return true
}

func headSamplingOperationHttpClientMatcher(operation *commonapisanpling.HeadSamplingHttpClientOperationMatcher, span ptrace.Span) bool {

	// this matcher is for http client operations only, and only client spans are considered.
	if span.Kind() != ptrace.SpanKindClient {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	if !found {
		return false
	}
	if operation.Method != "" && !compareHttpMethod(httpMethod, operation.Method) {
		return false
	}

	if operation.ServerAddress != "" && !matchServerAddress(span, operation.ServerAddress) {
		return false
	}

	if (operation.Route != "" || operation.RoutePrefix != "") && !matchHttpRoute(span, operation.Route, operation.RoutePrefix) {
		return false
	}

	return true
}
