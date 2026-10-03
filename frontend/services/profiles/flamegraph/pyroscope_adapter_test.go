package flamegraph

import (
	"testing"

	pyrofb "github.com/grafana/pyroscope/pkg/og/structs/flamebearer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdaptPyroscopeFlamebearerProfileAttachesTheTimelineAndSymbols(t *testing.T) {
	up := &pyrofb.FlamebearerProfile{
		Version: 1,
		FlamebearerProfileV1: pyrofb.FlamebearerProfileV1{
			Flamebearer: pyrofb.FlamebearerV1{Names: []string{"total", "main"}, NumTicks: 5},
		},
	}
	timeline := &pyrofb.FlamebearerTimelineV1{StartTime: 1700000000, Samples: []uint64{0, 5}}
	symbols := []SymbolStats{{Name: "main", Self: 5, Total: 5}}

	got := AdaptPyroscopeFlamebearerProfile(up, timeline, symbols)

	assert.Same(t, up, got.FlamebearerProfile)
	assert.Same(t, timeline, got.FlamebearerProfile.Timeline)
	assert.Equal(t, symbols, got.Symbols)
	assert.Equal(t, []string{"total", "main"}, got.Flamebearer.Names)
}

// The UI still needs a well-formed, empty flame graph when nothing could be built, otherwise the
// profiling screen has no shape to render.
func TestAdaptPyroscopeFlamebearerProfileSubstitutesAnEmptyProfile(t *testing.T) {
	got := AdaptPyroscopeFlamebearerProfile(nil, nil, nil)

	require.NotNil(t, got.FlamebearerProfile)
	assert.Equal(t, uint(1), got.Version)
	assert.Equal(t, []string{"total"}, got.Flamebearer.Names)
	assert.Equal(t, [][]int{}, got.Flamebearer.Levels)
	assert.Equal(t, 0, got.Flamebearer.NumTicks)
	assert.Equal(t, 0, got.Flamebearer.MaxSelf)
	assert.Nil(t, got.FlamebearerProfile.Timeline)
	assert.Nil(t, got.Symbols)
}

func TestAdaptPyroscopeFlamebearerProfileAttachesATimelineToTheSubstituteProfile(t *testing.T) {
	timeline := &pyrofb.FlamebearerTimelineV1{StartTime: 42}

	got := AdaptPyroscopeFlamebearerProfile(nil, timeline, nil)

	assert.Same(t, timeline, got.FlamebearerProfile.Timeline)
}

func TestAdaptPyroscopeFlamebearerProfileClearsAStaleTimeline(t *testing.T) {
	up := &pyrofb.FlamebearerProfile{}
	up.Timeline = &pyrofb.FlamebearerTimelineV1{StartTime: 1}

	got := AdaptPyroscopeFlamebearerProfile(up, nil, nil)

	assert.Nil(t, got.FlamebearerProfile.Timeline)
}
