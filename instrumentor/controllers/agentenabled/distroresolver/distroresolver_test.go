package distroresolver

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
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

func TestResolveDistroForContainer_prereleaseRuntimeVersionAccepted(t *testing.T) {
	g := mustNewCommunityGetter(t)
	config := &common.OdigosConfiguration{}
	rt := &odigosv1.RuntimeDetailsByContainer{
		Language:       common.GoProgrammingLanguage,
		RuntimeVersion: "v1.22.0-0",
	}
	dpl := map[common.ProgrammingLanguage]string{
		common.GoProgrammingLanguage: "golang-community",
	}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, nil, "c1")
	require.Nil(t, info, "prerelease suffix on runtime version should not fail supported version check")
	require.NotNil(t, d)
	require.Equal(t, "golang-community", d.Name)
}

func TestResolveDistroForContainer_browserOverrideAcceptsMismatchedContainerLanguage(t *testing.T) {
	g := mustNewCommunityGetter(t)
	overrideName := "browser-community"
	config := &common.OdigosConfiguration{}
	// A frontend served by a Node static server is detected as server-side javascript; the explicit
	// browser override must still win since the frontend nature cannot be auto-detected.
	rt := &odigosv1.RuntimeDetailsByContainer{Language: common.JavascriptProgrammingLanguage}
	dpl := map[common.ProgrammingLanguage]string{common.JavascriptProgrammingLanguage: "nodejs-community"}
	co := &odigosv1.ContainerOverride{OtelDistroName: &overrideName}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, co, "frontend")
	require.Nil(t, info)
	require.NotNil(t, d)
	require.Equal(t, "browser-community", d.Name)
	require.NotNil(t, d.BrowserSidecar, "browser-community must carry a browserSidecar marker")
}

func TestResolveDistroForContainer_browserOverrideAcceptsUnknownLanguage(t *testing.T) {
	g := mustNewCommunityGetter(t)
	overrideName := "browser-community"
	config := &common.OdigosConfiguration{}
	// A plain static-file server may not be detected at all; an explicit browser override must
	// still be honored (the unknown-language early return is skipped when an override is present).
	rt := &odigosv1.RuntimeDetailsByContainer{Language: common.UnknownProgrammingLanguage}
	dpl := map[common.ProgrammingLanguage]string{}
	co := &odigosv1.ContainerOverride{OtelDistroName: &overrideName}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, co, "frontend")
	require.Nil(t, info)
	require.NotNil(t, d)
	require.Equal(t, "browser-community", d.Name)
}

func TestResolveDistroForContainer_browserByLanguageOverride(t *testing.T) {
	g := mustNewCommunityGetter(t)
	config := &common.OdigosConfiguration{}
	// Opting in by setting the runtime language to browser (via containerOverride RuntimeInfo)
	// resolves to the default browser distro without an explicit distro name.
	rt := &odigosv1.RuntimeDetailsByContainer{Language: common.BrowserProgrammingLanguage}
	dpl := map[common.ProgrammingLanguage]string{common.BrowserProgrammingLanguage: "browser-community"}

	d, info := ResolveDistroForContainer(config, rt, dpl, g, nil, "frontend")
	require.Nil(t, info)
	require.NotNil(t, d)
	require.Equal(t, "browser-community", d.Name)
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
