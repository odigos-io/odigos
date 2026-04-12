package profiles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func chunkTimeProtoFromJSON(t *testing.T, json string) []byte {
	t.Helper()
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	require.NoError(t, protojson.Unmarshal([]byte(json), req))
	b, err := proto.Marshal(req)
	require.NoError(t, err)
	return b
}

func TestEarliestProfileStartTimeUnixSec(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  [][]byte
		want int64
	}{
		{
			name: "empty",
			raw:  nil,
			want: 0,
		},
		{
			name: "invalid_proto_skipped",
			raw:  [][]byte{[]byte("not-protobuf")},
			want: 0,
		},
		{
			name: "single_profile",
			raw: [][]byte{
				chunkTimeProtoFromJSON(t, `{
				"resourceProfiles": [{
					"scopeProfiles": [{
						"profiles": [{ "timeUnixNano": "1700000000123456789" }]
					}]
				}],
				"dictionary": { "stringTable": ["a"] }
			}`),
			},
			want: 1700000000,
		},
		{
			name: "picks_earliest_across_chunks",
			raw: [][]byte{
				chunkTimeProtoFromJSON(t, `{"resourceProfiles":[{"scopeProfiles":[{"profiles":[{"timeUnixNano":"2000000000000000000"}]}]}],"dictionary":{"stringTable":["a"]}}`),
				chunkTimeProtoFromJSON(t, `{"resourceProfiles":[{"scopeProfiles":[{"profiles":[{"timeUnixNano":"1000000000000000000"}]}]}],"dictionary":{"stringTable":["a"]}}`),
			},
			want: 1000000000,
		},
		{
			name: "ignores_zero_time",
			raw: [][]byte{
				chunkTimeProtoFromJSON(t, `{
				"resourceProfiles": [{
					"scopeProfiles": [{
						"profiles": [
							{ "timeUnixNano": "0" },
							{ "timeUnixNano": "500000000000000000" }
						]
					}]
				}],
				"dictionary": { "stringTable": ["a"] }
			}`),
			},
			want: 500000000,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := earliestProfileStartTimeUnixSec(tc.raw)
			assert.Equal(t, tc.want, got)
		})
	}
}
