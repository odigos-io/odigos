package flamegraph

import (
	"strings"
	"testing"
	"unicode/utf8"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
)

func TestStringFromPprofStringTableTreatsIndexZeroAsUnset(t *testing.T) {
	table := []string{"", "main", "handler"}

	assert.Equal(t, "main", stringFromPprofStringTable(table, 1))
	assert.Equal(t, "handler", stringFromPprofStringTable(table, 2))
	assert.Equal(t, "", stringFromPprofStringTable(table, 0))
	assert.Equal(t, "", stringFromPprofStringTable(table, -1))
	assert.Equal(t, "", stringFromPprofStringTable(table, 3))
	assert.Equal(t, "", stringFromPprofStringTable(nil, 0))
}

func TestStringFromTableReturnsIndexZero(t *testing.T) {
	table := []string{"zero", "one"}

	assert.Equal(t, "zero", stringFromTable(table, 0))
	assert.Equal(t, "one", stringFromTable(table, 1))
	assert.Equal(t, "", stringFromTable(table, -1))
	assert.Equal(t, "", stringFromTable(table, 2))
	assert.Equal(t, "", stringFromTable(nil, 0))
}

// The two string-table lookups differ only at index 0 and are not interchangeable: the pprof
// variant treats 0 as "unset" (pprof reserves it for the empty string) while the plain one
// returns the entry. Swapping them would silently blank or resurrect frame names.
func TestTheTwoStringTableLookupsDisagreeAtIndexZero(t *testing.T) {
	table := []string{"reserved", "main"}

	assert.Equal(t, "", stringFromPprofStringTable(table, 0))
	assert.Equal(t, "reserved", stringFromTable(table, 0))
}

func TestTruncateFrameNameLeavesShortNamesUntouched(t *testing.T) {
	assert.Equal(t, "main.handler", truncateFrameName("main.handler"))
	assert.Equal(t, "", truncateFrameName(""))
}

// The cap is part of the response contract the UI renders against, so it is pinned to a literal
// rather than to the constant under test.
func TestTruncateFrameNameCapsAtTwoHundredAndFiftySixRunes(t *testing.T) {
	assert.Equal(t, 256, utf8.RuneCountInString(truncateFrameName(strings.Repeat("a", 256))))
	assert.Equal(t, strings.Repeat("a", 256)+"…", truncateFrameName(strings.Repeat("a", 257)))
}

func TestTruncateFrameNameKeepsANameOfExactlyTheLimit(t *testing.T) {
	name := strings.Repeat("a", maxFrameNameLen)

	got := truncateFrameName(name)

	assert.Equal(t, name, got)
	assert.NotContains(t, got, "…")
}

func TestTruncateFrameNameAppendsAnEllipsisPastTheLimit(t *testing.T) {
	name := strings.Repeat("a", maxFrameNameLen+1)

	got := truncateFrameName(name)

	assert.Equal(t, strings.Repeat("a", maxFrameNameLen)+"…", got)
}

// The cap counts runes, not bytes, so a multibyte name is not cut in the middle of a rune.
func TestTruncateFrameNameCountsRunesNotBytes(t *testing.T) {
	name := strings.Repeat("é", maxFrameNameLen+10)

	got := truncateFrameName(name)

	assert.Equal(t, maxFrameNameLen+1, utf8.RuneCountInString(got))
	assert.True(t, utf8.ValidString(got))
	assert.Equal(t, strings.Repeat("é", maxFrameNameLen)+"…", got)
}

func TestFunctionLineLabelPrefersTheFunctionName(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{
		Name:       f.intern("main.handler"),
		SystemName: f.intern("main.handler.systemName"),
		Filename:   f.intern("handler.go"),
		StartLine:  7,
	}

	assert.Equal(t, "main.handler", functionLineLabel(f.build(), fn, 42))
}

func TestFunctionLineLabelFallsBackToTheSystemName(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{
		SystemName: f.intern("main.handler.systemName"),
		Filename:   f.intern("handler.go"),
		StartLine:  7,
	}

	assert.Equal(t, "main.handler.systemName", functionLineLabel(f.build(), fn, 42))
}

func TestFunctionLineLabelFallsBackToTheSourceLine(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{
		Filename:  f.intern("handler.go"),
		StartLine: 7,
	}

	assert.Equal(t, "handler.go:42", functionLineLabel(f.build(), fn, 42))
}

func TestFunctionLineLabelFallsBackToTheFunctionStartLineWhenThereIsNoSourceLine(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{
		Filename:  f.intern("handler.go"),
		StartLine: 7,
	}

	assert.Equal(t, "handler.go:7", functionLineLabel(f.build(), fn, 0))
}

func TestFunctionLineLabelFallsBackToTheBareFilename(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{Filename: f.intern("handler.go")}

	assert.Equal(t, "handler.go", functionLineLabel(f.build(), fn, 0))
}

func TestFunctionLineLabelIsEmptyWithNothingToNameIt(t *testing.T) {
	f := newPprofFixture()

	assert.Equal(t, "", functionLineLabel(f.build(), &googleProfile.Function{}, 42))
	assert.Equal(t, "", functionLineLabel(f.build(), nil, 42))
}

func TestFunctionLineLabelTruncatesLongNames(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{Name: f.intern(strings.Repeat("n", maxFrameNameLen+5))}

	assert.Equal(t, strings.Repeat("n", maxFrameNameLen)+"…", functionLineLabel(f.build(), fn, 0))
}

func TestLineFrameLabelResolvesTheFunctionById(t *testing.T) {
	f := newPprofFixture().withFunction(7, "main.handler")
	p := f.build()
	funcByID := map[uint64]*googleProfile.Function{7: p.Function[0]}

	assert.Equal(t, "main.handler", lineFrameLabel(p, &googleProfile.Line{FunctionId: 7, Line: 3}, funcByID))
	assert.Equal(t, "", lineFrameLabel(p, &googleProfile.Line{FunctionId: 999}, funcByID))
	assert.Equal(t, "", lineFrameLabel(p, nil, funcByID))
}

func TestLineFrameLabelPassesTheLineNumberThroughToTheLabel(t *testing.T) {
	f := newPprofFixture()
	fn := &googleProfile.Function{Id: 7, Filename: f.intern("handler.go")}
	f.profile.Function = append(f.profile.Function, fn)
	p := f.build()

	assert.Equal(t, "handler.go:99", lineFrameLabel(p, &googleProfile.Line{FunctionId: 7, Line: 99}, map[uint64]*googleProfile.Function{7: fn}))
}

func TestLocationFallbackLabelUsesTheMappingFileBuildIdAndAddress(t *testing.T) {
	f := newPprofFixture().withMapping(1, "/usr/bin/app", "abc123")
	p := f.build()
	mappingByID := map[uint64]*googleProfile.Mapping{1: p.Mapping[0]}
	loc := &googleProfile.Location{Id: 5, MappingId: 1, Address: 255}

	assert.Equal(t, "/usr/bin/app [abc123]+0xff", locationFallbackLabel(p, loc, mappingByID))
}

func TestLocationFallbackLabelOmitsTheBuildIdWhenItIsUnknown(t *testing.T) {
	f := newPprofFixture().withMapping(1, "/usr/bin/app", "")
	p := f.build()
	mappingByID := map[uint64]*googleProfile.Mapping{1: p.Mapping[0]}
	loc := &googleProfile.Location{Id: 5, MappingId: 1, Address: 255}

	assert.Equal(t, "/usr/bin/app+0xff", locationFallbackLabel(p, loc, mappingByID))
}

func TestLocationFallbackLabelUsesTheBareFilenameWithoutAnAddress(t *testing.T) {
	f := newPprofFixture().withMapping(1, "/usr/bin/app", "abc123")
	p := f.build()
	mappingByID := map[uint64]*googleProfile.Mapping{1: p.Mapping[0]}
	loc := &googleProfile.Location{Id: 5, MappingId: 1}

	assert.Equal(t, "/usr/bin/app", locationFallbackLabel(p, loc, mappingByID))
}

func TestLocationFallbackLabelUsesTheBuildIdWhenThereIsNoMappingFilename(t *testing.T) {
	f := newPprofFixture().withMapping(1, "", "abc123")
	p := f.build()
	mappingByID := map[uint64]*googleProfile.Mapping{1: p.Mapping[0]}
	loc := &googleProfile.Location{Id: 5, MappingId: 1, Address: 16}

	assert.Equal(t, "abc123+0x10", locationFallbackLabel(p, loc, mappingByID))
}

func TestLocationFallbackLabelUsesTheRawAddressWithoutAMapping(t *testing.T) {
	p := newPprofFixture().build()
	loc := &googleProfile.Location{Id: 5, Address: 4096}

	assert.Equal(t, "0x1000", locationFallbackLabel(p, loc, nil))
}

func TestLocationFallbackLabelUsesTheLocationIdAsALastResort(t *testing.T) {
	p := newPprofFixture().build()

	assert.Equal(t, "frame_5", locationFallbackLabel(p, &googleProfile.Location{Id: 5}, nil))
}

func TestLocationFallbackLabelHandlesANilLocation(t *testing.T) {
	assert.Equal(t, "[unknown frame]", locationFallbackLabel(newPprofFixture().build(), nil, nil))
}

// pprof stores inlined Line entries innermost-first. locationFrameLabels must reverse them so a
// root-first stack keeps the natural caller-to-callee order.
func TestLocationFrameLabelsReturnInlinedFramesCallerFirst(t *testing.T) {
	f := newPprofFixture().
		withFunction(1, "inner").
		withFunction(2, "middle").
		withFunction(3, "outer").
		withLocation(9, 1, 2, 3)
	p := f.build()
	funcByID := map[uint64]*googleProfile.Function{1: p.Function[0], 2: p.Function[1], 3: p.Function[2]}

	labels := locationFrameLabels(p, p.Location[0], funcByID, nil)

	assert.Equal(t, []string{"outer", "middle", "inner"}, labels)
}

func TestLocationFrameLabelsSkipLinesThatCannotBeNamed(t *testing.T) {
	f := newPprofFixture().
		withFunction(1, "inner").
		withLocation(9, 1, 404)
	p := f.build()
	funcByID := map[uint64]*googleProfile.Function{1: p.Function[0]}

	assert.Equal(t, []string{"inner"}, locationFrameLabels(p, p.Location[0], funcByID, nil))
}

func TestLocationFrameLabelsFallBackWhenNoLineCanBeNamed(t *testing.T) {
	f := newPprofFixture().withLocation(9, 404)
	p := f.build()
	p.Location[0].Address = 32

	assert.Equal(t, []string{"0x20"}, locationFrameLabels(p, p.Location[0], nil, nil))
}

func TestLocationFrameLabelsFallBackWhenThereAreNoLinesAtAll(t *testing.T) {
	f := newPprofFixture().withLocation(9)
	p := f.build()

	assert.Equal(t, []string{"frame_9"}, locationFrameLabels(p, p.Location[0], nil, nil))
}

func TestLocationFrameLabelsHandleANilLocation(t *testing.T) {
	assert.Equal(t, []string{"[unknown frame]"}, locationFrameLabels(newPprofFixture().build(), nil, nil, nil))
}

func TestGoogleProfileToSamplesBuildsRootFirstStacks(t *testing.T) {
	f := newPprofFixture().
		withFunction(1, "main").
		withFunction(2, "handler").
		withFunction(3, "read").
		withLocation(10, 1).
		withLocation(20, 2).
		withLocation(30, 3).
		withSample(5, 30, 20, 10)

	samples := googleProfileToSamples(f.build())

	require.Len(t, samples, 1)
	assert.Equal(t, []string{"main", "handler", "read"}, samples[0].Stack)
	assert.Equal(t, int64(5), samples[0].Value)
}

func TestGoogleProfileToSamplesExpandsInlinedFramesInStackOrder(t *testing.T) {
	f := newPprofFixture().
		withFunction(1, "main").
		withFunction(2, "inlinedInner").
		withFunction(3, "inlinedOuter").
		withLocation(10, 1).
		withLocation(20, 2, 3).
		withSample(5, 20, 10)

	samples := googleProfileToSamples(f.build())

	require.Len(t, samples, 1)
	assert.Equal(t, []string{"main", "inlinedOuter", "inlinedInner"}, samples[0].Stack)
}

func TestGoogleProfileToSamplesUsesOnlyTheFirstValueColumn(t *testing.T) {
	f := newPprofFixture().withFunction(1, "main").withLocation(10, 1)
	p := f.build()
	p.Sample = append(p.Sample, &googleProfile.Sample{LocationId: []uint64{10}, Value: []int64{3, 1000}})

	samples := googleProfileToSamples(p)

	require.Len(t, samples, 1)
	assert.Equal(t, int64(3), samples[0].Value)
}

func TestGoogleProfileToSamplesSkipsNonPositiveAndValuelessSamples(t *testing.T) {
	f := newPprofFixture().withFunction(1, "main").withLocation(10, 1)
	p := f.build()
	p.Sample = append(p.Sample,
		&googleProfile.Sample{LocationId: []uint64{10}, Value: []int64{0, 1000}},
		&googleProfile.Sample{LocationId: []uint64{10}, Value: []int64{-4}},
		&googleProfile.Sample{LocationId: []uint64{10}},
		&googleProfile.Sample{LocationId: []uint64{10}, Value: []int64{8}},
	)

	samples := googleProfileToSamples(p)

	require.Len(t, samples, 1)
	assert.Equal(t, int64(8), samples[0].Value)
}

// Dropping samples that reference a location the merge step removed would punch holes in the
// flame graph, so they keep their weight behind a placeholder frame instead.
func TestGoogleProfileToSamplesKeepsSamplesWithAMissingLocation(t *testing.T) {
	f := newPprofFixture().
		withFunction(1, "main").
		withLocation(10, 1).
		withSample(5, 77, 10)

	samples := googleProfileToSamples(f.build())

	require.Len(t, samples, 1)
	assert.Equal(t, []string{"main", "[missing location id=77]"}, samples[0].Stack)
}

// A sample with weight but no locations has nowhere to go in the flame graph; emitting it would
// add a frameless row to the tree.
func TestGoogleProfileToSamplesDropsASampleWithNoLocations(t *testing.T) {
	p := newPprofFixture().withFunction(1, "main").withLocation(10, 1).build()
	p.Sample = append(p.Sample,
		&googleProfile.Sample{Value: []int64{5}},
		&googleProfile.Sample{LocationId: []uint64{10}, Value: []int64{8}},
	)

	samples := googleProfileToSamples(p)

	require.Len(t, samples, 1)
	assert.Equal(t, []string{"main"}, samples[0].Stack)
	assert.Equal(t, int64(8), samples[0].Value)
}

func TestGoogleProfileToSamplesReturnsNothingForAnEmptyProfile(t *testing.T) {
	assert.Nil(t, googleProfileToSamples(nil))
	assert.Nil(t, googleProfileToSamples(newPprofFixture().build()))
}

func TestProfileTypeFromGoogleProfileReadsTheDefaultSampleType(t *testing.T) {
	f := newPprofFixture().withSampleType("samples", "count").withSampleType("cpu", "nanoseconds")
	p := f.build()
	p.DefaultSampleType = 1

	pt := profileTypeFromGoogleProfile(p)

	assert.Equal(t, "cpu", pt.SampleType)
	assert.Equal(t, "nanoseconds", pt.SampleUnit)
}

func TestProfileTypeFromGoogleProfileDefaultsToTheFirstSampleType(t *testing.T) {
	f := newPprofFixture().withSampleType("samples", "count").withSampleType("cpu", "nanoseconds")

	pt := profileTypeFromGoogleProfile(f.build())

	assert.Equal(t, "samples", pt.SampleType)
	assert.Equal(t, "count", pt.SampleUnit)
}

func TestProfileTypeFromGoogleProfileClampsAnOutOfRangeDefaultSampleType(t *testing.T) {
	for _, idx := range []int64{-1, 2, 100} {
		f := newPprofFixture().withSampleType("samples", "count").withSampleType("cpu", "nanoseconds")
		p := f.build()
		p.DefaultSampleType = idx

		assert.Equal(t, "samples", profileTypeFromGoogleProfile(p).SampleType, "index %d", idx)
	}
}

func TestProfileTypeFromGoogleProfileFallsBackToTheDefaultType(t *testing.T) {
	assert.Equal(t, DefaultProfileType(), profileTypeFromGoogleProfile(nil))
	assert.Equal(t, DefaultProfileType(), profileTypeFromGoogleProfile(newPprofFixture().build()))

	withNilEntry := newPprofFixture().build()
	withNilEntry.SampleType = []*googleProfile.ValueType{nil}
	assert.Equal(t, DefaultProfileType(), profileTypeFromGoogleProfile(withNilEntry))

	// A sample type whose name resolves to the empty string carries no information.
	blank := newPprofFixture().withSampleType("", "count").build()
	assert.Equal(t, DefaultProfileType(), profileTypeFromGoogleProfile(blank))
}

func TestDefaultProfileTypeIsCpu(t *testing.T) {
	assert.Equal(t, "cpu", DefaultProfileType().SampleType)
}

// Some eBPF profilers report only timestamps; Pyroscope's converter needs at least one numeric
// value per sample or the sample is dropped, so one aggregate weight is synthesized.
func TestNormalizeSampleValuesSynthesizesAWeightFromTimestamps(t *testing.T) {
	p := &otelProfile.Profile{Samples: []*otelProfile.Sample{
		{TimestampsUnixNano: []uint64{1, 2, 3}},
	}}

	normalizeSampleValuesForPyroscope(p)

	assert.Equal(t, []int64{3}, p.Samples[0].Values)
}

func TestNormalizeSampleValuesKeepsOnlyTheFirstValueColumn(t *testing.T) {
	p := &otelProfile.Profile{Samples: []*otelProfile.Sample{
		{Values: []int64{7, 11, 13}},
	}}

	normalizeSampleValuesForPyroscope(p)

	assert.Equal(t, []int64{7}, p.Samples[0].Values)
}

func TestNormalizeSampleValuesLeavesScalarSamplesAlone(t *testing.T) {
	p := &otelProfile.Profile{Samples: []*otelProfile.Sample{
		{Values: []int64{7}, TimestampsUnixNano: []uint64{1, 2, 3}},
	}}

	normalizeSampleValuesForPyroscope(p)

	assert.Equal(t, []int64{7}, p.Samples[0].Values)
}

func TestNormalizeSampleValuesLeavesAValuelessUntimedSampleEmpty(t *testing.T) {
	p := &otelProfile.Profile{Samples: []*otelProfile.Sample{{}}}

	normalizeSampleValuesForPyroscope(p)

	assert.Empty(t, p.Samples[0].Values)
}

func TestNormalizeSampleValuesToleratesNilInput(t *testing.T) {
	assert.NotPanics(t, func() { normalizeSampleValuesForPyroscope(nil) })
	assert.NotPanics(t, func() {
		normalizeSampleValuesForPyroscope(&otelProfile.Profile{Samples: []*otelProfile.Sample{nil}})
	})
}
