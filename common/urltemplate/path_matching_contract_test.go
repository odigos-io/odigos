package urltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitPathReportsSegmentsAndLeadingSlash(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantSegments     []string
		wantLeadingSlash bool
	}{
		{"slashed path", "/users/123", []string{"users", "123"}, true},
		{"bare path", "users/123", []string{"users", "123"}, false},
		{"single segment", "/health", []string{"health"}, true},
		{"root path", "/", []string{""}, true},
		{"empty path", "", []string{""}, false},
		{"trailing slash keeps an empty last segment", "/a/b/", []string{"a", "b", ""}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			segments, hadLeadingSlash := SplitPath(tc.path)
			assert.Equal(t, tc.wantSegments, segments)
			assert.Equal(t, tc.wantLeadingSlash, hadLeadingSlash)
		})
	}
}

func TestIsPathMatchingExactRule(t *testing.T) {
	tests := []struct {
		name string
		rule string
		path string
		want bool
	}{
		{"static rule and path agree", "/api/v1/users", "/api/v1/users", true},
		{"static rule tolerates a bare path", "/api/v1/users", "api/v1/users", true},
		{"bare static rule tolerates a slashed path", "api/v1/users", "/api/v1/users", true},
		{"static rule rejects a longer path", "/api/v1", "/api/v1/users", false},
		{"static rule rejects a shorter path", "/api/v1/users", "/api/v1", false},
		{"static rule rejects a differing segment", "/api/v1/users", "/api/v2/users", false},
		{"templated segment matches any value", "/users/{id}", "/users/42", true},
		{"unnamed templated segment matches any value", "/users/{}", "/users/42", true},
		{"templated rule still requires the same length", "/users/{id}", "/users/42/orders", false},
		{"templated rule requires its static segment to agree", "/users/{id}", "/orders/42", false},
		{"wildcard segment matches any value", "/v1/*", "/v1/users", true},
		{"wildcard segment matches one segment only", "/v1/*", "/v1/a/b", false},
		{"an unterminated brace is a static segment", "/users/{id", "/users/{id", true},
		{"an unterminated brace does not templatize", "/users/{id", "/users/42", false},
		{"root rule matches the root path", "/", "/", true},
		{"root rule matches the empty path", "/", "", true},
		{"root rule rejects a named path", "/", "/health", false},
		{"named rule rejects the empty path", "/health", "", false},
		{"single templated segment matches the empty path", "/{tenant}", "", true},
		{"single wildcard segment matches the root path", "/*", "/", true},
		{"two templated segments reject the empty path", "/{a}/{b}", "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := ParseUserInputRuleString(tc.rule, false)
			require.NoError(t, err)
			assert.Equal(t, tc.want, rule.IsPathMatching(tc.path))
		})
	}
}

func TestIsPathMatchingPrefixRule(t *testing.T) {
	tests := []struct {
		name string
		rule string
		path string
		want bool
	}{
		{"prefix matches itself", "/api", "/api", true},
		{"prefix matches a longer path", "/api", "/api/v1/users", true},
		{"prefix matches a trailing slash", "/api", "/api/", true},
		{"prefix stops at a segment boundary", "/api", "/apixyz", false},
		{"prefix is not a substring match", "/api", "/apis/v1", false},
		{"prefix rejects a shorter path", "/api", "/ap", false},
		{"multi segment prefix matches a longer path", "/api/v1", "/api/v1/users", true},
		{"multi segment prefix stops at a segment boundary", "/api/v1", "/api/v1x", false},
		{"multi segment prefix rejects a differing segment", "/api/v1", "/api/v2/users", false},
		{"templated prefix matches a longer path", "/users/{id}", "/users/42/orders", true},
		{"templated prefix matches its own length", "/users/{id}", "/users/42", true},
		{"templated prefix rejects a shorter path", "/users/{id}", "/users", false},
		{"templated prefix requires its static segment to agree", "/users/{id}", "/orders/42/items", false},
		{"empty prefix matches every path", "", "/anything/at/all", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := ParseUserInputRuleString(tc.rule, true)
			require.NoError(t, err)
			assert.Equal(t, tc.want, rule.IsPathMatching(tc.path))
		})
	}
}

// The tail sampling processor matches routes through IsPathMatching while the URL templatization
// processor matches them through IsPathSegmentsMatching. A rule must mean the same thing in both.
func TestBothMatchEntryPointsAgree(t *testing.T) {
	ruleStrings := []string{"", "/", "/api", "/api/v1", "/users/{id}", "/v1/*", "/a/b/c"}
	paths := []string{
		"", "/", "/api", "api", "/api/", "/apixyz", "/apis/v1",
		"/api/v1", "/api/v1/users", "/users/42", "/v1/users", "/v1/a/b", "/a/b/c",
	}

	for _, prefix := range []bool{false, true} {
		for _, ruleString := range ruleStrings {
			rule, err := ParseUserInputRuleString(ruleString, prefix)
			require.NoError(t, err)
			for _, path := range paths {
				segments, _ := SplitPath(path)
				assert.Equal(t, rule.IsPathMatching(path), rule.IsPathSegmentsMatching(segments),
					"rule %q (prefix=%v) disagrees about path %q", ruleString, prefix, path)
			}
		}
	}
}

// The template name is what the URL templatization processor emits in place of the matched
// segment, so it ends up verbatim in http.route.
func TestParseUserInputRuleStringTemplateNames(t *testing.T) {
	tests := []struct {
		name     string
		rule     string
		wantName string
	}{
		{"named template", "/users/{userId}", "userId"},
		{"unnamed template defaults to id", "/users/{}", "id"},
		{"surrounding whitespace is trimmed", "/users/{ userId }", "userId"},
		{"a blank name defaults to id", "/users/{  }", "id"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := ParseUserInputRuleString(tc.rule, false)
			require.NoError(t, err)
			require.Len(t, rule.Segments, 2)
			assert.Equal(t, tc.wantName, rule.Segments[1].TemplateName)
			assert.False(t, rule.AllStatic)
		})
	}
}

// Sampling reads Empty() as "no route filter is configured, match everything".
func TestPathRuleEmptyDistinguishesTheZeroRule(t *testing.T) {
	assert.True(t, PathRule{}.Empty())

	blank, err := ParseUserInputRuleString("", false)
	require.NoError(t, err)
	assert.False(t, blank.Empty(), "a parsed blank rule has one empty segment and is not the zero rule")
}
