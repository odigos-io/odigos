package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompareHttpMethod(t *testing.T) {
	tests := []struct {
		spanMethod string
		ruleMethod string
		wantMatch  bool
	}{
		{"GET", "GET", true},
		{"GET", "get", true},
		{"get", "GET", true},
		// not equal
		{"GET", "POST", false},
	}
	for _, tt := range tests {
		t.Run(tt.spanMethod+"_vs_"+tt.ruleMethod, func(t *testing.T) {
			got := compareHttpMethod(tt.spanMethod, tt.ruleMethod)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func TestCompareHttpRoute(t *testing.T) {
	tests := []struct {
		name            string
		spanRoute       string
		ruleRouteExact  string
		ruleRoutePrefix string
		wantMatch       bool
	}{
		// exact match
		{
			name:            "exact match",
			spanRoute:       "/users/:id",
			ruleRouteExact:  "/users/:id",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
		{
			name:            "exact no match",
			spanRoute:       "/users/:id",
			ruleRouteExact:  "/orders/:id",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
		{
			name:            "exact empty span route no match",
			spanRoute:       "",
			ruleRouteExact:  "/api",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
		// prefix match
		{
			name:            "prefix match",
			spanRoute:       "/api/v1/users",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       true,
		},
		{
			name:            "prefix match same as route",
			spanRoute:       "/api",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       true,
		},
		{
			name:            "prefix no match",
			spanRoute:       "/v2/users",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       false,
		},
		{
			name:            "prefix no match empty span",
			spanRoute:       "",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       false,
		},
		// exact takes precedence when both set
		{
			name:            "exact takes precedence over prefix when both set",
			spanRoute:       "/api/v1",
			ruleRouteExact:  "/api/v1",
			ruleRoutePrefix: "/somethingelse",
			wantMatch:       true,
		},
		{
			name:            "exact no match but prefix would match",
			spanRoute:       "/api/v1",
			ruleRouteExact:  "/other",
			ruleRoutePrefix: "/api",
			wantMatch:       false,
		},
		// both unset -> match any (compareHttpRoute)
		{
			name:            "both unset match any",
			spanRoute:       "/anything",
			ruleRouteExact:  "",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
		{
			name:            "both unset empty span",
			spanRoute:       "",
			ruleRouteExact:  "",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareHttpRoute(tt.spanRoute, tt.ruleRouteExact, tt.ruleRoutePrefix)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func TestComparePathToHttpRoute(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		ruleRouteExact  string
		ruleRoutePrefix string
		wantMatch       bool
	}{
		{
			name:            "exact match",
			path:            "/api/v1/users",
			ruleRouteExact:  "/api/v1/users",
			ruleRoutePrefix: "",
			wantMatch:       true,
		},
		{
			name:            "exact no match",
			path:            "/api/v1/users",
			ruleRouteExact:  "/api/v1/orders",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
		{
			name:            "prefix match",
			path:            "/api/v1/health",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       true,
		},
		{
			name:            "prefix no match",
			path:            "/v2/users",
			ruleRouteExact:  "",
			ruleRoutePrefix: "/api",
			wantMatch:       false,
		},
		{
			name:            "both unset returns false",
			path:            "/anything",
			ruleRouteExact:  "",
			ruleRoutePrefix: "",
			wantMatch:       false,
		},
		{
			name:            "exact takes precedence over prefix",
			path:            "/api/v1",
			ruleRouteExact:  "/api/v1",
			ruleRoutePrefix: "/api",
			wantMatch:       true,
		},
		{
			name:            "exact no match but prefix would match",
			path:            "/api/v1",
			ruleRouteExact:  "/other",
			ruleRoutePrefix: "/api",
			wantMatch:       false,
		},
		// Note: http.target with query string, and templatized path not supported yet.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := comparePathToHttpRoute(tt.path, tt.ruleRouteExact, tt.ruleRoutePrefix)
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}
