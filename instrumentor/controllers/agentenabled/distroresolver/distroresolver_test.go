package distroresolver

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/stretchr/testify/require"
)

const obiDistroName = "opentelemetry-ebpf-instrumentation"

func mustNewCommunityGetter(t *testing.T) *distros.Getter {
	t.Helper()
	g, err := distros.NewCommunityGetter()
	require.NoError(t, err)
	return g
}

func TestCalculateDefaultDistroPerLanguage_skipsWildcardDistroFromRules(t *testing.T) {
	g := mustNewCommunityGetter(t)
	defaults := map[common.ProgrammingLanguage]string{
		common.JavaProgrammingLanguage: "java-community",
	}
	rules := []odigosv1.InstrumentationRule{{
		Spec: odigosv1.InstrumentationRuleSpec{
			OtelDistros: &instrumentationrules.OtelDistros{
				OtelDistroNames: []string{obiDistroName},
			},
		},
	}}
	out := CalculateDefaultDistroPerLanguage(defaults, &rules, g)
	_, hasWildcard := out[common.ProgrammingLanguageWildcard]
	require.False(t, hasWildcard, "OBI (language *) must not add a map entry for the wildcard key")
	require.Equal(t, "java-community", out[common.JavaProgrammingLanguage])
}

func TestCalculateDefaultDistroPerLanguage_mixedRuleWithOBIAndJavaStillMapsJava(t *testing.T) {
	g := mustNewCommunityGetter(t)
	defaults := map[common.ProgrammingLanguage]string{
		common.JavaProgrammingLanguage: "java-community",
	}
	rules := []odigosv1.InstrumentationRule{{
		Spec: odigosv1.InstrumentationRuleSpec{
			OtelDistros: &instrumentationrules.OtelDistros{
				OtelDistroNames: []string{obiDistroName, "java-community"},
			},
		},
	}}
	out := CalculateDefaultDistroPerLanguage(defaults, &rules, g)
	_, hasWildcard := out[common.ProgrammingLanguageWildcard]
	require.False(t, hasWildcard, "OBI in a mixed list must not create a wildcard map entry")
	require.Equal(t, "java-community", out[common.JavaProgrammingLanguage])
}

func TestResolveDistroForContainer_wildcardOverrideAcceptsMismatchedContainerLanguage(t *testing.T) {
	g := mustNewCommunityGetter(t)
	overrideName := obiDistroName
	config := &common.OdigosConfiguration{}
	rt := &odigosv1.RuntimeDetailsByContainer{
		Language:       common.GoProgrammingLanguage,
		RuntimeVersion: "1.22.0",
	}
	dpl := map[common.ProgrammingLanguage]string{common.GoProgrammingLanguage: "golang-community"}
	co := &odigosv1.ContainerOverride{OtelDistroName: &overrideName}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, co, "c1")
	require.Nil(t, info)
	require.NotNil(t, d)
	require.Equal(t, obiDistroName, d.Name)
	require.True(t, common.IsProgrammingLanguageWildcard(d.Language), "OBI should report wildcard language in spec")
}

func TestResolveDistroForContainer_wildcardDistroSkipsRuntimeSemver(t *testing.T) {
	g := mustNewCommunityGetter(t)
	config := &common.OdigosConfiguration{}
	rt := &odigosv1.RuntimeDetailsByContainer{
		Language:       common.GoProgrammingLanguage,
		RuntimeVersion: "not-a-valid-constraint-check-999.0.0-xyz",
	}
	dpl := map[common.ProgrammingLanguage]string{
		common.GoProgrammingLanguage: obiDistroName,
	}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, nil, "c1")
	require.Nil(t, info, "wildcard-language distro should skip semver resolution against supportedVersions (e.g. *)")
	require.NotNil(t, d)
	require.Equal(t, obiDistroName, d.Name)
}

func TestResolveDistroForContainer_nonWildcardEnforcesRuntimeSemver(t *testing.T) {
	g := mustNewCommunityGetter(t)
	config := &common.OdigosConfiguration{}
	rt := &odigosv1.RuntimeDetailsByContainer{
		Language:       common.GoProgrammingLanguage,
		RuntimeVersion: "1.18.0",
	}
	dpl := map[common.ProgrammingLanguage]string{
		common.GoProgrammingLanguage: "golang-community",
	}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, nil, "c1")
	require.Nil(t, d)
	require.NotNil(t, info)
	require.Equal(t, odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion, info.AgentEnabledReason)
}
