package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	semconv_1_4_0 "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// spanWithAttrs creates a span with the given attributes (key -> string value).
func spanWithAttrs(t *testing.T, attrs map[string]string) ptrace.Span {
	t.Helper()
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	for k, v := range attrs {
		span.Attributes().PutStr(k, v)
	}
	return span
}

func TestGetHttpMethod(t *testing.T) {
	tests := []struct {
		name       string
		attrs      map[string]string
		wantMethod string
		wantFound  bool
	}{
		{
			name: "new semconv http.request.method",
			attrs: map[string]string{
				string(semconv.HTTPRequestMethodKey): "POST",
			},
			wantMethod: "POST",
			wantFound:  true,
		},
		{
			name: "old semconv http.method only",
			attrs: map[string]string{
				string(semconv_1_4_0.HTTPMethodKey): "GET",
			},
			wantMethod: "GET",
			wantFound:  true,
		},
		{
			name:       "no http method attribute",
			attrs:      map[string]string{"other.attr": "value"},
			wantMethod: "",
			wantFound:  false,
		},
		{
			name:       "empty span attributes",
			attrs:      map[string]string{},
			wantMethod: "",
			wantFound:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			gotMethod, gotFound := getHttpMethod(span)
			require.Equal(t, tt.wantFound, gotFound, "found")
			assert.Equal(t, tt.wantMethod, gotMethod, "method")
		})
	}
}

func TestGetHttpRoute(t *testing.T) {
	tests := []struct {
		name      string
		attrs     map[string]string
		wantRoute string
		wantFound bool
	}{
		{
			name: "http.route present",
			attrs: map[string]string{
				string(semconv.HTTPRouteKey): "/users/:id",
			},
			wantRoute: "/users/:id",
			wantFound: true,
		},
		{
			name: "http.route with root path",
			attrs: map[string]string{
				string(semconv.HTTPRouteKey): "/",
			},
			wantRoute: "/",
			wantFound: true,
		},
		{
			name:      "no http.route attribute",
			attrs:     map[string]string{"other.attr": "value"},
			wantRoute: "",
			wantFound: false,
		},
		{
			name:      "empty span attributes",
			attrs:     map[string]string{},
			wantRoute: "",
			wantFound: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			gotRoute, gotFound := getHttpRoute(span)
			require.Equal(t, tt.wantFound, gotFound, "found")
			assert.Equal(t, tt.wantRoute, gotRoute, "route")
		})
	}
}

func TestGetHttpServerPath(t *testing.T) {
	tests := []struct {
		name      string
		attrs     map[string]string
		wantPath  string
		wantFound bool
	}{
		{
			name: "url.path present",
			attrs: map[string]string{
				string(semconv.URLPathKey): "/api/v1/users",
			},
			wantPath:  "/api/v1/users",
			wantFound: true,
		},
		{
			name: "http.target when url.path absent",
			attrs: map[string]string{
				string(semconv_1_4_0.HTTPTargetKey): "/legacy?foo=1",
			},
			wantPath:  "/legacy?foo=1",
			wantFound: true,
		},
		{
			name: "url.path takes precedence over http.target",
			attrs: map[string]string{
				string(semconv.URLPathKey):          "/path",
				string(semconv_1_4_0.HTTPTargetKey): "/target",
			},
			wantPath:  "/path",
			wantFound: true,
		},
		{
			name:      "no path attribute",
			attrs:     map[string]string{"other.attr": "value"},
			wantPath:  "",
			wantFound: false,
		},
		{
			name:      "empty span attributes",
			attrs:     map[string]string{},
			wantPath:  "",
			wantFound: false,
		},
		{
			name: "url.path empty string still found",
			attrs: map[string]string{
				string(semconv.URLPathKey): "",
			},
			wantPath:  "",
			wantFound: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			gotPath, gotFound := getHttpServerPath(span)
			require.Equal(t, tt.wantFound, gotFound, "found")
			assert.Equal(t, tt.wantPath, gotPath, "path")
		})
	}
}

func TestGetServerAddress(t *testing.T) {
	tests := []struct {
		name        string
		attrs       map[string]string
		wantAddress string
		wantFound   bool
	}{
		{
			name: "server.address present",
			attrs: map[string]string{
				string(semconv.ServerAddressKey): "api.example.com",
			},
			wantAddress: "api.example.com",
			wantFound:   true,
		},
		{
			name:        "no server.address attribute",
			attrs:       map[string]string{"other.attr": "value"},
			wantAddress: "",
			wantFound:   false,
		},
		{
			name:        "empty span attributes",
			attrs:       map[string]string{},
			wantAddress: "",
			wantFound:   false,
		},
		{
			name: "server.address empty string still found",
			attrs: map[string]string{
				string(semconv.ServerAddressKey): "",
			},
			wantAddress: "",
			wantFound:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := spanWithAttrs(t, tt.attrs)
			gotAddress, gotFound := getServerAddress(span)
			require.Equal(t, tt.wantFound, gotFound, "found")
			assert.Equal(t, tt.wantAddress, gotAddress, "address")
		})
	}
}
