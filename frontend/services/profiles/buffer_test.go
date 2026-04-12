package profiles

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestBoundedBuffer_AddEvictsWholeOldestChunks(t *testing.T) {
	b := NewBoundedBuffer(9)
	b.Add([]byte("12345"))
	b.Add([]byte("67890"))
	require.Equal(t, 5, b.Size())
	require.Equal(t, [][]byte{[]byte("67890")}, b.Snapshot())
}

func TestBoundedBuffer_AddDropsOversizeChunkWhenCapped(t *testing.T) {
	b := NewBoundedBuffer(5)
	b.Add([]byte("1234567890"))
	require.Equal(t, 0, b.Size())
	require.Nil(t, b.Snapshot())
}

func TestBoundedBuffer_MaxBytesZeroTrimsImmediately(t *testing.T) {
	b := NewBoundedBuffer(0)
	b.Add([]byte("12"))
	require.Equal(t, 0, b.Size())
	require.Nil(t, b.Snapshot())
}

func TestBoundedBuffer_SnapshotShallowCopiesSliceHeaders(t *testing.T) {
	b := NewBoundedBuffer(100)
	payload := []byte("hello")
	b.Add(payload)
	s1 := b.Snapshot()
	s2 := b.Snapshot()
	require.Len(t, s1, 1)
	require.Len(t, s2, 1)
	require.Equal(t, "hello", string(s1[0]))
	require.Equal(t, "hello", string(s2[0]))
	require.Equal(t, unsafe.SliceData(s1[0]), unsafe.SliceData(s2[0]))
}
