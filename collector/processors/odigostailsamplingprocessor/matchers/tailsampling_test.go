package matchers

import (
	"testing"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func TestOperationHttpServerMatcher(t *testing.T) {
	tests := []struct {
		name      string
		operation *commonapi.HttpServerOperationMatcher
		spanKind  ptrace.SpanKind
		attrs     map[string]string
		want      bool
	}{
		{
			name:      "non-server span returns false",
			operation: &commonapi.HttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name:      "server span without http method returns false",
			operation: &commonapi.HttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs:     map[string]string{"other.attr": "value"},
			want:      false,
		},
		{
			name:      "server span with method and empty operation matches",
			operation: &commonapi.HttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "server span method exact match",
			operation: &commonapi.HttpServerOperationMatcher{
				Method: "GET",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "server span method mismatch",
			operation: &commonapi.HttpServerOperationMatcher{
				Method: "POST",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name: "server span route exact match",
			operation: &commonapi.HttpServerOperationMatcher{
				Route: "/users/:id",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/users/:id",
			},
			want: true,
		},
		{
			name: "server span route no match",
			operation: &commonapi.HttpServerOperationMatcher{
				Route: "/orders",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/users/:id",
			},
			want: false,
		},
		{
			name: "server span route prefix match",
			operation: &commonapi.HttpServerOperationMatcher{
				RoutePrefix: "/api",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.URLPathKey):           "/api/v1/health",
			},
			want: true,
		},
		{
			name: "server span route prefix no match",
			operation: &commonapi.HttpServerOperationMatcher{
				RoutePrefix: "/api",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/web/login",
			},
			want: false,
		},
		{
			name: "server span method and route both match",
			operation: &commonapi.HttpServerOperationMatcher{
				Method: "POST",
				Route:  "/users",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "POST",
				string(semconv.HTTPRouteKey):         "/users",
			},
			want: true,
		},
		{
			name: "server span method match but route does not",
			operation: &commonapi.HttpServerOperationMatcher{
				Method: "GET",
				Route:  "/users",
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/orders",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrsAndKind(t, tt.spanKind, tt.attrs)
			got := operationHttpServerMatcher(tt.operation, span)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTailSamplingOperationMatcher(t *testing.T) {
	tests := []struct {
		name      string
		operation *commonapi.TailSamplingOperationMatcher
		spanKind  ptrace.SpanKind
		attrs     map[string]string
		want      bool
	}{
		{
			name:      "nil operation matches any span",
			operation: nil,
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name:      "empty operation matches any span",
			operation: &commonapi.TailSamplingOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrsAndKind(t, tt.spanKind, tt.attrs)
			got := TailSamplingOperationMatcher(tt.operation, span)
			assert.Equal(t, tt.want, got)
		})
	}
}
