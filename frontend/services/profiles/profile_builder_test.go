package profiles

import (
	_ "embed"
	"testing"

	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

//go:embed flamegraph/testdata/minimal_otlp_profile.json
var minimalOTLPProfileJSON []byte

// minimalOTLPProfileChunk is JSON (protojson form) → protobuf wire, same shape as stored chunks.
func minimalOTLPProfileChunk(t *testing.T) []byte {
	t.Helper()
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	require.NoError(t, protojson.Unmarshal(minimalOTLPProfileJSON, req))
	b, err := proto.Marshal(req)
	require.NoError(t, err)
	return b
}

// Synthetic profile in testdata: one sample, value 100, stack aligned with tree_test example stacks.
var wantSyntheticFrames = []string{"runtime_main", "net_http_serve", "handler_process"}

func TestPipeline_ChunkToSamples_minimalTestdata(t *testing.T) {
	t.Parallel()
	chunk := minimalOTLPProfileChunk(t)
	samples := flamegraph.SamplesFromOTLPChunk(chunk)
	require.Len(t, samples, 1, "testdata should decode to exactly one stack sample")
	assert.Equal(t, int64(100), samples[0].Value)
	require.Len(t, samples[0].Stack, len(wantSyntheticFrames))
	assert.Equal(t, wantSyntheticFrames, samples[0].Stack, "root-first order must match InsertStack / tree tests")
}

func TestPipeline_FlamegraphProfile_singleChunk(t *testing.T) {
	t.Parallel()
	chunk := minimalOTLPProfileChunk(t)
	out := BuildFlamegraphProfileFromChunks([][]byte{chunk})
	fb := out.Flamebearer
	assert.Equal(t, int64(100), fb.NumTicks)
	assert.Equal(t, int64(100), fb.MaxSelf)
	for _, name := range wantSyntheticFrames {
		assert.Contains(t, fb.Names, name)
	}
	require.NotNil(t, out.Timeline)
	assert.Equal(t, int64(1234567890), out.Timeline.StartTime, "timeUnixNano in testdata → seconds on timeline")
}

func TestPipeline_FlamegraphProfile_twoIdenticalChunks_mergeWeights(t *testing.T) {
	t.Parallel()
	chunk := minimalOTLPProfileChunk(t)
	out := BuildFlamegraphProfileFromChunks([][]byte{chunk, chunk})
	fb := out.Flamebearer
	assert.Equal(t, int64(200), fb.NumTicks, "same stack in two chunks should add sample weights")
	assert.Equal(t, int64(200), fb.MaxSelf)
}

func TestPipeline_FlamegraphProfile_emptyChunks_emptyGraph(t *testing.T) {
	t.Parallel()
	out := BuildFlamegraphProfileFromChunks(nil)
	assert.Equal(t, int64(0), out.Flamebearer.NumTicks)
	assert.Nil(t, out.Timeline)
}
