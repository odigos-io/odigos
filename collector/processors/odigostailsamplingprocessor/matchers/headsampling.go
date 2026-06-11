package matchers

import (
	"strings"

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
	case operation.GrpcServer != nil:
		return newHeadSamplingGrpcServerMatcher(operation.GrpcServer)
	case operation.GrpcClient != nil:
		return newHeadSamplingGrpcClientMatcher(operation.GrpcClient)
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
	method              string
	serverAddress       string
	templatedPath       string
	templatedPathPrefix string
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

type headSamplingGrpcServerMatcher struct {
	method  string
	service string
}

func newHeadSamplingGrpcServerMatcher(operation *commonapisampling.HeadSamplingGrpcServerOperationMatcher) Matcher {
	return &headSamplingGrpcServerMatcher{
		method:  operation.Method,
		service: operation.Service,
	}
}

func (m *headSamplingGrpcServerMatcher) Match(span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindServer {
		return false
	}
	return matchGrpcMethodAndService(span, m.method, m.service)
}

type headSamplingGrpcClientMatcher struct {
	method        string
	service       string
	serverAddress string
}

func newHeadSamplingGrpcClientMatcher(operation *commonapisampling.HeadSamplingGrpcClientOperationMatcher) Matcher {
	return &headSamplingGrpcClientMatcher{
		method:        operation.Method,
		service:       operation.Service,
		serverAddress: operation.ServerAddress,
	}
}

func (m *headSamplingGrpcClientMatcher) Match(span ptrace.Span) bool {
	if span.Kind() != ptrace.SpanKindClient {
		return false
	}
	if !matchGrpcMethodAndService(span, m.method, m.service) {
		return false
	}
	if m.serverAddress != "" && !matchServerAddress(span, m.serverAddress) {
		return false
	}
	return true
}

// grpcRpcSystemValue is the well-known value of rpc.system / rpc.system.name for gRPC, identical
// across the v1.26 and the newer release-candidate OTel RPC semantic conventions.
const grpcRpcSystemValue = "grpc"

// matchGrpcMethodAndService matches a span against rule-supplied gRPC method/service.
// The span must look like a gRPC span — concretely: rpc.system / rpc.system.name (when present)
// must equal "grpc", and the span must carry at least rpc.method or rpc.service. The rpc.system
// check filters out spans from other RPC frameworks that share the rpc.* attribute namespace
// (Apache Dubbo, Connect RPC, JSON-RPC, .NET WCF, Java RMI, ONC RPC). Older instrumentations
// that don't set rpc.system are still considered (permissive default) and gated only by the
// rpc.method / rpc.service presence check.
// Rule fields are AND-ed; empty rule fields are wildcards.
func matchGrpcMethodAndService(span ptrace.Span, ruleMethod string, ruleService string) bool {
	if rpcSystem, found := getRpcSystem(span); found && !strings.EqualFold(rpcSystem, grpcRpcSystemValue) {
		return false
	}
	spanMethod, methodFound := getRpcMethod(span)
	spanService, serviceFound := getRpcService(span)
	if !methodFound && !serviceFound {
		return false
	}
	if ruleMethod != "" && spanMethod != ruleMethod {
		return false
	}
	if ruleService != "" && spanService != ruleService {
		return false
	}
	return true
}
