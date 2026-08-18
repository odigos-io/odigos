package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func traceVerbosityRule(tv *instrumentationrules.TraceVerbosity) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{TraceVerbosity: tv},
	}
}

func javaLibrary(name string) instrumentationrules.InstrumentationLibrary {
	return instrumentationrules.InstrumentationLibrary{Language: common.JavaProgrammingLanguage, LibraryName: name}
}

func pythonLibrary(name string) instrumentationrules.InstrumentationLibrary {
	return instrumentationrules.InstrumentationLibrary{Language: common.PythonProgrammingLanguage, LibraryName: name}
}

var traceVerbositySupportingDistro = distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
	TraceVerbosity: &distro.TraceVerbosity{DisablingOdigosAgentLibrariesSupported: true},
})

func TestDistroSupportsTracesVerbosity(t *testing.T) {
	require.False(t, DistroSupportsTracesVerbosity(distroWithTraces(common.JavaProgrammingLanguage, nil)))
	require.False(t, DistroSupportsTracesVerbosity(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{})))
	// unlike the other trace features there is no Supported flag; the presence of the entry is the gate
	require.True(t, DistroSupportsTracesVerbosity(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		TraceVerbosity: &distro.TraceVerbosity{},
	})))
}

func TestCalculateTraceVerbosityConfig_unsupportedDistro(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{traceVerbosityRule(&instrumentationrules.TraceVerbosity{
		DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
	})}

	require.Nil(t, CalculateTraceVerbosityConfig(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{}), &irls))
}

// A supported distro always gets a config, so the agent knows verbosity is under odigos control
// even when no rule configures it.
func TestCalculateTraceVerbosityConfig_noRulesReturnsEmptyConfig(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{}
	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.NotNil(t, got)
	require.Empty(t, got.DisabledLibraries)
	require.Empty(t, got.EnabledLibraries)
}

func TestCalculateTraceVerbosityConfig_nilRulesReturnsEmptyConfig(t *testing.T) {
	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, nil)

	require.NotNil(t, got)
	require.Empty(t, got.DisabledLibraries)
	require.Empty(t, got.EnabledLibraries)
}

func TestCalculateTraceVerbosityConfig_singleRule(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{traceVerbosityRule(&instrumentationrules.TraceVerbosity{
		DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
		EnabledLibraries:  []instrumentationrules.InstrumentationLibrary{javaLibrary("kafka")},
	})}

	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")}, got.DisabledLibraries)
	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("kafka")}, got.EnabledLibraries)
}

// Rules are cluster-wide but the config is per container: a library entry for another language
// must never reach this distro's agent.
func TestCalculateTraceVerbosityConfig_filtersOtherLanguages(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{traceVerbosityRule(&instrumentationrules.TraceVerbosity{
		DisabledLibraries: []instrumentationrules.InstrumentationLibrary{
			pythonLibrary("django"),
			javaLibrary("jdbc"),
			pythonLibrary("flask"),
		},
		EnabledLibraries: []instrumentationrules.InstrumentationLibrary{
			pythonLibrary("urllib"),
			javaLibrary("kafka"),
		},
	})}

	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")}, got.DisabledLibraries)
	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("kafka")}, got.EnabledLibraries)
}

// A rule that only mentions other languages contributes nothing, and must not blank out what the
// previous rules already asked for.
func TestCalculateTraceVerbosityConfig_ruleForAnotherLanguageIsDropped(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
		}),
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			DisabledLibraries: []instrumentationrules.InstrumentationLibrary{pythonLibrary("django")},
			EnabledLibraries:  []instrumentationrules.InstrumentationLibrary{pythonLibrary("flask")},
		}),
	}

	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")}, got.DisabledLibraries)
	require.Empty(t, got.EnabledLibraries)
}

func TestCalculateTraceVerbosityConfig_ruleWithoutVerbosityIsSkipped(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		traceVerbosityRule(nil),
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
		}),
		traceVerbosityRule(nil),
	}

	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.Equal(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")}, got.DisabledLibraries)
}

func TestCalculateTraceVerbosityConfig_concatenatesRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
			EnabledLibraries:  []instrumentationrules.InstrumentationLibrary{javaLibrary("kafka")},
		}),
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("redis")},
		}),
		traceVerbosityRule(&instrumentationrules.TraceVerbosity{
			EnabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("grpc")},
		}),
	}

	got := CalculateTraceVerbosityConfig(traceVerbositySupportingDistro, &irls)

	require.ElementsMatch(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc"), javaLibrary("redis")}, got.DisabledLibraries)
	require.ElementsMatch(t, []instrumentationrules.InstrumentationLibrary{javaLibrary("kafka"), javaLibrary("grpc")}, got.EnabledLibraries)
}

// A wildcard distro has no concrete language, so nothing matches the per-library language filter.
func TestCalculateTraceVerbosityConfig_wildcardLanguageDistro(t *testing.T) {
	wildcardDistro := distroWithTraces(common.ProgrammingLanguage("*"), &distro.Traces{
		TraceVerbosity: &distro.TraceVerbosity{DisablingAnyScopeSupported: true},
	})
	irls := []odigosv1.InstrumentationRule{traceVerbosityRule(&instrumentationrules.TraceVerbosity{
		DisabledLibraries: []instrumentationrules.InstrumentationLibrary{javaLibrary("jdbc")},
	})}

	got := CalculateTraceVerbosityConfig(wildcardDistro, &irls)

	require.NotNil(t, got)
	require.Empty(t, got.DisabledLibraries)
}
