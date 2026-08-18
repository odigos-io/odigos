package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func customInstrumentationsRule(ci *instrumentationrules.CustomInstrumentations) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{CustomInstrumentations: ci},
	}
}

func customInstrumentationsDistro(language common.ProgrammingLanguage) *distro.OtelDistro {
	return distroWithTraces(language, &distro.Traces{
		CustomInstrumentations: &distro.CustomInstrumentations{Supported: true},
	})
}

var (
	goProbe   = instrumentationrules.GolangCustomProbe{PackageName: "net/http", FunctionName: "ListenAndServe"}
	javaProbe = instrumentationrules.JavaCustomProbe{ClassName: "com.acme.Service", MethodName: "handle"}
	phpProbe  = instrumentationrules.PhpCustomProbe{ClassName: "Acme\\Service", FunctionName: "handle"}
)

func TestDistroSupportsCustomInstrumentations(t *testing.T) {
	require.False(t, DistroSupportsCustomInstrumentations(distroWithTraces(common.GoProgrammingLanguage, nil)))
	require.False(t, DistroSupportsCustomInstrumentations(distroWithTraces(common.GoProgrammingLanguage, &distro.Traces{})))
	require.False(t, DistroSupportsCustomInstrumentations(distroWithTraces(common.GoProgrammingLanguage, &distro.Traces{
		CustomInstrumentations: &distro.CustomInstrumentations{Supported: false},
	})))
	require.True(t, DistroSupportsCustomInstrumentations(customInstrumentationsDistro(common.GoProgrammingLanguage)))
}

func TestCalculateCustomInstrumentationsConfig_unsupportedDistro(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{customInstrumentationsRule(&instrumentationrules.CustomInstrumentations{
		Golang: []instrumentationrules.GolangCustomProbe{goProbe},
	})}

	require.Nil(t, CalculateCustomInstrumentationsConfig(distroWithTraces(common.GoProgrammingLanguage, &distro.Traces{
		CustomInstrumentations: &distro.CustomInstrumentations{Supported: false},
	}), &irls))
}

func TestCalculateCustomInstrumentationsConfig_noRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{}
	require.Nil(t, CalculateCustomInstrumentationsConfig(customInstrumentationsDistro(common.GoProgrammingLanguage), &irls))
}

func TestCalculateCustomInstrumentationsConfig_rulesWithoutProbes(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{customInstrumentationsRule(nil), customInstrumentationsRule(nil)}
	require.Nil(t, CalculateCustomInstrumentationsConfig(customInstrumentationsDistro(common.GoProgrammingLanguage), &irls))
}

// Only the probes of the container's own language are forwarded; a probe list for another language
// would be meaningless (and possibly rejected) by the agent.
func TestCalculateCustomInstrumentationsConfig_keepsOnlyTheDistroLanguage(t *testing.T) {
	allLanguages := &instrumentationrules.CustomInstrumentations{
		Golang: []instrumentationrules.GolangCustomProbe{goProbe},
		Java:   []instrumentationrules.JavaCustomProbe{javaProbe},
		Php:    []instrumentationrules.PhpCustomProbe{phpProbe},
	}

	tt := []struct {
		name     string
		language common.ProgrammingLanguage
		assert   func(t *testing.T, got *instrumentationrules.CustomInstrumentations)
	}{
		{
			name:     "go",
			language: common.GoProgrammingLanguage,
			assert: func(t *testing.T, got *instrumentationrules.CustomInstrumentations) {
				require.Equal(t, []instrumentationrules.GolangCustomProbe{goProbe}, got.Golang)
				require.Empty(t, got.Java)
				require.Empty(t, got.Php)
			},
		},
		{
			name:     "java",
			language: common.JavaProgrammingLanguage,
			assert: func(t *testing.T, got *instrumentationrules.CustomInstrumentations) {
				require.Equal(t, []instrumentationrules.JavaCustomProbe{javaProbe}, got.Java)
				require.Empty(t, got.Golang)
				require.Empty(t, got.Php)
			},
		},
		{
			name:     "php",
			language: common.PhpProgrammingLanguage,
			assert: func(t *testing.T, got *instrumentationrules.CustomInstrumentations) {
				require.Equal(t, []instrumentationrules.PhpCustomProbe{phpProbe}, got.Php)
				require.Empty(t, got.Golang)
				require.Empty(t, got.Java)
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			irls := []odigosv1.InstrumentationRule{customInstrumentationsRule(allLanguages)}

			got := CalculateCustomInstrumentationsConfig(customInstrumentationsDistro(tc.language), &irls)

			require.NotNil(t, got)
			tc.assert(t, got)
		})
	}
}

// A rule that carries probes only for other languages still creates a (non-nil, empty) config,
// which is what tells the agent the feature is managed but has nothing to run.
func TestCalculateCustomInstrumentationsConfig_ruleForAnotherLanguage(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{customInstrumentationsRule(&instrumentationrules.CustomInstrumentations{
		Java: []instrumentationrules.JavaCustomProbe{javaProbe},
	})}

	got := CalculateCustomInstrumentationsConfig(customInstrumentationsDistro(common.GoProgrammingLanguage), &irls)

	require.NotNil(t, got)
	require.Empty(t, got.Golang)
	require.Empty(t, got.Java)
}

func TestCalculateCustomInstrumentationsConfig_concatenatesProbesAcrossRules(t *testing.T) {
	secondProbe := instrumentationrules.GolangCustomProbe{
		PackageName:        "database/sql",
		ReceiverName:       "DB",
		ReceiverMethodName: "QueryContext",
	}
	irls := []odigosv1.InstrumentationRule{
		customInstrumentationsRule(&instrumentationrules.CustomInstrumentations{
			Golang: []instrumentationrules.GolangCustomProbe{goProbe},
		}),
		customInstrumentationsRule(nil),
		customInstrumentationsRule(&instrumentationrules.CustomInstrumentations{
			Golang: []instrumentationrules.GolangCustomProbe{secondProbe},
		}),
	}

	got := CalculateCustomInstrumentationsConfig(customInstrumentationsDistro(common.GoProgrammingLanguage), &irls)

	require.NotNil(t, got)
	require.Equal(t, []instrumentationrules.GolangCustomProbe{goProbe, secondProbe}, got.Golang)
}
