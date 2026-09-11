package flamegraph

import (
	"context"
	"path/filepath"
	"sort"
	"strconv"
	"testing"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	phlaremodel "github.com/grafana/pyroscope/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/proto"
)

func TestParseExportProfilesServiceRequestDecodesAChunk(t *testing.T) {
	chunk := allocChunkWithStacks(t, map[int64][]string{5: {"main", "handler"}})

	req, err := ParseExportProfilesServiceRequest(chunk)

	require.NoError(t, err)
	require.Len(t, req.ResourceProfiles, 1)
	require.NotNil(t, req.Dictionary)
}

func TestParseExportProfilesServiceRequestRejectsAChunkWithoutResourceProfiles(t *testing.T) {
	empty, err := proto.Marshal(&otelProfile.ResourceProfiles{})
	require.NoError(t, err)

	for name, chunk := range map[string][]byte{
		"empty":            {},
		"no resources":     empty,
		"not proto at all": []byte("this is not a protobuf message"),
	} {
		t.Run(name, func(t *testing.T) {
			req, err := ParseExportProfilesServiceRequest(chunk)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "no resource profiles found")
			assert.Nil(t, req)
		})
	}
}

func TestGoogleProfilesFromParsedRequestConvertsEveryProfile(t *testing.T) {
	f := newOtlpFixture()
	first := f.allocProfile()
	first.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main", "handler"), Values: []int64{5}}}
	second := f.allocProfile()
	second.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main", "write"), Values: []int64{7}}}

	profiles := googleProfilesFromParsedRequest(f.request(first, second))

	require.Len(t, profiles, 2)
	assert.Equal(t, []string{"main", "handler"}, googleProfileToSamples(profiles[0])[0].Stack)
	assert.Equal(t, int64(5), googleProfileToSamples(profiles[0])[0].Value)
	assert.Equal(t, []string{"main", "write"}, googleProfileToSamples(profiles[1])[0].Stack)
	assert.Equal(t, int64(7), googleProfileToSamples(profiles[1])[0].Value)
}

// The dictionary is mandatory for symbol resolution; a request that arrives without one gets an
// empty dictionary rather than crashing the converter.
func TestGoogleProfilesFromParsedRequestSubstitutesAMissingDictionary(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main"), Values: []int64{5}}}
	req := f.request(prof)
	req.Dictionary = nil

	profiles := googleProfilesFromParsedRequest(req)

	require.NotNil(t, req.Dictionary)
	assert.Empty(t, profiles)
}

func TestGoogleProfilesFromParsedRequestSkipsEmptyNesting(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main"), Values: []int64{5}}}

	req := f.request(prof)
	req.ResourceProfiles = append(req.ResourceProfiles,
		nil,
		&otelProfile.ResourceProfiles{},
		&otelProfile.ResourceProfiles{ScopeProfiles: []*otelProfile.ScopeProfiles{nil, {}}},
		&otelProfile.ResourceProfiles{ScopeProfiles: []*otelProfile.ScopeProfiles{{Profiles: []*otelProfile.Profile{nil}}}},
	)

	profiles := googleProfilesFromParsedRequest(req)

	require.Len(t, profiles, 1)
}

func TestGoogleProfilesFromParsedRequestDropsProfilesThatConvertToNoSamples(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{{StackIndex: 999, Values: []int64{5}}}

	assert.Empty(t, googleProfilesFromParsedRequest(f.request(prof)))
}

func TestGoogleProfilesFromParsedRequestToleratesANilRequest(t *testing.T) {
	assert.Nil(t, googleProfilesFromParsedRequest(nil))
}

func TestGoogleProfilesFromParsedRequestKeepsOnlyTheFirstValueColumn(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{
		// Only the second column carries weight, and the first value column is what survives
		// normalization, so this sample contributes nothing.
		{StackIndex: f.withStack("main", "handler"), Values: []int64{0, 5}},
		{StackIndex: f.withStack("main", "write"), Values: []int64{3}},
	}

	profiles := googleProfilesFromParsedRequest(f.request(prof))

	require.Len(t, profiles, 1)
	samples := googleProfileToSamples(profiles[0])
	require.Len(t, samples, 1)
	assert.Equal(t, []string{"main", "write"}, samples[0].Stack)
}

// A profile carrying only timestamps still has to produce weight, otherwise eBPF profiles would
// render as an empty flame graph.
func TestGoogleProfilesFromParsedRequestKeepsTimestampOnlySamples(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{{
		StackIndex:         f.withStack("main", "handler"),
		TimestampsUnixNano: []uint64{1, 2, 3, 4},
	}}

	profiles := googleProfilesFromParsedRequest(f.request(prof))

	require.Len(t, profiles, 1)
	samples := googleProfileToSamples(profiles[0])
	require.Len(t, samples, 1)
	assert.Equal(t, int64(4), samples[0].Value)
}

func TestCollectGoogleProfilesFromChunksSkipsEmptyAndUndecodableChunks(t *testing.T) {
	good := allocChunkWithStacks(t, map[int64][]string{5: {"main"}})

	profiles := collectGoogleProfilesFromChunks([][]byte{nil, {}, []byte("garbage"), good})

	require.Len(t, profiles, 1)
}

func TestCollectGoogleProfilesFromChunksReadsEveryChunk(t *testing.T) {
	profiles := collectGoogleProfilesFromChunks([][]byte{
		allocChunkWithStacks(t, map[int64][]string{5: {"main"}}),
		allocChunkWithStacks(t, map[int64][]string{7: {"other"}}),
	})

	assert.Len(t, profiles, 2)
}

func TestMergedGoogleProfileForPyroscopeSymdbMergesChunksOfTheSameType(t *testing.T) {
	merged, profileType, extra := MergedGoogleProfileForPyroscopeSymdb([][]byte{
		allocChunkWithStacks(t, map[int64][]string{5: {"main", "handler"}}),
		allocChunkWithStacks(t, map[int64][]string{7: {"main", "handler"}}),
	})

	require.NotNil(t, merged)
	assert.Empty(t, extra)
	assert.Equal(t, "alloc_space", profileType.SampleType)
	assert.Equal(t, "bytes", profileType.SampleUnit)
	assert.Equal(t, int64(12), profileTotalWeight(merged))
}

func TestMergedGoogleProfileForPyroscopeSymdbWithoutChunksReturnsTheDefaultType(t *testing.T) {
	merged, profileType, extra := MergedGoogleProfileForPyroscopeSymdb(nil)

	assert.Nil(t, merged)
	assert.Empty(t, extra)
	assert.Equal(t, DefaultProfileType(), profileType)
}

// Two profile types cannot share one pprof profile. The heaviest bucket becomes the merged
// profile and the rest are handed back as raw samples so their weight still reaches the tree.
func TestMergedGoogleProfileForPyroscopeSymdbPicksTheHeaviestBucketAndKeepsTheRest(t *testing.T) {
	light := newOtlpFixture()
	lightProfile := light.allocProfile()
	lightProfile.Samples = []*otelProfile.Sample{{StackIndex: light.withStack("lightFrame"), Values: []int64{3}}}

	heavy := newOtlpFixture()
	heavyProfile := heavy.allocProfile()
	heavyProfile.SampleType = &otelProfile.ValueType{
		TypeStrindex: heavy.intern("inuse_space"),
		UnitStrindex: heavy.intern("bytes"),
	}
	heavyProfile.Samples = []*otelProfile.Sample{{StackIndex: heavy.withStack("heavyFrame"), Values: []int64{100}}}

	merged, _, extra := MergedGoogleProfileForPyroscopeSymdb([][]byte{
		light.chunk(t, lightProfile),
		heavy.chunk(t, heavyProfile),
	})

	require.NotNil(t, merged)
	assert.Equal(t, int64(100), profileTotalWeight(merged))
	require.Len(t, extra, 1)
	assert.Equal(t, []string{"lightFrame"}, extra[0].Stack)
	assert.Equal(t, int64(3), extra[0].Value)
}

func TestBuildFlamebearerViaPyroscopeSymdbRendersAFlameGraph(t *testing.T) {
	chunk := allocChunkWithStacks(t, map[int64][]string{
		5: {"main", "handler", "read"},
		7: {"main", "handler", "write"},
	})

	fb, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, 2048)

	require.NoError(t, err)
	require.NotNil(t, fb)
	require.NotNil(t, tree)
	assert.Equal(t, 12, fb.Flamebearer.NumTicks)
	assert.Subset(t, fb.Flamebearer.Names, []string{"main", "handler", "read", "write"})
	assert.Equal(t, "alloc_space", fb.Metadata.Name)
}

func TestBuildFlamebearerViaPyroscopeSymdbWithoutChunksReturnsNothing(t *testing.T) {
	fb, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), nil, 2048)

	require.NoError(t, err)
	assert.Nil(t, fb)
	assert.Nil(t, tree)
}

// Buckets that could not be merged into a pprof profile still have to render, using the raw
// sample stacks instead of the symbol database.
func TestBuildFlamebearerViaPyroscopeSymdbRendersUnmergeableSamplesOnTheirOwn(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.PeriodType = nil
	prof.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main", "orphan"), Values: []int64{9}}}

	fb, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{f.chunk(t, prof)}, 2048)

	require.NoError(t, err)
	require.NotNil(t, fb)
	require.NotNil(t, tree)
	assert.Equal(t, 9, fb.Flamebearer.NumTicks)
	assert.Subset(t, fb.Flamebearer.Names, []string{"main", "orphan"})
}

// A flame graph built from a mix of mergeable and unmergeable buckets has to include both, or
// the lighter profile types silently vanish from the UI.
func TestBuildFlamebearerViaPyroscopeSymdbCombinesMergedAndLeftoverSamples(t *testing.T) {
	heavy := newOtlpFixture()
	heavyProfile := heavy.allocProfile()
	heavyProfile.Samples = []*otelProfile.Sample{{StackIndex: heavy.withStack("heavyFrame"), Values: []int64{100}}}

	light := newOtlpFixture()
	lightProfile := light.allocProfile()
	lightProfile.SampleType = &otelProfile.ValueType{
		TypeStrindex: light.intern("inuse_space"),
		UnitStrindex: light.intern("bytes"),
	}
	lightProfile.Samples = []*otelProfile.Sample{{StackIndex: light.withStack("lightFrame"), Values: []int64{3}}}

	fb, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{
		heavy.chunk(t, heavyProfile),
		light.chunk(t, lightProfile),
	}, 2048)

	require.NoError(t, err)
	require.NotNil(t, fb)
	require.NotNil(t, tree)
	assert.Equal(t, 103, fb.Flamebearer.NumTicks)
	assert.Subset(t, fb.Flamebearer.Names, []string{"heavyFrame", "lightFrame"})
}

func TestBuildFlamebearerViaPyroscopeSymdbFallsBackToADefaultNodeBudget(t *testing.T) {
	chunk := allocChunkWithStacks(t, map[int64][]string{5: {"main", "handler"}})

	for _, maxNodes := range []int64{0, -1} {
		fb, _, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, maxNodes)

		require.NoError(t, err)
		require.NotNil(t, fb, "maxNodes %d", maxNodes)
		assert.Equal(t, 5, fb.Flamebearer.NumTicks)
	}
}

// Pyroscope treats a non-positive node budget as "no limit", so without the default the UI would
// be sent an unbounded flame graph for a wide profile.
func TestBuildFlamebearerViaPyroscopeSymdbBoundsAWideProfileWithoutAnExplicitBudget(t *testing.T) {
	const frameCount = symdbFlameMaxNodesDefault + 200

	stacks := make(map[int64][]string, frameCount)
	for i := int64(0); i < frameCount; i++ {
		stacks[i+1] = []string{"main", "frame" + strconv.FormatInt(i, 10)}
	}
	chunk := allocChunkWithStacks(t, stacks)

	bounded, _, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, 0)
	require.NoError(t, err)
	require.NotNil(t, bounded)

	unbounded, _, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, frameCount*2)
	require.NoError(t, err)
	require.NotNil(t, unbounded)

	assert.Less(t, len(bounded.Flamebearer.Names), len(unbounded.Flamebearer.Names))
	assert.Contains(t, bounded.Flamebearer.Names, otherName)
}

// Leftovers from two different failure modes arrive on the same path: a bucket pprof refused to
// merge at all, and the lighter of two mutually incompatible buckets. Both have to survive.
func TestMergedGoogleProfileForPyroscopeSymdbKeepsLeftoversFromEveryBucket(t *testing.T) {
	heaviest := newOtlpFixture()
	heaviestProfile := heaviest.allocProfile()
	heaviestProfile.Samples = []*otelProfile.Sample{{StackIndex: heaviest.withStack("heaviestFrame"), Values: []int64{100}}}

	otherType := newOtlpFixture()
	otherTypeProfile := otherType.allocProfile()
	otherTypeProfile.SampleType = &otelProfile.ValueType{
		TypeStrindex: otherType.intern("inuse_space"),
		UnitStrindex: otherType.intern("bytes"),
	}
	otherTypeProfile.Samples = []*otelProfile.Sample{{StackIndex: otherType.withStack("otherTypeFrame"), Values: []int64{5}}}

	unmergeable := newOtlpFixture()
	unmergeableProfile := unmergeable.allocProfile()
	unmergeableProfile.PeriodType = nil
	unmergeableProfile.Samples = []*otelProfile.Sample{{StackIndex: unmergeable.withStack("unmergeableFrame"), Values: []int64{7}}}

	merged, _, extra := MergedGoogleProfileForPyroscopeSymdb([][]byte{
		heaviest.chunk(t, heaviestProfile),
		otherType.chunk(t, otherTypeProfile),
		unmergeable.chunk(t, unmergeableProfile),
	})

	require.NotNil(t, merged)
	assert.Equal(t, int64(100), profileTotalWeight(merged))

	leftoverFrames := make([]string, 0, len(extra))
	for _, s := range extra {
		leftoverFrames = append(leftoverFrames, s.Stack...)
	}
	assert.ElementsMatch(t, []string{"unmergeableFrame", "otherTypeFrame"}, leftoverFrames)
}

func TestBuildFlamebearerViaPyroscopeSymdbFailsWhenItCannotCreateATempDir(t *testing.T) {
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "missing-parent"))
	chunk := allocChunkWithStacks(t, map[int64][]string{5: {"main"}})

	fb, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, 2048)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "symdb temp dir")
	assert.Nil(t, fb)
	assert.Nil(t, tree)
}

func TestSymbolStatsFromFunctionNameTreeSkipsStacksWithNoNamedFrames(t *testing.T) {
	tree := new(phlaremodel.FunctionNameTree)
	insertSamplesIntoFunctionNameTree(tree, []Sample{
		{Stack: []string{""}, Value: 5},
		{Stack: []string{"named"}, Value: 3},
	})

	assert.Equal(t, []SymbolStats{{Name: "named", Self: 3, Total: 3}}, SymbolStatsFromFunctionNameTree(tree))
}

func TestInsertSamplesIntoFunctionNameTreeSkipsWeightlessAndEmptyStacks(t *testing.T) {
	tree := new(phlaremodel.FunctionNameTree)

	insertSamplesIntoFunctionNameTree(tree, []Sample{
		{Stack: []string{"kept"}, Value: 5},
		{Stack: []string{"zeroValue"}, Value: 0},
		{Stack: []string{"negativeValue"}, Value: -3},
		{Stack: nil, Value: 10},
	})

	assert.Equal(t, []SymbolStats{{Name: "kept", Self: 5, Total: 5}}, SymbolStatsFromFunctionNameTree(tree))
	// A weightless frame must not even reach the tree: it would still be rendered as a flame
	// graph bar despite contributing nothing.
	assert.Equal(t, []string{"kept"}, frameNamesInTree(tree))
}

// frameNamesInTree lists every frame the tree holds, regardless of its weight.
func frameNamesInTree(tree *phlaremodel.FunctionNameTree) []string {
	seen := map[string]struct{}{}
	tree.IterateStacks(func(_ phlaremodel.FunctionName, _ int64, stack []phlaremodel.FunctionName) {
		for _, name := range stack {
			seen[string(name)] = struct{}{}
		}
	})
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func TestSymbolStatsFromFunctionNameTreeAggregatesLeafWeights(t *testing.T) {
	tree := new(phlaremodel.FunctionNameTree)
	insertSamplesIntoFunctionNameTree(tree, []Sample{
		{Stack: []string{"main", "handler", "read"}, Value: 3},
		{Stack: []string{"main", "handler", "write"}, Value: 7},
	})

	stats := SymbolStatsFromFunctionNameTree(tree)
	byName := make(map[string]SymbolStats, len(stats))
	for _, s := range stats {
		byName[s.Name] = s
	}

	assert.Equal(t, SymbolStats{Name: "write", Self: 7, Total: 7}, byName["write"])
	assert.Equal(t, SymbolStats{Name: "read", Self: 3, Total: 3}, byName["read"])
	assert.Equal(t, SymbolStats{Name: "handler", Self: 0, Total: 10}, byName["handler"])
	assert.Equal(t, SymbolStats{Name: "main", Self: 0, Total: 10}, byName["main"])
}

func TestSymbolStatsFromFunctionNameTreeOnANilTreeReturnsNothing(t *testing.T) {
	assert.Nil(t, SymbolStatsFromFunctionNameTree(nil))
}

func TestSymbolStatsFromTheBuiltFlamebearerTreeMatchTheSampleWeights(t *testing.T) {
	chunk := allocChunkWithStacks(t, map[int64][]string{
		4: {"main", "handler", "read"},
		6: {"main", "handler", "write"},
	})

	_, tree, err := BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, 2048)
	require.NoError(t, err)

	stats := SymbolStatsFromFunctionNameTree(tree)
	byName := make(map[string]SymbolStats, len(stats))
	for _, s := range stats {
		byName[s.Name] = s
	}

	assert.Equal(t, int64(10), byName["main"].Total)
	assert.Equal(t, int64(4), byName["read"].Self)
	assert.Equal(t, int64(6), byName["write"].Self)
}

func TestExtractGoogleProfileReadsPyroscopesUnexportedProfileField(t *testing.T) {
	f := newOtlpFixture()
	prof := f.allocProfile()
	prof.Samples = []*otelProfile.Sample{{StackIndex: f.withStack("main"), Values: []int64{5}}}

	// googleProfilesFromParsedRequest is the only caller, so a successful conversion proves the
	// unsafe/reflect read of Pyroscope's unexported ConvertedProfile field still works.
	profiles := googleProfilesFromParsedRequest(f.request(prof))

	require.Len(t, profiles, 1)
	require.Len(t, profiles[0].Sample, 1)
}

func TestExtractGoogleProfileFallsBackToAJsonProfileField(t *testing.T) {
	holder := struct {
		Profile *googleProfile.Profile `json:"profile"`
	}{Profile: singleFrameProfile("main", 5)}

	got := extractGoogleProfile(holder)

	require.NotNil(t, got)
	assert.Len(t, got.Sample, 1)
}

func TestExtractGoogleProfileReadsAnExportedProfilePointerField(t *testing.T) {
	holder := struct {
		Profile *googleProfile.Profile
	}{Profile: singleFrameProfile("main", 5)}

	assert.Same(t, holder.Profile, extractGoogleProfile(holder))
}

func TestExtractGoogleProfileReadsAnExportedProfileValueField(t *testing.T) {
	built := singleFrameProfile("main", 5)
	holder := &struct {
		Profile googleProfile.Profile
	}{}
	holder.Profile.StringTable = built.StringTable
	holder.Profile.Sample = built.Sample

	got := extractGoogleProfile(holder)

	require.NotNil(t, got)
	assert.Len(t, got.Sample, 1)
}

func TestExtractGoogleProfileReturnsNilWhenThereIsNoProfileToFind(t *testing.T) {
	assert.Nil(t, extractGoogleProfile(nil))
	assert.Nil(t, extractGoogleProfile((*struct{})(nil)))
	assert.Nil(t, extractGoogleProfile("not a struct"))
	assert.Nil(t, extractGoogleProfile(struct{ Unrelated int }{Unrelated: 3}))
	// Neither the reflect scan nor the JSON fallback can make sense of these.
	assert.Nil(t, extractGoogleProfile(struct{ Unmarshalable func() }{}))
	assert.Nil(t, extractGoogleProfile(struct {
		Profile string `json:"profile"`
	}{Profile: "not a profile"}))
}

func TestExtractGoogleProfileDereferencesAPointerToAStruct(t *testing.T) {
	holder := &struct {
		Profile *googleProfile.Profile
	}{Profile: singleFrameProfile("main", 5)}

	assert.Same(t, holder.Profile, extractGoogleProfile(holder))
}
