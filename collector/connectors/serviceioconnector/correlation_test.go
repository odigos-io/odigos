package serviceioconnector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
)

func TestExtractSpanAttributes(t *testing.T) {
	span := ptrace.NewSpan()
	span.SetKind(ptrace.SpanKindServer)
	span.Attributes().PutStr("http.route", "/users/{id}")
	span.Attributes().PutInt("http.status_code", 200)
	span.Attributes().PutStr("rpc.service", "UserService")

	scope := pcommon.NewInstrumentationScope()
	scope.SetName("io.opentelemetry.http")
	scope.SetVersion("1.2.3")

	node := &TraceTreeNode{
		Span:  span,
		Scope: scope,
	}

	values := ExtractSpanAttributes(node, inputAttributePrefix, []string{
		"http.route",
		"http.status_code",
		"rpc.service",
		"missing.key",
	})
	require.Equal(t, 7, values.Len())

	spanName, ok := values.Get(inputAttributePrefix + spanNameAttribute)
	require.True(t, ok)
	require.Equal(t, "", spanName.Str())

	kind, ok := values.Get(inputAttributePrefix + spanKindAttribute)
	require.True(t, ok)
	require.Equal(t, "Server", kind.Str())

	scopeName, ok := values.Get(inputAttributePrefix + string(semconv.OTelScopeNameKey))
	require.True(t, ok)
	require.Equal(t, "io.opentelemetry.http", scopeName.Str())

	scopeVersion, ok := values.Get(inputAttributePrefix + string(semconv.OTelScopeVersionKey))
	require.True(t, ok)
	require.Equal(t, "1.2.3", scopeVersion.Str())

	route, ok := values.Get(inputAttributePrefix + "http.route")
	require.True(t, ok)
	require.Equal(t, "/users/{id}", route.Str())

	statusCode, ok := values.Get(inputAttributePrefix + "http.status_code")
	require.True(t, ok)
	require.Equal(t, pcommon.ValueTypeInt, statusCode.Type())
	require.EqualValues(t, 200, statusCode.Int())

	rpcService, ok := values.Get(inputAttributePrefix + "rpc.service")
	require.True(t, ok)
	require.Equal(t, "UserService", rpcService.Str())
}

func TestExtractSpanAttributes_EmptyConfig(t *testing.T) {
	span := ptrace.NewSpan()
	span.SetKind(ptrace.SpanKindClient)
	span.Attributes().PutStr("http.route", "/health")

	node := &TraceTreeNode{
		Span:  span,
		Scope: pcommon.NewInstrumentationScope(),
	}

	values := ExtractSpanAttributes(node, inputAttributePrefix, nil)
	require.Equal(t, 2, values.Len())

	spanName, ok := values.Get(inputAttributePrefix + spanNameAttribute)
	require.True(t, ok)
	require.Equal(t, "", spanName.Str())

	kind, ok := values.Get(inputAttributePrefix + spanKindAttribute)
	require.True(t, ok)
	require.Equal(t, "Client", kind.Str())
}
