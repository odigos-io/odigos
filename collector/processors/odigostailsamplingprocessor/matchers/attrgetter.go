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

func getHttpTemplatedPath(span ptrace.Span) (string, bool) {
	httpTemplatedPath, found := span.Attributes().Get(string(semconv.URLTemplateKey))
	if found {
		return httpTemplatedPath.Str(), true
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

// getRpcSystem returns the value of the rpc.system / rpc.system.name attribute. The newer
// rpc.system.name is checked first since it is the current spec key.
func getRpcSystem(span ptrace.Span) (string, bool) {
	if rpcSystem, found := span.Attributes().Get("rpc.system.name"); found {
		return rpcSystem.Str(), true
	}
	if rpcSystem, found := span.Attributes().Get(string(semconv.RPCSystemKey)); found {
		return rpcSystem.Str(), true
	}
	return "", false
}

// getRpcMethod returns the bare gRPC method name from a span, handling both OTel semconv conventions:
//   - older split convention: rpc.method already carries just the bare method (e.g. "ListItems").
//   - newer fully-qualified convention: rpc.method is "Service/method" (e.g.
//     "acme.inventory.v1.InventoryService/ListItems"); we return the part after the "/".
//
// the split is applied only when both sides of "/" are non-empty, so sentinel values like
// "_OTHER" and degenerate forms like "/foo" / "foo/" are returned as-is.
func getRpcMethod(span ptrace.Span) (string, bool) {
	rpcMethod, found := span.Attributes().Get(string(semconv.RPCMethodKey))
	if !found {
		return "", false
	}
	value := rpcMethod.Str()
	if service, method, hasSlash := strings.Cut(value, "/"); hasSlash && service != "" && method != "" {
		return method, true
	}
	return value, true
}

// getRpcService returns the fully-qualified gRPC service name from a span, handling both OTel
// semconv conventions:
//   - older split convention: read rpc.service directly.
//   - newer fully-qualified convention: rpc.service is deprecated and the service is encoded as
//     the prefix of rpc.method ("Service/method"); when rpc.service is absent we extract it from
//     the part before the "/".
func getRpcService(span ptrace.Span) (string, bool) {
	if rpcService, found := span.Attributes().Get(string(semconv.RPCServiceKey)); found {
		return rpcService.Str(), true
	}
	rpcMethod, found := span.Attributes().Get(string(semconv.RPCMethodKey))
	if !found {
		return "", false
	}
	value := rpcMethod.Str()
	if service, method, hasSlash := strings.Cut(value, "/"); hasSlash && service != "" && method != "" {
		return service, true
	}
	return "", false
}
