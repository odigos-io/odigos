package transport

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadWriteFrameRoundTrip(t *testing.T) {
	payload := []byte("hello-opamp")
	var buf bytes.Buffer
	require.NoError(t, WriteFrame(&buf, payload))
	got, err := ReadFrame(&buf)
	require.NoError(t, err)
	require.Equal(t, payload, got)
}

func TestReadFrameRejectsOversized(t *testing.T) {
	var buf bytes.Buffer
	oversized := make([]byte, maxFrameSize+1)
	require.Error(t, WriteFrame(&buf, oversized))
}

func TestReadFrameRejectsEmpty(t *testing.T) {
	var buf bytes.Buffer
	buf.Write([]byte{0, 0, 0, 0})
	_, err := ReadFrame(&buf)
	require.Error(t, err)
}
