package matchers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common/urltemplate"
)

func mustParsePathRulePrefix(t *testing.T, path string, prefix bool) urltemplate.PathRule {
	t.Helper()
	if path == "" {
		return urltemplate.PathRule{}
	}
	rule, err := urltemplate.ParseUserInputRuleString(path, prefix)
	require.NoError(t, err)
	return rule
}

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
		name      string
		spanRoute string
		ruleRoute string
		prefix    bool
		wantMatch bool
	}{
		// exact match
		{
			name:      "exact match",
			spanRoute: "/users/:id",
			ruleRoute: "/users/:id",
			wantMatch: true,
		},
		{
			name:      "exact no match",
			spanRoute: "/users/:id",
			ruleRoute: "/orders/:id",
			wantMatch: false,
		},
		{
			name:      "exact empty span route no match",
			spanRoute: "",
			ruleRoute: "/api",
			wantMatch: false,
		},
		{
			name:      "exact templated segment matches any",
			spanRoute: "/users/123",
			ruleRoute: "/users/{id}",
			wantMatch: true,
		},
		{
			name:      "exact wildcard segment matches any",
			spanRoute: "/users/john",
			ruleRoute: "/users/*",
			wantMatch: true,
		},
		{
			name:      "exact templated segment count mismatch",
			spanRoute: "/users/123/orders",
			ruleRoute: "/users/{id}",
			wantMatch: false,
		},
		// prefix match
		{
			name:      "prefix match",
			spanRoute: "/api/v1/users",
			ruleRoute: "/api",
			prefix:    true,
			wantMatch: true,
		},
		{
			name:      "prefix match same as route",
			spanRoute: "/api",
			ruleRoute: "/api",
			prefix:    true,
			wantMatch: true,
		},
		{
			name:      "prefix no match",
			spanRoute: "/v2/users",
			ruleRoute: "/api",
			prefix:    true,
			wantMatch: false,
		},
		{
			name:      "prefix no match empty span",
			spanRoute: "",
			ruleRoute: "/api",
			prefix:    true,
			wantMatch: false,
		},
		{
			name:      "prefix with templated segment",
			spanRoute: "/users/123/orders",
			ruleRoute: "/users/{id}",
			prefix:    true,
			wantMatch: true,
		},
		// unset -> match any
		{
			name:      "unset match any",
			spanRoute: "/anything",
			wantMatch: true,
		},
		{
			name:      "unset empty span",
			spanRoute: "",
			wantMatch: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareHttpRoute(tt.spanRoute, mustParsePathRulePrefix(t, tt.ruleRoute, tt.prefix))
			assert.Equal(t, tt.wantMatch, got)
		})
	}
}

func TestParseRoutePathSegments(t *testing.T) {
	t.Run("exact takes precedence over prefix", func(t *testing.T) {
		rule := parseRoutePathSegments("/api/v1", "/somethingelse")
		assert.False(t, rule.Prefix)
		assert.True(t, compareHttpRoute("/api/v1", rule))
		assert.False(t, compareHttpRoute("/somethingelse/x", rule))
	})
	t.Run("prefix used when exact empty", func(t *testing.T) {
		rule := parseRoutePathSegments("", "/api")
		assert.True(t, rule.Prefix)
		assert.True(t, compareHttpRoute("/api/v1", rule))
	})
	t.Run("empty when both unset", func(t *testing.T) {
		rule := parseRoutePathSegments("", "")
		assert.True(t, rule.Empty())
	})
}
