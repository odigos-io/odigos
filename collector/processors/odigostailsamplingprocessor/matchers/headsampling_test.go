package matchers

import (
	"testing"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// spanWithAttrsAndKind creates a span with the given kind and attributes (reuses spanWithAttrs from attrgetter_test.go).
func spanWithAttrsAndKind(t *testing.T, kind ptrace.SpanKind, attrs map[string]string) ptrace.Span {
	t.Helper()
	span := spanWithAttrs(t, attrs)
	span.SetKind(kind)
	return span
}

func TestHeadSamplingOperationHttpServerMatcher(t *testing.T) {
	tests := []struct {
		name      string
		operation *commonapisampling.HeadSamplingHttpServerOperationMatcher
		spanKind  ptrace.SpanKind
		attrs     map[string]string
		want      bool
	}{
		{
			name:      "non-server span returns false",
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name:      "server span without http method returns false",
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs:     map[string]string{"other.attr": "value"},
			want:      false,
		},
		{
			name:      "server span with method and empty operation matches",
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "server span method exact match",
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			operation: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
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
			got := headSamplingOperationHttpServerMatcher(tt.operation, span)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeadSamplingOperationHttpClientMatcher(t *testing.T) {
	tests := []struct {
		name      string
		operation *commonapisampling.HeadSamplingHttpClientOperationMatcher
		spanKind  ptrace.SpanKind
		attrs     map[string]string
		want      bool
	}{
		{
			name:      "non-client span returns false",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name:      "client span without http method returns false",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{},
			spanKind:  ptrace.SpanKindClient,
			attrs:     map[string]string{"other.attr": "value"},
			want:      false,
		},
		{
			name:      "client span with method and empty operation matches",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{},
			spanKind:  ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "client span method exact match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Method: "GET",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "client span method mismatch",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Method: "POST",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name: "client span server address match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				ServerAddress: "api.example.com",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.ServerAddressKey):     "api.example.com",
			},
			want: true,
		},
		{
			name: "client span server address no match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				ServerAddress: "other.example.com",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.ServerAddressKey):     "api.example.com",
			},
			want: false,
		},
		{
			name: "client span server address missing when required",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				ServerAddress: "api.example.com",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name: "client span route exact match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Route: "/users/:id",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/users/:id",
			},
			want: true,
		},
		{
			name: "client span route no match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Route: "/orders",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.HTTPRouteKey):         "/users/:id",
			},
			want: false,
		},
		{
			name: "client span route prefix match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				RoutePrefix: "/api",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.URLPathKey):           "/api/v1/health",
			},
			want: true,
		},
		{
			name: "client span method and server address and route all match",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Method:        "POST",
				ServerAddress: "collector.example.com",
				Route:         "/v1/export",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "POST",
				string(semconv.ServerAddressKey):     "collector.example.com",
				string(semconv.HTTPRouteKey):         "/v1/export",
			},
			want: true,
		},
		{
			name: "client span method match but server address does not",
			operation: &commonapisampling.HeadSamplingHttpClientOperationMatcher{
				Method:        "GET",
				ServerAddress: "collector.example.com",
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
				string(semconv.ServerAddressKey):     "other.example.com",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrsAndKind(t, tt.spanKind, tt.attrs)
			got := headSamplingOperationHttpClientMatcher(tt.operation, span)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeadSamplingOperationMatcher(t *testing.T) {
	// Tests only the dispatch and empty-operation behavior; full matcher logic is covered by
	// TestHeadSamplingOperationHttpServerMatcher and TestHeadSamplingOperationHttpClientMatcher.
	tests := []struct {
		name      string
		operation *commonapisampling.HeadSamplingOperationMatcher
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
			operation: &commonapisampling.HeadSamplingOperationMatcher{},
			spanKind:  ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "HttpServer set delegates to server matcher - match",
			operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Method: "GET"},
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "HttpServer set delegates to server matcher - client span no match",
			operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{},
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
		{
			name: "HttpClient set delegates to client matcher - match",
			operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpClient: &commonapisampling.HeadSamplingHttpClientOperationMatcher{Method: "GET"},
			},
			spanKind: ptrace.SpanKindClient,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: true,
		},
		{
			name: "HttpClient set delegates to client matcher - server span no match",
			operation: &commonapisampling.HeadSamplingOperationMatcher{
				HttpClient: &commonapisampling.HeadSamplingHttpClientOperationMatcher{},
			},
			spanKind: ptrace.SpanKindServer,
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "GET",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrsAndKind(t, tt.spanKind, tt.attrs)
			got := HeadSamplingOperationMatcher(tt.operation, span)
			assert.Equal(t, tt.want, got)
		})
	}
}
