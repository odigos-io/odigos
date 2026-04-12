package otlpchunk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalExportProfilesRequest_Empty(t *testing.T) {
	_, err := UnmarshalExportProfilesRequest(nil)
	require.Error(t, err)

	_, err = UnmarshalExportProfilesRequest([]byte{})
	require.Error(t, err)
}

func TestUnmarshalExportProfilesRequest_InvalidWire(t *testing.T) {
	_, err := UnmarshalExportProfilesRequest([]byte{0xff, 0xff})
	assert.Error(t, err)
}
