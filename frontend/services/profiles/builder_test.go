package profiles

import (
	"context"
	"testing"

	pyrometadata "github.com/grafana/pyroscope/pkg/og/storage/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPyroscopeMetadataDescribesASingleCpuProfile(t *testing.T) {
	metadata := pyroscopeMetadata()

	assert.Equal(t, "single", metadata.Format)
	assert.Equal(t, "cpu", metadata.Name)
	assert.Equal(t, pyrometadata.Units("samples"), metadata.Units)
	assert.Equal(t, uint32(1_000_000_000), metadata.SampleRate)
	assert.Equal(t, "", metadata.SpyName)
}

func TestPyroscopeTimelineIsOmittedWithoutSamples(t *testing.T) {
	assert.Nil(t, pyroscopeTimeline(0, 1_700_000_000))
}

func TestPyroscopeTimelineCarriesTheStartTimeAndWeight(t *testing.T) {
	timeline := pyroscopeTimeline(42, 1_700_000_000)

	require.NotNil(t, timeline)
	assert.Equal(t, int64(1_700_000_000), timeline.StartTime)
	assert.Equal(t, []uint64{0, 42}, timeline.Samples)
	assert.Equal(t, int64(15), timeline.DurationDelta)
	assert.Nil(t, timeline.Watermarks)
}

func TestBuildPyroscopeProfileFromChunksAssemblesTheWholeResponse(t *testing.T) {
	stamped := checkoutWorkloadProfile([]string{"main", "handler"}, 9)
	stamped.timeNano = 1_700_000_123_000_000_000
	chunks := [][]byte{otlpChunk(t, otlpProfilesBatch(t, stamped))}

	profile := buildPyroscopeProfileFromChunks(context.Background(), chunks)

	require.NotNil(t, profile.FlamebearerProfile)
	assert.Equal(t, 9, profile.Flamebearer.NumTicks)
	assert.Subset(t, profile.Flamebearer.Names, []string{"main", "handler"})

	require.NotNil(t, profile.Timeline)
	assert.Equal(t, int64(1_700_000_123), profile.Timeline.StartTime)
	assert.Equal(t, []uint64{0, 9}, profile.Timeline.Samples)

	symbolNames := make([]string, 0, len(profile.Symbols))
	for _, s := range profile.Symbols {
		symbolNames = append(symbolNames, s.Name)
	}
	assert.Subset(t, symbolNames, []string{"main", "handler"})
}

// Pyroscope's own export already fills in the metadata for a profile it could build; the Odigos
// default is only a backstop for the profiles it could not.
func TestBuildPyroscopeProfileFromChunksKeepsPyroscopesOwnMetadata(t *testing.T) {
	chunks := [][]byte{otlpChunk(t, otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5)))}

	profile := buildPyroscopeProfileFromChunks(context.Background(), chunks)

	require.NotNil(t, profile.FlamebearerProfile)
	assert.Equal(t, "alloc_space", profile.Metadata.Name)
	assert.Equal(t, pyrometadata.Units("bytes"), profile.Metadata.Units)
}

func TestBuildPyroscopeProfileFromChunksFallsBackToTheDefaultMetadata(t *testing.T) {
	profile := buildPyroscopeProfileFromChunks(context.Background(), [][]byte{[]byte("garbage")})

	require.NotNil(t, profile.FlamebearerProfile)
	assert.Equal(t, pyroscopeMetadata(), profile.Metadata)
	assert.Equal(t, []string{"total"}, profile.Flamebearer.Names)
	assert.Equal(t, 0, profile.Flamebearer.NumTicks)
	assert.Nil(t, profile.Timeline)
}

func TestBuildPyroscopeProfileFromChunksWithoutChunksIsEmpty(t *testing.T) {
	profile := buildPyroscopeProfileFromChunks(context.Background(), nil)

	require.NotNil(t, profile.FlamebearerProfile)
	assert.Equal(t, 0, profile.Flamebearer.NumTicks)
	assert.Nil(t, profile.Timeline)
	assert.Empty(t, profile.Symbols)
}

func TestBuildPyroscopeProfileFromChunksMergesChunksFromTheSameWorkload(t *testing.T) {
	first := checkoutWorkloadProfile([]string{"main", "handler"}, 4)
	first.timeNano = 1_700_000_200_000_000_000
	second := checkoutWorkloadProfile([]string{"main", "handler"}, 6)
	second.timeNano = 1_700_000_100_000_000_000

	profile := buildPyroscopeProfileFromChunks(context.Background(), [][]byte{
		otlpChunk(t, otlpProfilesBatch(t, first)),
		otlpChunk(t, otlpProfilesBatch(t, second)),
	})

	require.NotNil(t, profile.FlamebearerProfile)
	assert.Equal(t, 10, profile.Flamebearer.NumTicks)
	// The timeline starts at the earliest chunk, not the first one buffered.
	require.NotNil(t, profile.Timeline)
	assert.Equal(t, int64(1_700_000_100), profile.Timeline.StartTime)
}
