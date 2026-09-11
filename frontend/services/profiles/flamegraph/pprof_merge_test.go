package flamegraph

import (
	"testing"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileCompatibilityKeyIgnoresEverythingButTheSchema(t *testing.T) {
	base := singleFrameProfile("main", 5)
	// Same schema, completely different samples and symbols.
	other := newPprofFixture().
		withPeriodType("cpu", "nanoseconds").
		withSampleType("samples", "count").
		withFunction(1, "somethingElse").
		withLocation(1, 1).
		withSample(99, 1).
		withSample(1, 1).
		build()

	assert.Equal(t, profileCompatibilityKey(base), profileCompatibilityKey(other))
}

// Each schema field is changed one at a time: a key that ignores any of them would merge
// profiles pprof considers incompatible, and a key that is too eager would stop merging
// profiles that belong together.
func TestProfileCompatibilityKeyChangesWithEverySchemaField(t *testing.T) {
	baseKey := profileCompatibilityKey(singleFrameProfile("main", 5))

	variants := map[string]func(p *googleProfile.Profile){
		"period type name": func(p *googleProfile.Profile) {
			p.PeriodType.Type = int64(len(p.StringTable))
			p.StringTable = append(p.StringTable, "wall")
		},
		"period type unit": func(p *googleProfile.Profile) {
			p.PeriodType.Unit = int64(len(p.StringTable))
			p.StringTable = append(p.StringTable, "microseconds")
		},
		"no period type": func(p *googleProfile.Profile) {
			p.PeriodType = nil
		},
		"sample type name": func(p *googleProfile.Profile) {
			p.SampleType[0].Type = int64(len(p.StringTable))
			p.StringTable = append(p.StringTable, "alloc_space")
		},
		"sample type unit": func(p *googleProfile.Profile) {
			p.SampleType[0].Unit = int64(len(p.StringTable))
			p.StringTable = append(p.StringTable, "bytes")
		},
		"extra sample type": func(p *googleProfile.Profile) {
			p.SampleType = append(p.SampleType, &googleProfile.ValueType{Type: 1, Unit: 2})
		},
		"nil sample type": func(p *googleProfile.Profile) {
			p.SampleType = append(p.SampleType, nil)
		},
		"default sample type": func(p *googleProfile.Profile) {
			p.DefaultSampleType = 1
		},
	}

	for name, mutate := range variants {
		t.Run(name, func(t *testing.T) {
			p := singleFrameProfile("main", 5)
			mutate(p)
			assert.NotEqual(t, baseKey, profileCompatibilityKey(p))
		})
	}
}

func TestProfileCompatibilityKeyOfANilProfileIsEmpty(t *testing.T) {
	assert.Equal(t, "", profileCompatibilityKey(nil))
}

func TestMergeGoogleProfilesGroupedMergesProfilesSharingASchema(t *testing.T) {
	merged, extra := mergeGoogleProfilesGrouped([]*googleProfile.Profile{
		singleFrameProfile("main", 3),
		singleFrameProfile("main", 4),
	})

	assert.Empty(t, extra)
	require.Len(t, merged, 1)
	for _, mp := range merged {
		assert.Equal(t, int64(7), profileTotalWeight(mp))
	}
}

func TestMergeGoogleProfilesGroupedKeepsIncompatibleSchemasApart(t *testing.T) {
	cpu := singleFrameProfile("main", 3)
	alloc := newPprofFixture().
		withPeriodType("space", "bytes").
		withSampleType("alloc_space", "bytes").
		withFunction(1, "main").
		withLocation(1, 1).
		withSample(4, 1).
		build()

	merged, extra := mergeGoogleProfilesGrouped([]*googleProfile.Profile{cpu, alloc})

	assert.Empty(t, extra)
	assert.Len(t, merged, 2)
}

func TestMergeGoogleProfilesGroupedSkipsNilProfiles(t *testing.T) {
	merged, extra := mergeGoogleProfilesGrouped([]*googleProfile.Profile{nil, singleFrameProfile("main", 3), nil})

	assert.Empty(t, extra)
	assert.Len(t, merged, 1)
}

func TestMergeGoogleProfilesGroupedOnNoInputReturnsNothing(t *testing.T) {
	merged, extra := mergeGoogleProfilesGrouped(nil)

	assert.Empty(t, merged)
	assert.Empty(t, extra)
}

func TestMergeGoogleProfilesGroupedDropsBucketsThatMergeToNoSamples(t *testing.T) {
	// pprof skips profiles with a string table too small to rewrite, so the bucket merges to nothing.
	unmergeable := &googleProfile.Profile{
		StringTable: []string{""},
		PeriodType:  &googleProfile.ValueType{},
		SampleType:  []*googleProfile.ValueType{{}},
		Sample:      []*googleProfile.Sample{{LocationId: []uint64{1}, Value: []int64{5}}},
	}

	merged, extra := mergeGoogleProfilesGrouped([]*googleProfile.Profile{unmergeable})

	assert.Empty(t, merged)
	assert.Empty(t, extra)
}

// A profile with no period type shares a bucket key with its peers but pprof still refuses to
// merge it. Rather than lose the weight, the bucket is expanded into raw samples.
func TestMergeGoogleProfilesGroupedFallsBackToRawSamplesWhenABucketCannotMerge(t *testing.T) {
	noPeriodType := newPprofFixture().
		withSampleType("samples", "count").
		withFunction(1, "main").
		withLocation(1, 1).
		withSample(5, 1).
		build()
	require.Nil(t, noPeriodType.PeriodType)

	merged, extra := mergeGoogleProfilesGrouped([]*googleProfile.Profile{noPeriodType})

	assert.Empty(t, merged)
	require.Len(t, extra, 1)
	assert.Equal(t, []string{"main"}, extra[0].Stack)
	assert.Equal(t, int64(5), extra[0].Value)
}

func TestMergeGoogleProfilesGroupedDoesNotMutateItsInput(t *testing.T) {
	input := singleFrameProfile("main", 3)
	before := input.String()

	mergeGoogleProfilesGrouped([]*googleProfile.Profile{input, singleFrameProfile("main", 4)})

	assert.Equal(t, before, input.String())
}

func TestSortedKeysAreSorted(t *testing.T) {
	keys := sortedKeys(map[string]*googleProfile.Profile{"zulu": nil, "alpha": nil, "mike": nil})

	assert.Equal(t, []string{"alpha", "mike", "zulu"}, keys)
}

func TestSortedKeysOfAnEmptyMapIsEmpty(t *testing.T) {
	assert.Empty(t, sortedKeys(nil))
}

func TestProfileTotalWeightSumsTheFirstValueColumn(t *testing.T) {
	p := newPprofFixture().build()
	p.Sample = append(p.Sample,
		&googleProfile.Sample{Value: []int64{3, 1000}},
		&googleProfile.Sample{Value: []int64{4}},
		&googleProfile.Sample{Value: []int64{}},
		nil,
	)

	assert.Equal(t, int64(7), profileTotalWeight(p))
}

func TestProfileTotalWeightOfANilProfileIsZero(t *testing.T) {
	assert.Equal(t, int64(0), profileTotalWeight(nil))
	assert.Equal(t, int64(0), profileTotalWeight(newPprofFixture().build()))
}
