package urltemplate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseUserInputRuleString(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		prefix         bool
		wantSegments   []RulePathSegment
		wantAllStatic  bool
		wantStaticPath string
	}{
		{
			name:           "all static segments",
			input:          "api/v1/users",
			wantSegments:   []RulePathSegment{{StaticString: "api"}, {StaticString: "v1"}, {StaticString: "users"}},
			wantAllStatic:  true,
			wantStaticPath: "api/v1/users",
		},
		{
			name:           "leading slash is stripped from segments and from the static path",
			input:          "/api/v1/users",
			wantSegments:   []RulePathSegment{{StaticString: "api"}, {StaticString: "v1"}, {StaticString: "users"}},
			wantAllStatic:  true,
			wantStaticPath: "api/v1/users",
		},
		{
			name:          "named template segment",
			input:         "users/{userId}",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {TemplateName: "userId"}},
			wantAllStatic: false,
		},
		{
			name:          "empty template name defaults to id",
			input:         "users/{}",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {TemplateName: "id"}},
			wantAllStatic: false,
		},
		{
			name:          "template name is trimmed",
			input:         "users/{  userId  }",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {TemplateName: "userId"}},
			wantAllStatic: false,
		},
		{
			name:          "whitespace only template name defaults to id",
			input:         "users/{   }",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {TemplateName: "id"}},
			wantAllStatic: false,
		},
		{
			name:          "wildcard segment",
			input:         "users/*",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {Wildcard: true}},
			wantAllStatic: false,
		},
		{
			name:          "wildcard and template mixed with static",
			input:         "/api/{version}/*/items/{itemId}",
			wantSegments:  []RulePathSegment{{StaticString: "api"}, {TemplateName: "version"}, {Wildcard: true}, {StaticString: "items"}, {TemplateName: "itemId"}},
			wantAllStatic: false,
		},
		{
			name:          "wildcard only as a substring is a static segment",
			input:         "users/*s",
			wantSegments:  []RulePathSegment{{StaticString: "users"}, {StaticString: "*s"}},
			wantAllStatic: true,
			// a rule with a literal "*s" segment is static, so the fast string path is used
			wantStaticPath: "users/*s",
		},
		{
			name:           "unbalanced curly braces are static segments",
			input:          "users/{id",
			wantSegments:   []RulePathSegment{{StaticString: "users"}, {StaticString: "{id"}},
			wantAllStatic:  true,
			wantStaticPath: "users/{id",
		},
		{
			name:           "trailing slash produces a trailing empty static segment",
			input:          "api/",
			wantSegments:   []RulePathSegment{{StaticString: "api"}, {StaticString: ""}},
			wantAllStatic:  true,
			wantStaticPath: "api/",
		},
		{
			name:           "empty rule is a single empty static segment",
			input:          "",
			wantSegments:   []RulePathSegment{{StaticString: ""}},
			wantAllStatic:  true,
			wantStaticPath: "",
		},
		{
			name:           "root rule is a single empty static segment",
			input:          "/",
			wantSegments:   []RulePathSegment{{StaticString: ""}},
			wantAllStatic:  true,
			wantStaticPath: "",
		},
		{
			name:           "prefix flag is recorded on the rule",
			input:          "/api",
			prefix:         true,
			wantSegments:   []RulePathSegment{{StaticString: "api"}},
			wantAllStatic:  true,
			wantStaticPath: "api",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rule, err := ParseUserInputRuleString(tt.input, tt.prefix)
			require.NoError(t, err)
			assert.Equal(t, tt.wantSegments, rule.Segments)
			assert.Equal(t, tt.prefix, rule.Prefix)
			assert.Equal(t, tt.wantAllStatic, rule.AllStatic)
			assert.Equal(t, tt.wantStaticPath, rule.StaticPath)
		})
	}
}

func TestPathRuleEmpty(t *testing.T) {
	t.Run("zero value is empty", func(t *testing.T) {
		assert.True(t, PathRule{}.Empty())
	})

	// callers use Empty() to mean "no rule configured, match everything". Parsing an empty
	// string does not produce that: it produces a rule with one empty static segment, which
	// only matches the root path. Callers must not parse an unset user input.
	t.Run("parsed empty string is not an empty rule", func(t *testing.T) {
		rule, err := ParseUserInputRuleString("", false)
		require.NoError(t, err)
		assert.False(t, rule.Empty())
		assert.False(t, rule.IsPathMatching("/users"))
	})

	t.Run("parsed rule is not empty", func(t *testing.T) {
		rule, err := ParseUserInputRuleString("/users/{id}", false)
		require.NoError(t, err)
		assert.False(t, rule.Empty())
	})
}
