package urltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func riMustParse(t *testing.T, ruleString string) PathRule {
	t.Helper()
	rule, err := ParseUserInputRuleString(ruleString, false)
	require.NoError(t, err)
	return rule
}

func TestAppendRuleBySegmentCountAccumulatesInDeclarationOrder(t *testing.T) {
	index := map[int][]PathRule{}
	for _, ruleString := range []string{"/users/admin", "/users/{id}", "/health", "/a/b/c"} {
		AppendRuleBySegmentCount(index, riMustParse(t, ruleString))
	}

	require.Len(t, index, 3)

	require.Len(t, index[2], 2, "rules sharing a segment count must accumulate, not replace each other")
	assert.Equal(t, "users/admin", index[2][0].StaticPath)
	assert.Equal(t, "id", index[2][1].Segments[1].TemplateName)

	require.Len(t, index[1], 1)
	assert.Equal(t, "health", index[1][0].StaticPath)
	require.Len(t, index[3], 1)
	assert.Equal(t, "a/b/c", index[3][0].StaticPath)
}

func TestParseRulesBySegmentCountKeepsEveryRule(t *testing.T) {
	ruleStrings := []string{"/users/{id}", "users/*", "/health", "/a/b/c", "orders/{orderId}"}
	index := ParseRulesBySegmentCount(ruleStrings)

	total := 0
	for _, bucket := range index {
		total += len(bucket)
	}
	assert.Equal(t, len(ruleStrings), total)

	require.Len(t, index[1], 1)
	assert.Len(t, index[2], 3, "an optional leading slash must not change a rule's segment count")
	require.Len(t, index[3], 1)
	assert.Empty(t, index[4])
}

// The rule that wins decides the template the processor emits as http.route, so it is not enough
// to know that some rule matched.
func TestFindMatchingRuleReturnsTheFirstDeclaredMatch(t *testing.T) {
	t.Run("a templated rule declared first beats a later exact rule", func(t *testing.T) {
		index := ParseRulesBySegmentCount([]string{"/users/{id}", "/users/admin"})

		rule, found := FindMatchingRule([]string{"users", "admin"}, index)
		require.True(t, found)
		assert.Equal(t, "id", rule.Segments[1].TemplateName)
		assert.Empty(t, rule.Segments[1].StaticString)
	})

	t.Run("the same two rules swapped make the exact rule win", func(t *testing.T) {
		index := ParseRulesBySegmentCount([]string{"/users/admin", "/users/{id}"})

		rule, found := FindMatchingRule([]string{"users", "admin"}, index)
		require.True(t, found)
		assert.Equal(t, "admin", rule.Segments[1].StaticString)
		assert.Empty(t, rule.Segments[1].TemplateName)
	})

	t.Run("a wildcard declared first beats a later templated rule", func(t *testing.T) {
		index := ParseRulesBySegmentCount([]string{"/users/*", "/users/{id}"})

		rule, found := FindMatchingRule([]string{"users", "42"}, index)
		require.True(t, found)
		assert.True(t, rule.Segments[1].Wildcard)
		assert.Empty(t, rule.Segments[1].TemplateName)
	})
}

func TestFindMatchingRuleReportsNoMatch(t *testing.T) {
	index := ParseRulesBySegmentCount([]string{"/users/{id}", "/health"})

	tests := []struct {
		name     string
		segments []string
	}{
		{"no bucket for this segment count", []string{"users", "42", "orders"}},
		{"right bucket but the static segment differs", []string{"orders", "42"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, found := FindMatchingRule(tc.segments, index)
			assert.False(t, found)
			assert.True(t, rule.Empty(), "the zero rule must be returned when nothing matched")
		})
	}
}

// Rules come from user configuration, where the leading slash is optional; paths come from live
// traffic, where it is always present.
func TestPathMatchesAnyRuleIgnoresLeadingSlashes(t *testing.T) {
	tests := []struct {
		name string
		rule string
		path string
	}{
		{"slashed rule and slashed path", "/users/{id}", "/users/42"},
		{"slashed rule and bare path", "/users/{id}", "users/42"},
		{"bare rule and slashed path", "users/{id}", "/users/42"},
		{"bare rule and bare path", "users/{id}", "users/42"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			index := ParseRulesBySegmentCount([]string{tc.rule})
			assert.True(t, PathMatchesAnyRule(tc.path, index))
			assert.False(t, PathMatchesAnyRule("/orders/42", index))
			assert.False(t, PathMatchesAnyRule("orders/42", index))
		})
	}
}

// ParseUserInputRuleString never fails, so ParseRulesBySegmentCount's "invalid rules are skipped"
// branch is unreachable and a blank entry in the configured rule list becomes a real rule.
func TestParseRulesBySegmentCountTreatsABlankRuleAsTheRootPath(t *testing.T) {
	t.Run("a blank rule matches the root path only", func(t *testing.T) {
		index := ParseRulesBySegmentCount([]string{""})

		assert.True(t, PathMatchesAnyRule("/", index))
		assert.True(t, PathMatchesAnyRule("", index))
		assert.False(t, PathMatchesAnyRule("/health", index),
			"the index holds exact rules, so a blank rule is not a catch-all")
		assert.False(t, PathMatchesAnyRule("/users/42", index))
	})

	t.Run("a blank rule does not displace a real rule of the same length", func(t *testing.T) {
		index := ParseRulesBySegmentCount([]string{"", "/health"})

		assert.True(t, PathMatchesAnyRule("/", index))
		assert.True(t, PathMatchesAnyRule("/health", index))
		assert.False(t, PathMatchesAnyRule("/users/42", index))
	})
}

// Bucketing by rule segment count only works for exact rules: a prefix rule matches paths longer
// than itself, which are looked up in a bucket it was never added to.
func TestSegmentCountIndexIsExactMatchOnly(t *testing.T) {
	prefixRule, err := ParseUserInputRuleString("/api", true)
	require.NoError(t, err)
	require.True(t, prefixRule.IsPathMatching("/api/v1/users"))

	index := map[int][]PathRule{}
	AppendRuleBySegmentCount(index, prefixRule)

	assert.True(t, PathMatchesAnyRule("/api", index))
	assert.False(t, PathMatchesAnyRule("/api/v1/users", index),
		"an indexed rule is only consulted for paths with its own segment count")
}
