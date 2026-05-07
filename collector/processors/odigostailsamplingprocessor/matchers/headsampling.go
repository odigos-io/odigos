package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func HeadSamplingOperationMatcher(operation *commonapisampling.HeadSamplingOperationMatcher, span ptrace.Span) bool {
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

func headSamplingOperationHttpServerMatcher(operation *commonapisampling.HeadSamplingHttpServerOperationMatcher, span ptrace.Span) bool {

	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	switch {
	case !found:
		return false
	case operation.Method != "" && !compareHttpMethod(httpMethod, operation.Method):
		return false
	case (operation.Route != "" || operation.RoutePrefix != "") && !matchHttpRoute(span, operation.Route, operation.RoutePrefix):
		return false
	default:
		return true
	}
}

func headSamplingOperationHttpClientMatcher(operation *commonapisampling.HeadSamplingHttpClientOperationMatcher, span ptrace.Span) bool {

	// this matcher is for http client operations only, and only client spans are considered.
	if span.Kind() != ptrace.SpanKindClient {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	switch {
	case !found:
		return false
	case operation.Method != "" && !compareHttpMethod(httpMethod, operation.Method):
		return false
	case operation.ServerAddress != "" && !matchServerAddress(span, operation.ServerAddress):
		return false
	case (operation.TemplatedPath != "" || operation.TemplatedPathPrefix != "") && !matchTemplatedPath(span, operation.TemplatedPath, operation.TemplatedPathPrefix):
		return false
	default:
		return true
	}
}
