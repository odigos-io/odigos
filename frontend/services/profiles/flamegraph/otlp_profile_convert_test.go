package flamegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func missingDictionaryChunkProto(t *testing.T) []byte {
	t.Helper()
	const badJSON = `{"resourceProfiles":[{"scopeProfiles":[{"profiles":[{"timeUnixNano":"1"}]}]}]}`
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	require.NoError(t, protojson.Unmarshal([]byte(badJSON), req))
	b, err := proto.Marshal(req)
	require.NoError(t, err)
	return b
}

func TestDecodeOTLPChunkToSamples_EmptyChunk(t *testing.T) {
	t.Parallel()
	_, ok, reason := decodeOTLPChunkToSamples(nil)
	assert.False(t, ok)
	assert.Equal(t, "empty_chunk", reason)
}

func TestDecodeOTLPChunkToSamples_MissingDictionary(t *testing.T) {
	t.Parallel()
	bad := missingDictionaryChunkProto(t)
	_, ok, reason := decodeOTLPChunkToSamples(bad)
	assert.False(t, ok)
	assert.Equal(t, "missing_or_empty_dictionary_string_table", reason)
}

func TestCollapseMultiValueSamplesToSingleCounter_SumsMultipleValues(t *testing.T) {
	t.Parallel()
	p := &otelProfile.Profile{
		Samples: []*otelProfile.Sample{{
			Values: []int64{10, 20, 5},
		}},
	}
	collapseMultiValueSamplesToSingleCounter(p)
	require.Len(t, p.Samples[0].Values, 1)
	assert.Equal(t, int64(35), p.Samples[0].Values[0])
}
