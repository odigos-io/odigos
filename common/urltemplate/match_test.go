package urltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseRule(t *testing.T, ruleString string, prefix bool) PathRule {
	t.Helper()
	rule, err := ParseUserInputRuleString(ruleString, prefix)
	require.NoError(t, err)
	return rule
}

type pathMatchCase struct {
	name      string
	rule      string
	prefix    bool
	path      string
	wantMatch bool
}

// cases shared by IsPathMatching and IsPathSegmentsMatching, to pin that the two entry
// points (sampling matches a raw path, templatization matches pre-split segments) agree.
var pathMatchCases = []pathMatchCase{
	// exact, all static (the StaticPath fast path)
	{name: "static exact match", rule: "/api/v1/users", path: "/api/v1/users", wantMatch: true},
	{name: "static exact match when rule has no leading slash", rule: "api/v1/users", path: "/api/v1/users", wantMatch: true},
	{name: "static exact match when path has no leading slash", rule: "/api/v1/users", path: "api/v1/users", wantMatch: true},
	{name: "static exact different segment", rule: "/api/v1/users", path: "/api/v2/users", wantMatch: false},
	{name: "static exact longer path", rule: "/api/v1", path: "/api/v1/users", wantMatch: false},
	{name: "static exact shorter path", rule: "/api/v1/users", path: "/api/v1", wantMatch: false},
	{name: "static exact trailing slash on path", rule: "/api/v1", path: "/api/v1/", wantMatch: false},
	{name: "static exact is not a substring match", rule: "/api", path: "/apiv2", wantMatch: false},

	// prefix, all static
	{name: "static prefix same path", rule: "/api", prefix: true, path: "/api", wantMatch: true},
	{name: "static prefix longer path", rule: "/api", prefix: true, path: "/api/v1/users", wantMatch: true},
	{name: "static prefix multi segment", rule: "/api/v1", prefix: true, path: "/api/v1/users", wantMatch: true},
	{name: "static prefix stops at a segment boundary", rule: "/api", prefix: true, path: "/apiv2/users", wantMatch: false},
	{name: "static prefix partial last segment", rule: "/api/v1", prefix: true, path: "/api/v12", wantMatch: false},
	{name: "static prefix shorter path", rule: "/api/v1", prefix: true, path: "/api", wantMatch: false},
	{name: "static prefix different path", rule: "/api", prefix: true, path: "/v2/users", wantMatch: false},
	{name: "static prefix trailing slash on path", rule: "/api", prefix: true, path: "/api/", wantMatch: true},
	// a root prefix rule has an empty static path, which matches anything
	{name: "root prefix rule matches any path", rule: "/", prefix: true, path: "/api/v1/users", wantMatch: true},
	{name: "root prefix rule matches the root path", rule: "/", prefix: true, path: "/", wantMatch: true},
	{name: "root exact rule matches only the root path", rule: "/", path: "/", wantMatch: true},
	{name: "root exact rule does not match a longer path", rule: "/", path: "/api", wantMatch: false},

	// exact, templated / wildcard segments
	{name: "template segment matches any value", rule: "/users/{id}", path: "/users/123", wantMatch: true},
	{name: "template segment matches a non numeric value", rule: "/users/{id}", path: "/users/john-doe", wantMatch: true},
	{name: "template segment does not span segments", rule: "/users/{id}", path: "/users/123/orders", wantMatch: false},
	{name: "template segment requires a segment to be present", rule: "/users/{id}", path: "/users", wantMatch: false},
	{name: "template segment matches an empty segment", rule: "/users/{id}", path: "/users/", wantMatch: true},
	{name: "wildcard segment matches any value", rule: "/users/*", path: "/users/john", wantMatch: true},
	{name: "wildcard segment does not span segments", rule: "/users/*", path: "/users/john/orders", wantMatch: false},
	{name: "static segment around a template must still match", rule: "/users/{id}/orders", path: "/accounts/123/orders", wantMatch: false},
	{name: "trailing static segment after a template must match", rule: "/users/{id}/orders", path: "/users/123/invoices", wantMatch: false},
	{name: "several templates and wildcards", rule: "/api/{version}/*/items/{itemId}", path: "/api/v1/anything/items/42", wantMatch: true},
	{name: "several templates and wildcards with a wrong static segment", rule: "/api/{version}/*/items/{itemId}", path: "/api/v1/anything/things/42", wantMatch: false},
	{name: "templated rule does not match an empty path", rule: "/users/{id}", path: "", wantMatch: false},

	// prefix, templated / wildcard segments
	{name: "templated prefix matches a longer path", rule: "/users/{id}", prefix: true, path: "/users/123/orders", wantMatch: true},
	{name: "templated prefix matches the exact length path", rule: "/users/{id}", prefix: true, path: "/users/123", wantMatch: true},
	{name: "templated prefix requires all rule segments", rule: "/users/{id}", prefix: true, path: "/users", wantMatch: false},
	{name: "templated prefix static segment must match", rule: "/users/{id}", prefix: true, path: "/accounts/123/orders", wantMatch: false},
	{name: "wildcard prefix matches a longer path", rule: "/api/*", prefix: true, path: "/api/v1/users/123", wantMatch: true},
}

func TestPathRuleIsPathMatching(t *testing.T) {
	for _, tt := range pathMatchCases {
		t.Run(tt.name, func(t *testing.T) {
			rule := mustParseRule(t, tt.rule, tt.prefix)
			assert.Equal(t, tt.wantMatch, rule.IsPathMatching(tt.path))
		})
	}
}

func TestPathRuleIsPathSegmentsMatching(t *testing.T) {
	for _, tt := range pathMatchCases {
		t.Run(tt.name, func(t *testing.T) {
			rule := mustParseRule(t, tt.rule, tt.prefix)
			segments, _ := SplitPath(tt.path)
			assert.Equal(t, tt.wantMatch, rule.IsPathSegmentsMatching(segments))
		})
	}
}
