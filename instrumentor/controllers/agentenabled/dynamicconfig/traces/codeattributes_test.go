package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func codeAttributesRule(ca *instrumentationrules.CodeAttributes) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{CodeAttributes: ca},
	}
}

var codeAttributesSupportingDistro = distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
	CodeAttributes: &distro.CodeAttributes{Supported: true},
})

func TestDistroSupportsTracesCodeAttributes(t *testing.T) {
	require.False(t, DistroSupportsTracesCodeAttributes(distroWithTraces(common.JavaProgrammingLanguage, nil)))
	require.False(t, DistroSupportsTracesCodeAttributes(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{})))
	require.False(t, DistroSupportsTracesCodeAttributes(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		CodeAttributes: &distro.CodeAttributes{Supported: false},
	})))
	require.True(t, DistroSupportsTracesCodeAttributes(codeAttributesSupportingDistro))
}

func TestCalculateCodeAttributesConfig_unsupportedDistro(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{codeAttributesRule(&instrumentationrules.CodeAttributes{
		Function: newPtr(true),
	})}

	require.Nil(t, CalculateCodeAttributesConfig(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		CodeAttributes: &distro.CodeAttributes{Supported: false},
	}), &irls))
}

func TestCalculateCodeAttributesConfig_noRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{}
	require.Nil(t, CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls))
}

func TestCalculateCodeAttributesConfig_rulesWithoutCodeAttributes(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{codeAttributesRule(nil), codeAttributesRule(nil)}
	require.Nil(t, CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls))
}

func TestCalculateCodeAttributesConfig_unrelatedRuleDoesNotClearTheConfig(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		codeAttributesRule(&instrumentationrules.CodeAttributes{Function: newPtr(true)}),
		codeAttributesRule(nil),
	}

	got := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls)

	require.NotNil(t, got)
	require.True(t, *got.Function)
}

func TestCalculateCodeAttributesConfig_singleRulePassedThrough(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{codeAttributesRule(&instrumentationrules.CodeAttributes{
		Function:   newPtr(true),
		LineNumber: newPtr(false),
	})}

	got := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls)

	require.NotNil(t, got)
	require.True(t, *got.Function)
	require.False(t, *got.LineNumber)
	require.Nil(t, got.Column)
	require.Nil(t, got.FilePath)
	require.Nil(t, got.Namespace)
	require.Nil(t, got.Stacktrace)
}

// Every field merges independently with an OR, so one rule enabling an attribute is enough and a
// field left unset by one rule keeps the other rule's value.
func TestCalculateCodeAttributesConfig_mergesEachFieldIndependently(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		codeAttributesRule(&instrumentationrules.CodeAttributes{
			Column:     newPtr(true),
			FilePath:   newPtr(false),
			Function:   newPtr(false),
			LineNumber: newPtr(true),
		}),
		codeAttributesRule(&instrumentationrules.CodeAttributes{
			FilePath:   newPtr(true),
			Function:   newPtr(false),
			Namespace:  newPtr(true),
			Stacktrace: newPtr(false),
		}),
	}

	got := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls)

	require.NotNil(t, got)
	require.True(t, *got.Column, "set only by the first rule")
	require.True(t, *got.FilePath, "false in the first rule, true in the second")
	require.False(t, *got.Function, "false in both rules")
	require.True(t, *got.LineNumber, "set only by the first rule")
	require.True(t, *got.Namespace, "set only by the second rule")
	require.False(t, *got.Stacktrace, "set only by the second rule")
}

// Each attribute has to survive the merge under its own name; a swap between two of them would
// silently record the wrong source location on every span.
func TestCalculateCodeAttributesConfig_eachFieldKeepsItsIdentity(t *testing.T) {
	fields := map[string]struct {
		set func(ca *instrumentationrules.CodeAttributes)
		get func(ca *instrumentationrules.CodeAttributes) *bool
	}{
		"column":     {func(ca *instrumentationrules.CodeAttributes) { ca.Column = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.Column }},
		"filePath":   {func(ca *instrumentationrules.CodeAttributes) { ca.FilePath = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.FilePath }},
		"function":   {func(ca *instrumentationrules.CodeAttributes) { ca.Function = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.Function }},
		"lineNumber": {func(ca *instrumentationrules.CodeAttributes) { ca.LineNumber = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.LineNumber }},
		"namespace":  {func(ca *instrumentationrules.CodeAttributes) { ca.Namespace = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.Namespace }},
		"stacktrace": {func(ca *instrumentationrules.CodeAttributes) { ca.Stacktrace = newPtr(true) }, func(ca *instrumentationrules.CodeAttributes) *bool { return ca.Stacktrace }},
	}

	for name, field := range fields {
		t.Run(name, func(t *testing.T) {
			onlyThisField := &instrumentationrules.CodeAttributes{}
			field.set(onlyThisField)
			irls := []odigosv1.InstrumentationRule{
				codeAttributesRule(onlyThisField),
				codeAttributesRule(&instrumentationrules.CodeAttributes{}),
			}

			got := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &irls)

			require.NotNil(t, got)
			for otherName, otherField := range fields {
				if otherName == name {
					require.NotNil(t, otherField.get(got), "%s must be set", otherName)
					require.True(t, *otherField.get(got))
					continue
				}
				require.Nil(t, otherField.get(got), "%s must not be set by the %s rule", otherName, name)
			}
		})
	}
}

func TestCalculateCodeAttributesConfig_mergeIsOrderIndependent(t *testing.T) {
	first := &instrumentationrules.CodeAttributes{Function: newPtr(true), Stacktrace: newPtr(false)}
	second := &instrumentationrules.CodeAttributes{Function: newPtr(false), Namespace: newPtr(true)}

	forward := []odigosv1.InstrumentationRule{codeAttributesRule(first), codeAttributesRule(second)}
	reversed := []odigosv1.InstrumentationRule{codeAttributesRule(second), codeAttributesRule(first)}

	gotForward := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &forward)
	gotReversed := CalculateCodeAttributesConfig(codeAttributesSupportingDistro, &reversed)

	require.Equal(t, *gotForward.Function, *gotReversed.Function)
	require.Equal(t, *gotForward.Namespace, *gotReversed.Namespace)
	require.Equal(t, *gotForward.Stacktrace, *gotReversed.Stacktrace)
	require.True(t, *gotForward.Function)
}

func TestMerge2RuleBooleans(t *testing.T) {
	tt := []struct {
		name     string
		value1   *bool
		value2   *bool
		expected *bool
	}{
		{name: "both unset", expected: nil},
		{name: "only the second set", value2: newPtr(true), expected: newPtr(true)},
		{name: "only the first set", value1: newPtr(false), expected: newPtr(false)},
		{name: "true wins over false", value1: newPtr(false), value2: newPtr(true), expected: newPtr(true)},
		{name: "true wins over false in reverse", value1: newPtr(true), value2: newPtr(false), expected: newPtr(true)},
		{name: "both false", value1: newPtr(false), value2: newPtr(false), expected: newPtr(false)},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			got := merge2RuleBooleans(tc.value1, tc.value2)
			if tc.expected == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tc.expected, *got)
		})
	}
}
