package graph

import (
	"fmt"
	"slices"
	"testing"
	"time"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func whBool(b bool) *bool { return &b }

// whIC builds an InstrumentationConfig from the three inputs
// collectEffectiveDetectedLanguages actually branches on: the per-container
// overrides (which win), the automatic detection results, and nothing else.
func whIC(overrides []odigosv1.ContainerOverride, detected []odigosv1.RuntimeDetailsByContainer) *odigosv1.InstrumentationConfig {
	return &odigosv1.InstrumentationConfig{
		Spec:   odigosv1.InstrumentationConfigSpec{ContainersOverrides: overrides},
		Status: odigosv1.InstrumentationConfigStatus{RuntimeDetailsByContainer: detected},
	}
}

func whOverride(containerName string, lang common.ProgrammingLanguage) odigosv1.ContainerOverride {
	return odigosv1.ContainerOverride{
		ContainerName: containerName,
		RuntimeInfo:   &odigosv1.RuntimeDetailsByContainer{ContainerName: containerName, Language: lang},
	}
}

func whDetected(containerName string, lang common.ProgrammingLanguage) odigosv1.RuntimeDetailsByContainer {
	return odigosv1.RuntimeDetailsByContainer{ContainerName: containerName, Language: lang}
}

func whIgnored(names ...string) map[string]struct{} {
	ignored := make(map[string]struct{}, len(names))
	for _, name := range names {
		ignored[name] = struct{}{}
	}
	return ignored
}

func whProcessAttr(name, value string) *model.K8sWorkloadPodContainerProcessAttribute {
	return &model.K8sWorkloadPodContainerProcessAttribute{Name: name, Value: value}
}

func whProcess(attrs ...*model.K8sWorkloadPodContainerProcessAttribute) *model.K8sWorkloadPodContainerProcess {
	return &model.K8sWorkloadPodContainerProcess{IdentifyingAttributes: attrs}
}

func TestCollectEffectiveDetectedLanguages_NilInstrumentationConfig(t *testing.T) {
	assert.Nil(t, collectEffectiveDetectedLanguages(nil, nil))
}

// The override is the user's explicit correction of a wrong auto-detection. If the
// detected value were to win, the UI would keep showing the language the user just
// corrected, and there would be no way to fix it from the product.
func TestCollectEffectiveDetectedLanguages_TheOverrideWinsOverTheDetectedLanguage(t *testing.T) {
	ic := whIC(
		[]odigosv1.ContainerOverride{whOverride("app", common.PythonProgrammingLanguage)},
		[]odigosv1.RuntimeDetailsByContainer{whDetected("app", common.JavaProgrammingLanguage)},
	)

	got := collectEffectiveDetectedLanguages(ic, nil)

	assert.Equal(t, []model.ProgrammingLanguage{model.ProgrammingLanguage(common.PythonProgrammingLanguage)}, got)
}

// An override entry with no RuntimeInfo is the normal shape: ContainersOverrides lists
// every container of the workload, and RuntimeInfo is only set when the user corrects
// the detection. Such a container must still report its auto-detected language.
func TestCollectEffectiveDetectedLanguages_AnOverrideWithoutRuntimeInfoFallsBackToDetection(t *testing.T) {
	ic := whIC(
		[]odigosv1.ContainerOverride{{ContainerName: "app"}},
		[]odigosv1.RuntimeDetailsByContainer{whDetected("app", common.JavaProgrammingLanguage)},
	)

	got := collectEffectiveDetectedLanguages(ic, nil)

	assert.Equal(t, []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)}, got)
}

// The defensive second loop: ContainersOverrides is supposed to list every container,
// but a container present only in the detection results must not silently lose its
// language.
func TestCollectEffectiveDetectedLanguages_AContainerMissingFromTheOverridesIsStillCollected(t *testing.T) {
	ic := whIC(
		[]odigosv1.ContainerOverride{whOverride("app", common.PythonProgrammingLanguage)},
		[]odigosv1.RuntimeDetailsByContainer{whDetected("sidecar", common.GoProgrammingLanguage)},
	)

	got := collectEffectiveDetectedLanguages(ic, nil)

	assert.Equal(t, []model.ProgrammingLanguage{
		model.ProgrammingLanguage(common.GoProgrammingLanguage),
		model.ProgrammingLanguage(common.PythonProgrammingLanguage),
	}, got)
}

// Both loops filter ignored containers and unknown languages, and each filter has to be
// proven on both loops separately: dropping either one leaks a language into the UI.
func TestCollectEffectiveDetectedLanguages_IgnoredAndUnknownAreFilteredOnBothLoops(t *testing.T) {
	for _, tt := range []struct {
		name      string
		overrides []odigosv1.ContainerOverride
		detected  []odigosv1.RuntimeDetailsByContainer
		ignored   map[string]struct{}
		want      []model.ProgrammingLanguage
	}{
		{
			name:      "an ignored container in the overrides loop is dropped",
			overrides: []odigosv1.ContainerOverride{whOverride("app", common.JavaProgrammingLanguage), whOverride("istio-proxy", common.GoProgrammingLanguage)},
			ignored:   whIgnored("istio-proxy"),
			want:      []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)},
		},
		{
			name:      "an ignored container in the fallback loop is dropped",
			overrides: []odigosv1.ContainerOverride{whOverride("app", common.JavaProgrammingLanguage)},
			detected:  []odigosv1.RuntimeDetailsByContainer{whDetected("istio-proxy", common.GoProgrammingLanguage)},
			ignored:   whIgnored("istio-proxy"),
			want:      []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)},
		},
		{
			name:      "an unknown language in the overrides loop is dropped",
			overrides: []odigosv1.ContainerOverride{whOverride("app", common.JavaProgrammingLanguage), whOverride("mystery", common.UnknownProgrammingLanguage)},
			want:      []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)},
		},
		{
			name:      "an unknown language in the fallback loop is dropped",
			overrides: []odigosv1.ContainerOverride{whOverride("app", common.JavaProgrammingLanguage)},
			detected:  []odigosv1.RuntimeDetailsByContainer{whDetected("mystery", common.UnknownProgrammingLanguage)},
			want:      []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)},
		},
		{
			name:      "a container listed in the overrides with neither an override nor a detection is dropped",
			overrides: []odigosv1.ContainerOverride{whOverride("app", common.JavaProgrammingLanguage), {ContainerName: "empty"}},
			want:      []model.ProgrammingLanguage{model.ProgrammingLanguage(common.JavaProgrammingLanguage)},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, collectEffectiveDetectedLanguages(whIC(tt.overrides, tt.detected), tt.ignored))
		})
	}
}

// The result feeds a language filter in the UI, so it has to be de-duplicated and
// sorted rather than reflecting Go's randomised map iteration order.
func TestCollectEffectiveDetectedLanguages_TheResultIsUniqueAndSorted(t *testing.T) {
	ic := whIC(
		[]odigosv1.ContainerOverride{
			whOverride("c-python", common.PythonProgrammingLanguage),
			whOverride("a-java", common.JavaProgrammingLanguage),
			whOverride("b-java", common.JavaProgrammingLanguage),
		},
		[]odigosv1.RuntimeDetailsByContainer{whDetected("d-go", common.GoProgrammingLanguage)},
	)

	// Run repeatedly: a missing sort only shows up against map iteration randomisation.
	for range 10 {
		got := collectEffectiveDetectedLanguages(ic, nil)
		assert.Equal(t, []model.ProgrammingLanguage{
			model.ProgrammingLanguage(common.GoProgrammingLanguage),
			model.ProgrammingLanguage(common.JavaProgrammingLanguage),
			model.ProgrammingLanguage(common.PythonProgrammingLanguage),
		}, got)
	}
}

// An empty InstrumentationConfig must produce an empty, non-nil slice — the GraphQL
// schema declares detectedLanguages as a list and the UI iterates it directly.
func TestCollectEffectiveDetectedLanguages_NoContainersYieldsAnEmptyNonNilSlice(t *testing.T) {
	got := collectEffectiveDetectedLanguages(whIC(nil, nil), nil)

	require.NotNil(t, got)
	assert.Empty(t, got)
}

// The whole point of the rank is that "worse" sorts lower, so aggregation can pick the
// minimum. Pinning the relative order (rather than the literals) is what the caller
// depends on.
func TestInstrumentationHealthRank_UnhealthyIsWorseThanUnknownIsWorseThanHealthy(t *testing.T) {
	unhealthy := instrumentationHealthRank(whBool(false))
	unknown := instrumentationHealthRank(nil)
	healthy := instrumentationHealthRank(whBool(true))

	assert.Less(t, unhealthy, unknown, "unhealthy must rank below unknown")
	assert.Less(t, unknown, healthy, "unknown must rank below healthy")
}

// The workload container view aggregates one instrumentation library across every
// instrumentation instance of the workload and must surface the least healthy report.
// If this comparison is inverted or made non-strict, a single healthy process hides a
// crashing one from the UI, so every ordered pair needs its own assertion.
func TestShouldReplaceInstrumentationForAggregation_TheLeastHealthyReportWins(t *testing.T) {
	unhealthy, unknown, healthy := whBool(false), (*bool)(nil), whBool(true)

	for _, tt := range []struct {
		name     string
		existing *bool
		current  *bool
		want     bool
	}{
		{"unhealthy replaces healthy", healthy, unhealthy, true},
		{"unhealthy replaces unknown", unknown, unhealthy, true},
		{"unknown replaces healthy", healthy, unknown, true},
		{"healthy does not replace unhealthy", unhealthy, healthy, false},
		{"healthy does not replace unknown", unknown, healthy, false},
		{"unknown does not replace unhealthy", unhealthy, unknown, false},
		{"unhealthy does not replace an equally unhealthy report", unhealthy, unhealthy, false},
		{"unknown does not replace an equally unknown report", unknown, unknown, false},
		{"healthy does not replace an equally healthy report", healthy, healthy, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, shouldReplaceInstrumentationForAggregation(tt.existing, tt.current))
		})
	}
}

func TestComponentToInstrumentation_EveryPopulatedFieldIsMapped(t *testing.T) {
	statusTime := metav1.Date(2026, time.March, 4, 5, 6, 7, 0, time.UTC)
	component := odigosv1.InstrumentationLibraryStatus{
		Name:           "net/http",
		Type:           odigosv1.InstrumentationLibraryTypeInstrumentation,
		Healthy:        whBool(true),
		Message:        "all good",
		LastStatusTime: statusTime,
		NonIdentifyingAttributes: []odigosv1.Attribute{
			{Key: "is_standard_lib", Value: "true"},
			{Key: "version", Value: "1.2.3"},
		},
	}

	got := componentToInstrumentation(component)

	assert.Equal(t, "net/http", got.Name)
	require.NotNil(t, got.Type)
	assert.Equal(t, string(odigosv1.InstrumentationLibraryTypeInstrumentation), *got.Type)
	assert.Equal(t, whBool(true), got.Healthy)
	require.NotNil(t, got.Message)
	assert.Equal(t, "all good", *got.Message)
	require.NotNil(t, got.LastStatusTime)
	assert.Equal(t, "2026-03-04T05:06:07Z", *got.LastStatusTime)
	require.NotNil(t, got.IsStandardLibrary)
	assert.True(t, *got.IsStandardLibrary)
	assert.Equal(t, []*model.NonIdentifyingAttribute{
		{Key: "is_standard_lib", Value: "true"},
		{Key: "version", Value: "1.2.3"},
	}, got.NonIdentifyingAttributes)
}

// Each optional field has its own "absent" encoding. Asserting them one at a time is
// what distinguishes "the field was omitted" from "the field was mapped from the wrong
// source" — a fully populated fixture cannot tell the two apart.
func TestComponentToInstrumentation_AbsentOptionalFieldsStayNil(t *testing.T) {
	got := componentToInstrumentation(odigosv1.InstrumentationLibraryStatus{Name: "bare"})

	assert.Equal(t, "bare", got.Name)
	assert.Nil(t, got.Type, "an empty type must not be mapped to a pointer to the empty string")
	assert.Nil(t, got.Healthy)
	assert.Nil(t, got.Message, "an empty message must not be mapped to a pointer to the empty string")
	assert.Nil(t, got.LastStatusTime, "a zero status time must not be rendered")
	assert.Nil(t, got.IsStandardLibrary, "without the is_standard_lib attribute the flag is unknown, not false")
	require.NotNil(t, got.NonIdentifyingAttributes)
	assert.Empty(t, got.NonIdentifyingAttributes)
}

// is_standard_lib arrives as a string and drives whether the UI folds the library into
// the "standard library" group. Only the exact literal "true" may enable it.
func TestComponentToInstrumentation_IsStandardLibraryIsParsedFromTheAttributeValue(t *testing.T) {
	for _, tt := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"false", false},
		{"True", false},
		{"1", false},
		{"", false},
	} {
		t.Run(fmt.Sprintf("value %q", tt.value), func(t *testing.T) {
			got := componentToInstrumentation(odigosv1.InstrumentationLibraryStatus{
				Name:                     "lib",
				NonIdentifyingAttributes: []odigosv1.Attribute{{Key: "is_standard_lib", Value: tt.value}},
			})

			require.NotNil(t, got.IsStandardLibrary)
			assert.Equal(t, tt.want, *got.IsStandardLibrary)
		})
	}
}

// A healthy=false component is the one the UI renders in red. It must survive the
// mapping as a non-nil false and not collapse into the "unknown" nil.
func TestComponentToInstrumentation_AnUnhealthyComponentIsNotReportedAsUnknown(t *testing.T) {
	got := componentToInstrumentation(odigosv1.InstrumentationLibraryStatus{Name: "lib", Healthy: whBool(false)})

	require.NotNil(t, got.Healthy)
	assert.False(t, *got.Healthy)
}

func TestProcessPidFromAttributes_PrefersPidOverVpid(t *testing.T) {
	// vpid comes first so a naive "last attribute wins" implementation is caught.
	got := processPidFromAttributes([]*model.K8sWorkloadPodContainerProcessAttribute{
		whProcessAttr(processAttributeNameVpid, "7"),
		whProcessAttr(processAttributeNamePid, "42"),
	})

	assert.Equal(t, "42", got)
}

func TestProcessPidFromAttributes_FallsBackToVpid(t *testing.T) {
	got := processPidFromAttributes([]*model.K8sWorkloadPodContainerProcessAttribute{
		whProcessAttr("process.command", "java"),
		whProcessAttr(processAttributeNameVpid, "7"),
	})

	assert.Equal(t, "7", got)
}

func TestProcessPidFromAttributes_NoPidAttributesYieldsTheEmptyString(t *testing.T) {
	assert.Empty(t, processPidFromAttributes(nil))
	assert.Empty(t, processPidFromAttributes([]*model.K8sWorkloadPodContainerProcessAttribute{
		whProcessAttr("process.command", "java"),
	}))
}

func TestProcessPidFromAttributes_NilAttributesAreSkipped(t *testing.T) {
	got := processPidFromAttributes([]*model.K8sWorkloadPodContainerProcessAttribute{
		nil,
		whProcessAttr(processAttributeNamePid, "42"),
	})

	assert.Equal(t, "42", got)
}

// The wire literals the agents emit. Nothing links these constants to the agents at
// compile time, so a rename here would silently leave every process unsorted.
func TestProcessPidAttributeNamesMatchTheAgentWireLiterals(t *testing.T) {
	assert.Equal(t, "process.pid", processAttributeNamePid)
	assert.Equal(t, "process.vpid", processAttributeNameVpid)
}

// "10" must sort after "2". A plain string comparison is the bug this guards: it is
// invisible with single-digit fixtures, which is why every numeric case here crosses a
// digit-count boundary.
func TestCompareProcessesByPid_NumericPidsSortNumerically(t *testing.T) {
	two := whProcess(whProcessAttr(processAttributeNamePid, "2"))
	ten := whProcess(whProcessAttr(processAttributeNamePid, "10"))

	assert.Negative(t, compareProcessesByPid(two, ten))
	assert.Positive(t, compareProcessesByPid(ten, two))
	assert.Zero(t, compareProcessesByPid(ten, ten))
}

func TestCompareProcessesByPid_FallsBackToLexicographicWhenAPidIsNotNumeric(t *testing.T) {
	numeric := whProcess(whProcessAttr(processAttributeNamePid, "10"))
	textual := whProcess(whProcessAttr(processAttributeNamePid, "abc"))

	assert.Negative(t, compareProcessesByPid(numeric, textual), "\"10\" sorts before \"abc\"")
	assert.Positive(t, compareProcessesByPid(textual, numeric))
}

// A process that only reports vpid must still be ordered against one that reports pid,
// rather than both collapsing onto the empty string.
func TestCompareProcessesByPid_ComparesAPidAgainstAVpid(t *testing.T) {
	pid := whProcess(whProcessAttr(processAttributeNamePid, "10"))
	vpid := whProcess(whProcessAttr(processAttributeNameVpid, "2"))

	assert.Positive(t, compareProcessesByPid(pid, vpid))
}

// The ordering is what keeps the process table stable across re-fetches, so assert the
// end-to-end sort and not only the pairwise comparator.
func TestCompareProcessesByPid_SortsAMixedProcessListDeterministically(t *testing.T) {
	noPid := whProcess()
	vpidTwo := whProcess(whProcessAttr(processAttributeNameVpid, "2"))
	pidNine := whProcess(whProcessAttr(processAttributeNamePid, "9"))
	pidTen := whProcess(whProcessAttr(processAttributeNamePid, "10"))

	processes := []*model.K8sWorkloadPodContainerProcess{pidTen, noPid, pidNine, vpidTwo}
	slices.SortFunc(processes, compareProcessesByPid)

	assert.Equal(t, []*model.K8sWorkloadPodContainerProcess{noPid, vpidTwo, pidNine, pidTen}, processes)
}
