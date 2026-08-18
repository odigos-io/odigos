package traces

import (
	"sort"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	renameractions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func spanRenamerAction(language common.ProgrammingLanguage, scopeName string, replacements ...actions.SpanRenamerRegexReplacement) odigosv1.Action {
	return odigosv1.Action{
		Spec: odigosv1.ActionSpec{
			SpanRenamer: &renameractions.SpanRenamerConfig{
				ProgrammingLanguage: language,
				ScopeName:           scopeName,
				RegexReplacements:   replacements,
			},
		},
	}
}

func spanRenamerReplacement(pattern, template string) actions.SpanRenamerRegexReplacement {
	return actions.SpanRenamerRegexReplacement{RegexPattern: pattern, TemplateText: template}
}

var spanRenamerSupportingDistro = distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
	SpanRenamer: &distro.SpanRenamer{Supported: true},
})

// scopeRuleByName looks the scope up by name because the production code builds the slice by
// iterating a map, so its order is not stable.
func scopeRuleByName(t *testing.T, cfg *actions.SpanRenamerConfig, scopeName string) actions.SpanRenamerScopeRules {
	t.Helper()
	for _, scopeRule := range cfg.ScopeRules {
		if scopeRule.ScopeName == scopeName {
			return scopeRule
		}
	}
	t.Fatalf("no scope rules for scope %q, got %v", scopeName, cfg.ScopeRules)
	return actions.SpanRenamerScopeRules{}
}

func scopeNames(cfg *actions.SpanRenamerConfig) []string {
	names := make([]string, 0, len(cfg.ScopeRules))
	for _, scopeRule := range cfg.ScopeRules {
		names = append(names, scopeRule.ScopeName)
	}
	sort.Strings(names)
	return names
}

func TestDistroSupportsTracesSpanRenamer(t *testing.T) {
	require.False(t, DistroSupportsTracesSpanRenamer(distroWithTraces(common.JavaProgrammingLanguage, nil)))
	require.False(t, DistroSupportsTracesSpanRenamer(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{})))
	require.False(t, DistroSupportsTracesSpanRenamer(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		SpanRenamer: &distro.SpanRenamer{Supported: false},
	})))
	require.True(t, DistroSupportsTracesSpanRenamer(spanRenamerSupportingDistro))
}

func TestCalculateSpanRenamerConfig_unsupportedDistro(t *testing.T) {
	actionsList := []odigosv1.Action{
		spanRenamerAction(common.JavaProgrammingLanguage, "jdbc", spanRenamerReplacement("^SELECT", "select")),
	}

	got := CalculateSpanRenamerConfig(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		SpanRenamer: &distro.SpanRenamer{Supported: false},
	}), &actionsList, common.JavaProgrammingLanguage)

	require.Nil(t, got)
}

func TestCalculateSpanRenamerConfig_noActions(t *testing.T) {
	actionsList := []odigosv1.Action{}
	require.Nil(t, CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage))
}

func TestCalculateSpanRenamerConfig_actionWithoutSpanRenamer(t *testing.T) {
	actionsList := []odigosv1.Action{{Spec: odigosv1.ActionSpec{}}}
	require.Nil(t, CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage))
}

// The agent-level action list is not language scoped, so an action written for another language
// must never rename this container's spans.
func TestCalculateSpanRenamerConfig_ignoresOtherLanguages(t *testing.T) {
	actionsList := []odigosv1.Action{
		spanRenamerAction(common.PythonProgrammingLanguage, "django", spanRenamerReplacement("^GET", "get")),
	}

	got := CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage)

	require.Nil(t, got)
}

// An action with no replacements contributes nothing, so it must not produce an empty scope entry
// that the agent would have to interpret.
func TestCalculateSpanRenamerConfig_actionWithoutReplacements(t *testing.T) {
	actionsList := []odigosv1.Action{spanRenamerAction(common.JavaProgrammingLanguage, "jdbc")}

	got := CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage)

	require.Nil(t, got)
}

func TestCalculateSpanRenamerConfig_singleAction(t *testing.T) {
	replacement := spanRenamerReplacement(`^SELECT (\w+)`, "select $1")
	actionsList := []odigosv1.Action{spanRenamerAction(common.JavaProgrammingLanguage, "jdbc", replacement)}

	got := CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage)

	require.NotNil(t, got)
	require.Len(t, got.ScopeRules, 1)
	require.Equal(t, "jdbc", got.ScopeRules[0].ScopeName)
	require.Equal(t, []actions.SpanRenamerRegexReplacement{replacement}, got.ScopeRules[0].RegexReplacements)
}

// Two actions targeting the same scope have to end up in one scope entry, otherwise the agent
// would only apply one of them.
func TestCalculateSpanRenamerConfig_mergesActionsOfTheSameScope(t *testing.T) {
	first := spanRenamerReplacement("^SELECT", "select")
	second := spanRenamerReplacement("^INSERT", "insert")
	third := spanRenamerReplacement("^DELETE", "delete")
	actionsList := []odigosv1.Action{
		spanRenamerAction(common.JavaProgrammingLanguage, "jdbc", first, second),
		spanRenamerAction(common.JavaProgrammingLanguage, "jdbc", third),
	}

	got := CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage)

	require.NotNil(t, got)
	require.Len(t, got.ScopeRules, 1)
	require.Equal(t, []actions.SpanRenamerRegexReplacement{first, second, third}, scopeRuleByName(t, got, "jdbc").RegexReplacements)
}

func TestCalculateSpanRenamerConfig_separateScopesAreKeptApart(t *testing.T) {
	jdbcReplacement := spanRenamerReplacement("^SELECT", "select")
	kafkaReplacement := spanRenamerReplacement("^send", "publish")
	actionsList := []odigosv1.Action{
		spanRenamerAction(common.JavaProgrammingLanguage, "jdbc", jdbcReplacement),
		spanRenamerAction(common.JavaProgrammingLanguage, "kafka", kafkaReplacement),
		spanRenamerAction(common.PythonProgrammingLanguage, "django", spanRenamerReplacement("^GET", "get")),
	}

	got := CalculateSpanRenamerConfig(spanRenamerSupportingDistro, &actionsList, common.JavaProgrammingLanguage)

	require.NotNil(t, got)
	require.Equal(t, []string{"jdbc", "kafka"}, scopeNames(got))
	require.Equal(t, []actions.SpanRenamerRegexReplacement{jdbcReplacement}, scopeRuleByName(t, got, "jdbc").RegexReplacements)
	require.Equal(t, []actions.SpanRenamerRegexReplacement{kafkaReplacement}, scopeRuleByName(t, got, "kafka").RegexReplacements)
}
