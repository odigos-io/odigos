package matchers

import (
	"go.opentelemetry.io/collector/pdata/ptrace"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

func NewHeadSamplingOperationMatcher(operation *commonapisampling.HeadSamplingOperationMatcher) Matcher {
	if operation == nil {
		return anyMatcher{}
	}
	switch {
	case operation.HttpServer != nil:
		return newHeadSamplingHttpServerMatcher(operation.HttpServer)
	case operation.HttpClient != nil:
		return newHeadSamplingHttpClientMatcher(operation.HttpClient)
	default:
		return anyMatcher{}
	}
}

type headSamplingHttpServerMatcher struct {
	method      string
	route       string
	routePrefix string
}

func newHeadSamplingHttpServerMatcher(operation *commonapisampling.HeadSamplingHttpServerOperationMatcher) Matcher {
	return &headSamplingHttpServerMatcher{
		method:      operation.Method,
		route:       operation.Route,
		routePrefix: operation.RoutePrefix,
	}
}

func (m *headSamplingHttpServerMatcher) Match(span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindServer {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	switch {
	case !found:
		return false
	case m.method != "" && !compareHttpMethod(httpMethod, m.method):
		return false
	case (m.route != "" || m.routePrefix != "") && !matchHttpRoute(span, m.route, m.routePrefix):
		return false
	default:
		return true
	}
}

type headSamplingHttpClientMatcher struct {
	method                string
	serverAddress         string
	templatedPath         string
	templatedPathPrefix   string
}

func newHeadSamplingHttpClientMatcher(operation *commonapisampling.HeadSamplingHttpClientOperationMatcher) Matcher {
	return &headSamplingHttpClientMatcher{
		method:              operation.Method,
		serverAddress:       operation.ServerAddress,
		templatedPath:       operation.TemplatedPath,
		templatedPathPrefix: operation.TemplatedPathPrefix,
	}
}

func (m *headSamplingHttpClientMatcher) Match(span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindClient {
		return false
	}

	httpMethod, found := getHttpMethod(span)
	switch {
	case !found:
		return false
	case m.method != "" && !compareHttpMethod(httpMethod, m.method):
		return false
	case m.serverAddress != "" && !matchServerAddress(span, m.serverAddress):
		return false
	case (m.templatedPath != "" || m.templatedPathPrefix != "") && !matchTemplatedPath(span, m.templatedPath, m.templatedPathPrefix):
		return false
	default:
		return true
	}
}
