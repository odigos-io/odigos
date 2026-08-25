package profilecache

import (
	"sync"
	"testing"
	"time"
)

func chunkOfSize(size int, fill byte) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = fill
	}
	return data
}

func TestBoundedBufferEvictsOldestChunksToFitBudget(t *testing.T) {
	b := NewBoundedBuffer(300)
	b.Add(chunkOfSize(100, 'a'))
	b.Add(chunkOfSize(100, 'b'))
	b.Add(chunkOfSize(100, 'c'))
	if b.Size() != 300 {
		t.Fatalf("size = %d, want 300", b.Size())
	}
	if b.evictedTotal != 0 {
		t.Fatalf("evictedTotal = %d, want 0 while the buffer still fits", b.evictedTotal)
	}

	// One chunk over the budget evicts exactly the oldest one.
	b.Add(chunkOfSize(100, 'd'))
	snapshot := b.Snapshot()
	if len(snapshot) != 3 {
		t.Fatalf("live chunks = %d, want 3", len(snapshot))
	}
	for i, want := range []byte{'b', 'c', 'd'} {
		if snapshot[i][0] != want {
			t.Fatalf("chunk %d starts with %q, want %q (FIFO order broken)", i, snapshot[i][0], want)
		}
	}
	if b.Size() != 300 {
		t.Fatalf("size = %d, want 300 (byte accounting drifted from the live chunks)", b.Size())
	}
	if b.addedTotal != 4 || b.evictedTotal != 1 {
		t.Fatalf("addedTotal = %d, evictedTotal = %d, want 4 and 1", b.addedTotal, b.evictedTotal)
	}
}

func TestBoundedBufferEvictsAsManyChunksAsNeeded(t *testing.T) {
	b := NewBoundedBuffer(500)
	for i := range 5 {
		b.Add(chunkOfSize(100, byte('a'+i)))
	}

	// A chunk that fits the budget on its own but needs room for 4 of the 5 existing chunks.
	b.Add(chunkOfSize(450, 'z'))
	snapshot := b.Snapshot()
	if len(snapshot) != 1 {
		t.Fatalf("live chunks = %d, want 1", len(snapshot))
	}
	if snapshot[0][0] != 'z' {
		t.Fatalf("surviving chunk starts with %q, want %q", snapshot[0][0], byte('z'))
	}
	if b.Size() != 450 {
		t.Fatalf("size = %d, want 450", b.Size())
	}
	if b.evictedTotal != 5 {
		t.Fatalf("evictedTotal = %d, want 5", b.evictedTotal)
	}
}

func TestBoundedBufferRejectsChunkLargerThanBudget(t *testing.T) {
	b := NewBoundedBuffer(300)
	b.Add(chunkOfSize(100, 'a'))

	// An oversized chunk can never be retained, so it must be dropped without evicting
	// what is already buffered.
	if b.AddAt(time.Now(), chunkOfSize(301, 'x')) {
		t.Fatal("AddAt returned true for a chunk larger than the whole budget")
	}
	snapshot := b.Snapshot()
	if len(snapshot) != 1 || snapshot[0][0] != 'a' {
		t.Fatalf("live chunks = %v, want the single pre-existing chunk", snapshot)
	}
	if b.Size() != 100 {
		t.Fatalf("size = %d, want 100", b.Size())
	}
	if b.addedTotal != 1 {
		t.Fatalf("addedTotal = %d, want 1 (the rejected chunk must not be counted)", b.addedTotal)
	}
}

func TestBoundedBufferIgnoresEmptyChunk(t *testing.T) {
	b := NewBoundedBuffer(300)
	if !b.Add(nil) {
		t.Fatal("Add(nil) returned false")
	}
	if !b.Add([]byte{}) {
		t.Fatal("Add(empty) returned false")
	}
	if b.Size() != 0 {
		t.Fatalf("size = %d, want 0", b.Size())
	}
	if got := b.Snapshot(); got != nil {
		t.Fatalf("snapshot = %v, want nil", got)
	}
	if b.addedTotal != 0 {
		t.Fatalf("addedTotal = %d, want 0", b.addedTotal)
	}
}

func TestBoundedBufferNonPositiveBudgetDisablesEviction(t *testing.T) {
	b := NewBoundedBuffer(0)
	for range 5 {
		b.Add(chunkOfSize(1000, 'a'))
	}
	if b.Size() != 5000 {
		t.Fatalf("size = %d, want 5000", b.Size())
	}
	if b.evictedTotal != 0 {
		t.Fatalf("evictedTotal = %d, want 0", b.evictedTotal)
	}
}

func TestBoundedBufferSnapshotSinceIsInclusive(t *testing.T) {
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	b := NewBoundedBuffer(1000)
	b.AddAt(base, chunkOfSize(10, 'a'))
	b.AddAt(base.Add(time.Second), chunkOfSize(10, 'b'))
	b.AddAt(base.Add(2*time.Second), chunkOfSize(10, 'c'))

	// The UI polls with the capture time of the last chunk it received, so the boundary has to be
	// inclusive: exclusive would silently drop a chunk on every poll.
	got := b.SnapshotSince(base.Add(time.Second))
	if len(got) != 2 {
		t.Fatalf("chunks since base+1s = %d, want 2", len(got))
	}
	if got[0][0] != 'b' || got[1][0] != 'c' {
		t.Fatalf("chunks since base+1s = %q%q, want \"b\"\"c\"", got[0][0], got[1][0])
	}

	if all := b.SnapshotSince(time.Time{}); len(all) != 3 {
		t.Fatalf("chunks since the zero time = %d, want all 3", len(all))
	}
	if none := b.SnapshotSince(base.Add(time.Hour)); none != nil {
		t.Fatalf("chunks since base+1h = %v, want nil", none)
	}
}

func TestBoundedBufferCompactsBackingArrayUnderChurn(t *testing.T) {
	// The budget holds 10 chunks; the backing array must not grow with the number of adds.
	b := NewBoundedBuffer(1000)
	for range 10000 {
		b.Add(chunkOfSize(100, 'a'))
	}
	if len(b.chunks) != 10 {
		t.Fatalf("live chunks = %d, want 10", len(b.chunks))
	}
	if cap(b.chunks) > 64 {
		t.Fatalf("backing array cap = %d after 10000 adds, want it compacted to the live size", cap(b.chunks))
	}
	if b.Size() != 1000 {
		t.Fatalf("size = %d, want 1000", b.Size())
	}
}

func TestBoundedBufferClear(t *testing.T) {
	b := NewBoundedBuffer(1000)
	b.Add(chunkOfSize(100, 'a'))
	b.Clear()
	if b.Size() != 0 {
		t.Fatalf("size after Clear = %d, want 0", b.Size())
	}
	if got := b.Snapshot(); got != nil {
		t.Fatalf("snapshot after Clear = %v, want nil", got)
	}

	// The buffer stays usable, with the budget intact.
	b.Add(chunkOfSize(100, 'b'))
	if b.Size() != 100 {
		t.Fatalf("size after reuse = %d, want 100", b.Size())
	}
}

func TestBoundedBufferConcurrentAccess(t *testing.T) {
	b := NewBoundedBuffer(1000)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				b.Add(chunkOfSize(100, 'a'))
				b.Snapshot()
				b.Size()
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 200 {
			b.Resize(1000)
		}
	}()
	wg.Wait()

	if size := b.Size(); size > 1000 {
		t.Fatalf("size = %d, want the budget to hold under concurrency", size)
	}
}
