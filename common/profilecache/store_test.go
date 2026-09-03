package profilecache

import (
	"context"
	"sync"
	"testing"
	"time"
)

// The frontend consumes the store through this interface; keep the concrete type satisfying it.
var _ StoreRef = (*Store)(nil)

// EnsureSlot stamps lastRequestAt with time.Now(), so tests that depend on the recency order of
// their slots need the stamps to be distinguishable.
func ensureSlotThenAdvanceClock(t *testing.T, s *Store, sourceKey string) {
	t.Helper()
	s.EnsureSlot(sourceKey)
	time.Sleep(2 * time.Millisecond)
}

func TestStoreNewStoreAppliesDefaultsForNonPositiveConfig(t *testing.T) {
	s := NewStore(StoreConfig{})
	stats := s.MemoryStats()
	if stats.MaxSlots != DefaultMaxSlots {
		t.Fatalf("MaxSlots = %d, want %d", stats.MaxSlots, DefaultMaxSlots)
	}
	if stats.SlotMaxBytes != DefaultSlotMaxBytes {
		t.Fatalf("SlotMaxBytes = %d, want %d", stats.SlotMaxBytes, DefaultSlotMaxBytes)
	}
	if stats.SlotTTLSeconds != DefaultSlotTTLSeconds {
		t.Fatalf("SlotTTLSeconds = %d, want %d", stats.SlotTTLSeconds, DefaultSlotTTLSeconds)
	}
	if s.cleanupInterval != DefaultCleanupInterval {
		t.Fatalf("cleanupInterval = %v, want %v", s.cleanupInterval, DefaultCleanupInterval)
	}

	explicit := NewStore(StoreConfig{MaxSlots: 3, TTLSeconds: 60, SlotMaxBytes: 512, CleanupInterval: time.Second})
	stats = explicit.MemoryStats()
	if stats.MaxSlots != 3 || stats.SlotMaxBytes != 512 || stats.SlotTTLSeconds != 60 {
		t.Fatalf("explicit config not honoured: %+v", stats)
	}
	if explicit.cleanupInterval != time.Second {
		t.Fatalf("cleanupInterval = %v, want 1s", explicit.cleanupInterval)
	}
}

func TestStoreEnsureSlotEvictsLeastRecentlyRequestedSlot(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	ensureSlotThenAdvanceClock(t, s, "oldest")
	ensureSlotThenAdvanceClock(t, s, "newer")

	s.EnsureSlot("newest")
	if s.IsActive("oldest") {
		t.Fatal("the least recently requested slot was not evicted at capacity")
	}
	if !s.IsActive("newer") || !s.IsActive("newest") {
		t.Fatalf("wrong slot evicted: newer=%v newest=%v", s.IsActive("newer"), s.IsActive("newest"))
	}
	if s.MaxSlots() != 2 {
		t.Fatalf("MaxSlots = %d, want 2", s.MaxSlots())
	}
	if got := s.EvictedSlots(); got != 1 {
		t.Fatalf("EvictedSlots = %d, want 1", got)
	}
}

func TestStoreEnsureSlotOnExistingKeyPreservesBufferedData(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	s.EnsureSlot("app")
	s.AddProfileData("app", chunkOfSize(100, 'a'))

	// The UI calls EnsureSlot on every poll; re-opening a live slot must not discard its buffer.
	s.EnsureSlot("app")
	if got := s.GetProfileData("app"); len(got) != 1 {
		t.Fatalf("chunks after re-opening the slot = %d, want 1", len(got))
	}
	if total := s.MemoryStats().TotalBytes; total != 100 {
		t.Fatalf("TotalBytes = %d, want 100", total)
	}
	if got := s.EvictedSlots(); got != 0 {
		t.Fatalf("EvictedSlots = %d, want 0 (refreshing a slot is not an eviction)", got)
	}
}

func TestStoreAddProfileDataIgnoresUnopenedSlot(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})

	// Buffering is opt-in per source: data for a source nobody asked for must not create a slot,
	// otherwise every profiled workload in the cluster consumes the UI's memory budget.
	s.AddProfileData("never-opened", chunkOfSize(100, 'a'))
	if s.IsActive("never-opened") {
		t.Fatal("AddProfileData created a slot for a source that was never opened")
	}
	if total := s.MemoryStats().TotalBytes; total != 0 {
		t.Fatalf("TotalBytes = %d, want 0", total)
	}
	if got := s.GetProfileData("never-opened"); got != nil {
		t.Fatalf("GetProfileData = %v, want nil", got)
	}
}

func TestStoreGetProfileDataRefreshesRecency(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	ensureSlotThenAdvanceClock(t, s, "read")
	ensureSlotThenAdvanceClock(t, s, "idle")

	// Reading a slot marks it as recently used, so the slot nobody reads is the one evicted.
	s.GetProfileData("read")
	s.EnsureSlot("new")

	if !s.IsActive("read") {
		t.Fatal("the slot that was just read got evicted")
	}
	if s.IsActive("idle") {
		t.Fatal("the idle slot survived eviction")
	}
}

func TestStoreSnapshotSinceDoesNotRefreshRecency(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	ensureSlotThenAdvanceClock(t, s, "streamed")
	ensureSlotThenAdvanceClock(t, s, "recent")

	// SnapshotSince serves the background stream, which must not keep a slot alive on its own.
	s.SnapshotSince("streamed", time.Time{})
	s.EnsureSlot("new")

	if s.IsActive("streamed") {
		t.Fatal("SnapshotSince refreshed the slot's recency")
	}
	if !s.IsActive("recent") {
		t.Fatal("the more recently requested slot was evicted")
	}
}

func TestStoreClearSlotBufferKeepsTheSlot(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 2, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	s.EnsureSlot("app")
	s.AddProfileData("app", chunkOfSize(100, 'a'))

	if !s.ClearSlotBuffer("app") {
		t.Fatal("ClearSlotBuffer returned false for an open slot")
	}
	if !s.IsActive("app") {
		t.Fatal("ClearSlotBuffer removed the slot instead of emptying it")
	}
	if got := s.GetProfileData("app"); got != nil {
		t.Fatalf("chunks after clearing = %v, want nil", got)
	}

	// The slot keeps accepting data after being cleared.
	s.AddProfileData("app", chunkOfSize(50, 'b'))
	if total := s.MemoryStats().TotalBytes; total != 50 {
		t.Fatalf("TotalBytes = %d, want 50", total)
	}

	if s.ClearSlotBuffer("unknown") {
		t.Fatal("ClearSlotBuffer returned true for a source with no slot")
	}
}

func TestStoreRemoveSlotAndClearAllSlots(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	for _, key := range []string{"a", "b", "c"} {
		s.EnsureSlot(key)
		s.AddProfileData(key, chunkOfSize(100, 'x'))
	}

	s.RemoveSlot("b")
	if s.IsActive("b") {
		t.Fatal("RemoveSlot left the slot active")
	}
	if !s.IsActive("a") || !s.IsActive("c") {
		t.Fatal("RemoveSlot dropped the wrong slots")
	}
	if total := s.MemoryStats().TotalBytes; total != 200 {
		t.Fatalf("TotalBytes = %d, want 200", total)
	}

	// Removing an unknown key is a no-op.
	s.RemoveSlot("unknown")
	if active, _ := s.ActiveSlots(); len(active) != 2 {
		t.Fatalf("active slots = %d, want 2", len(active))
	}

	s.ClearAllSlots()
	if active, withData := s.ActiveSlots(); len(active) != 0 || len(withData) != 0 {
		t.Fatalf("active = %v, withData = %v, want both empty", active, withData)
	}
	if total := s.MemoryStats().TotalBytes; total != 0 {
		t.Fatalf("TotalBytes = %d, want 0", total)
	}
}

func TestStoreSweepNowDropsOnlyIdleSlots(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 1, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	s.EnsureSlot("idle")
	s.AddProfileData("idle", chunkOfSize(100, 'a'))

	// Sweeping before the TTL elapses must keep everything.
	s.SweepNow()
	if !s.IsActive("idle") {
		t.Fatal("a slot within its TTL was swept")
	}

	time.Sleep(1100 * time.Millisecond)
	s.EnsureSlot("fresh")
	s.SweepNow()

	if s.IsActive("idle") {
		t.Fatal("a slot idle past its TTL was not swept")
	}
	if !s.IsActive("fresh") {
		t.Fatal("a freshly requested slot was swept")
	}
	if total := s.MemoryStats().TotalBytes; total != 0 {
		t.Fatalf("TotalBytes = %d, want 0 after the buffered slot was swept", total)
	}
}

func TestStoreRunCleanupSweepsUntilContextCanceled(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 1, SlotMaxBytes: 1000, CleanupInterval: 10 * time.Millisecond})
	s.EnsureSlot("idle")
	time.Sleep(1100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	stopped := make(chan struct{})
	go func() {
		s.RunCleanup(ctx)
		close(stopped)
	}()

	deadline := time.After(2 * time.Second)
	for s.IsActive("idle") {
		select {
		case <-deadline:
			cancel()
			t.Fatal("RunCleanup never swept the idle slot")
		case <-time.After(5 * time.Millisecond):
		}
	}

	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("RunCleanup did not return after its context was canceled")
	}
}

func TestStoreActiveSlotsSeparatesSlotsWithData(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	s.EnsureSlot("with-data")
	s.AddProfileData("with-data", chunkOfSize(100, 'a'))
	s.EnsureSlot("empty")

	active, withData := s.ActiveSlots()
	if len(active) != 2 {
		t.Fatalf("active = %v, want both slots", active)
	}
	if len(withData) != 1 || withData[0] != "with-data" {
		t.Fatalf("withData = %v, want only [with-data]", withData)
	}
}

func TestStoreMemoryStatsReportsTotalsAndBudget(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 3, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	s.EnsureSlot("a")
	s.AddProfileData("a", chunkOfSize(100, 'a'))
	s.EnsureSlot("b")
	s.AddProfileData("b", chunkOfSize(250, 'b'))

	stats := s.MemoryStats()
	if stats.TotalBytes != 350 {
		t.Fatalf("TotalBytes = %d, want 350", stats.TotalBytes)
	}
	if stats.MaxTotalBytesBudget != 3000 {
		t.Fatalf("MaxTotalBytesBudget = %d, want maxSlots*slotMaxBytes = 3000", stats.MaxTotalBytesBudget)
	}
}

func TestStoreSnapshotAllSinceCoversEverySlot(t *testing.T) {
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, time.UTC)
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	for _, key := range []string{"a", "b"} {
		s.EnsureSlot(key)
		s.slots[key].buffer.AddAt(base, chunkOfSize(10, 'o'))
		s.slots[key].buffer.AddAt(base.Add(time.Minute), chunkOfSize(10, 'n'))
	}

	if got := s.SnapshotAllSince(time.Time{}); len(got) != 4 {
		t.Fatalf("chunks since the zero time = %d, want 4", len(got))
	}
	if got := s.SnapshotAllSince(base.Add(time.Minute)); len(got) != 2 {
		t.Fatalf("chunks since base+1m = %d, want 2 (one per slot)", len(got))
	}
	if got := s.SnapshotAllSince(base.Add(time.Hour)); got != nil {
		t.Fatalf("chunks since base+1h = %v, want nil", got)
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := NewStore(StoreConfig{MaxSlots: 4, TTLSeconds: 300, SlotMaxBytes: 1000, CleanupInterval: time.Minute})
	keys := []string{"a", "b", "c", "d", "e", "f"}
	var wg sync.WaitGroup
	for _, key := range keys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 200 {
				s.EnsureSlot(key)
				s.AddProfileData(key, chunkOfSize(100, 'a'))
				s.GetProfileData(key)
				s.ActiveSlots()
				s.MemoryStats()
				s.SweepNow()
			}
		}()
	}
	wg.Wait()

	stats := s.MemoryStats()
	if active, _ := s.ActiveSlots(); len(active) > stats.MaxSlots {
		t.Fatalf("active slots = %d, want at most maxSlots = %d", len(active), stats.MaxSlots)
	}
	if stats.TotalBytes > stats.MaxTotalBytesBudget {
		t.Fatalf("TotalBytes = %d exceeded the budget %d", stats.TotalBytes, stats.MaxTotalBytesBudget)
	}
}
