package profilecache

import (
	"testing"
	"time"
)

func TestBoundedBuffer_Resize(t *testing.T) {
	b := NewBoundedBuffer(1000)
	b.Add(make([]byte, 400))
	b.Add(make([]byte, 400))
	if b.Size() != 800 {
		t.Fatalf("size = %d, want 800", b.Size())
	}

	// Shrink below current usage → oldest chunk trimmed.
	b.Resize(500)
	if b.Size() != 400 {
		t.Fatalf("after Resize(500) size = %d, want 400", b.Size())
	}

	// Grow → no trim; subsequent adds fit.
	b.Resize(2000)
	b.Add(make([]byte, 600))
	if b.Size() != 1000 {
		t.Fatalf("after grow+add size = %d, want 1000", b.Size())
	}
}

func TestStore_Reconfigure(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	for _, key := range []string{"a", "b", "c", "d"} {
		s.EnsureSlot(key)
		s.AddProfileData(key, make([]byte, 400))
	}
	if active, _ := s.ActiveSlots(); len(active) != 4 {
		t.Fatalf("active = %d, want 4", len(active))
	}

	// Shrink maxSlots → evict the 2 LRU slots.
	s.Reconfigure(2, 0, 0)
	if active, _ := s.ActiveSlots(); len(active) != 2 {
		t.Fatalf("after maxSlots=2, active = %d, want 2", len(active))
	}

	// Shrink slotMaxBytes below the chunk size → each buffer trims.
	before := s.MemoryStats().TotalBytes
	s.Reconfigure(0, 200, 0)
	if after := s.MemoryStats().TotalBytes; after >= before {
		t.Fatalf("slotMaxBytes shrink didn't trim: before=%d after=%d", before, after)
	}

	// ttl-only change; 0 args must leave maxSlots/slotMaxBytes untouched.
	s.Reconfigure(0, 0, 900)
	ms := s.MemoryStats()
	if ms.SlotTTLSeconds != 900 || ms.MaxSlots != 2 || ms.SlotMaxBytes != 200 {
		t.Fatalf("ttl-only reconfigure changed other fields: %+v", ms)
	}
}
