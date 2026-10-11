package profilecache

import (
	"sync"
	"time"
)

// compactThreshold triggers a slice copy to release the evicted prefix once the
// live count drops below 1/compactThreshold of cap.
const compactThreshold = 2

// chunk is one raw OTLP ExportProfilesServiceRequest blob and the time it was stored.
type chunk struct {
	capturedAt time.Time
	bytes      []byte
}

// BoundedBuffer is a byte-budgeted FIFO of raw profile chunks.
// Whole oldest chunks are evicted when the total exceeds maxBytes;
// evicted chunks are zeroed and the backing slice is compacted so it does not grow without bound under churn.
type BoundedBuffer struct {
	mu           sync.RWMutex
	chunks       []chunk
	totalBytes   int
	maxBytes     int
	addedTotal   uint64
	evictedTotal uint64
}

func NewBoundedBuffer(maxBytes int) *BoundedBuffer {
	return &BoundedBuffer{maxBytes: maxBytes}
}

// Add stores a chunk stamped at the current time.
func (b *BoundedBuffer) Add(data []byte) bool {
	return b.AddAt(time.Now(), data)
}

// AddAt stores a chunk with an explicit capture time.
// It returns false only when a single chunk exceeds the whole budget and can never be retained.
func (b *BoundedBuffer) AddAt(capturedAt time.Time, data []byte) bool {
	if len(data) == 0 {
		return true
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.maxBytes > 0 && len(data) > b.maxBytes {
		return false
	}

	b.chunks = append(b.chunks, chunk{capturedAt: capturedAt, bytes: data})
	b.totalBytes += len(data)
	b.addedTotal++
	b.trimLocked()
	return true
}

// Resize sets a new byte budget and evicts oldest chunks to fit it.
func (b *BoundedBuffer) Resize(maxBytes int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maxBytes = maxBytes
	b.trimLocked()
}

// trimLocked evicts oldest chunks until the buffer fits maxBytes. The caller must hold b.mu.
func (b *BoundedBuffer) trimLocked() {
	dropped := 0
	for dropped < len(b.chunks) && b.maxBytes > 0 && b.totalBytes > b.maxBytes {
		b.totalBytes -= len(b.chunks[dropped].bytes)
		b.chunks[dropped] = chunk{}
		dropped++
	}
	if dropped == 0 {
		return
	}
	b.evictedTotal += uint64(dropped)

	live := b.chunks[dropped:]
	if cap(b.chunks) >= compactThreshold*len(live) && cap(b.chunks) > 16 {
		fresh := make([]chunk, len(live))
		copy(fresh, live)
		b.chunks = fresh
	} else {
		b.chunks = live
	}
}

// Snapshot returns the bytes of every live chunk.
func (b *BoundedBuffer) Snapshot() [][]byte {
	return b.SnapshotSince(time.Time{})
}

// SnapshotSince returns the bytes of chunks captured at or after since.
func (b *BoundedBuffer) SnapshotSince(since time.Time) [][]byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.chunks) == 0 {
		return nil
	}
	out := make([][]byte, 0, len(b.chunks))
	for _, c := range b.chunks {
		if c.bytes == nil || c.capturedAt.Before(since) {
			continue
		}
		out = append(out, c.bytes)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (b *BoundedBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chunks = nil
	b.totalBytes = 0
}

func (b *BoundedBuffer) Size() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.totalBytes
}
