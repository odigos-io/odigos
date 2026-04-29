package profiles

import (
	"sync"
)

// BoundedBuffer keeps a size-bounded list of profile data chunks (raw bytes).
// Each chunk is one full pdata ProtoMarshaler.MarshalProfiles blob (OTLP ExportProfilesServiceRequest wire).
// Stored chunk bytes are immutable after append; only whole chunks are dropped from the list.

type BoundedBuffer struct {
	mu         sync.RWMutex
	chunks     [][]byte
	totalBytes int
	maxBytes   int
}

func NewBoundedBuffer(maxBytes int) *BoundedBuffer {
	return &BoundedBuffer{maxBytes: maxBytes}
}

// Add appends a full chunk, then evicts whole oldest chunks until total size is at most maxBytes.
func (b *BoundedBuffer) Add(chunk []byte) bool {
	if len(chunk) == 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chunks = append(b.chunks, chunk)
	b.totalBytes += len(chunk)
	b.trimToMaxLocked()
	return true
}

// trimToMaxLocked removes whole oldest chunks so total size stays within maxBytes across many Add calls.
func (b *BoundedBuffer) trimToMaxLocked() {
	for len(b.chunks) > 0 && b.totalBytes > b.maxBytes {
		old := b.chunks[0]
		b.chunks = b.chunks[1:]
		b.totalBytes -= len(old)
	}
}

// Snapshot returns a shallow copy of the chunk list
func (b *BoundedBuffer) Snapshot() [][]byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.chunks) == 0 {
		return nil
	}
	out := make([][]byte, len(b.chunks))
	copy(out, b.chunks)
	return out
}

func (b *BoundedBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalBytes
}

func (b *BoundedBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chunks = nil
	b.totalBytes = 0
}
