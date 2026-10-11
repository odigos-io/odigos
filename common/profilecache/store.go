package profilecache

import (
	"context"
	"sync"
	"time"
)

type Slot struct {
	lastRequestAt time.Time
	buffer        *BoundedBuffer
}

// Store keeps at most maxSlots source-keyed slots, each buffer bounded by slotMaxBytes.
// A slot is evicted when the count hits maxSlots (least-recently-used) or when it is
// idle past ttlSeconds. Total memory is bounded by maxSlots × slotMaxBytes.
type Store struct {
	mu              sync.RWMutex
	slots           map[string]*Slot
	maxSlots        int
	ttlSeconds      int
	slotMaxBytes    int
	cleanupInterval time.Duration
	evictedSlots    uint64
}

// MemoryStats summarizes buffered data and configured limits for the UI / TUI.
type MemoryStats struct {
	TotalBytes          int
	MaxSlots            int
	SlotMaxBytes        int
	SlotTTLSeconds      int
	MaxTotalBytesBudget int
}

// StoreRef is the read/lifecycle API the frontend GraphQL layer depends on.
type StoreRef interface {
	EnsureSlot(sourceKey string)
	RemoveSlot(sourceKey string)
	ClearSlotBuffer(sourceKey string) bool
	GetProfileData(sourceKey string) [][]byte
	MaxSlots() int
	ActiveSlots() (activeKeys []string, keysWithData []string)
	MemoryStats() MemoryStats
	Reconfigure(maxSlots, slotMaxBytes, ttlSeconds int)
}

// StoreConfig holds the cache limits for NewStore. Any non-positive field falls
// back to the package default.
type StoreConfig struct {
	MaxSlots        int
	TTLSeconds      int
	SlotMaxBytes    int
	CleanupInterval time.Duration
}

func NewStore(cfg StoreConfig) *Store {
	if cfg.MaxSlots <= 0 {
		cfg.MaxSlots = DefaultMaxSlots
	}
	if cfg.TTLSeconds <= 0 {
		cfg.TTLSeconds = DefaultSlotTTLSeconds
	}
	if cfg.SlotMaxBytes <= 0 {
		cfg.SlotMaxBytes = DefaultSlotMaxBytes
	}
	if cfg.CleanupInterval <= 0 {
		cfg.CleanupInterval = DefaultCleanupInterval
	}
	return &Store{
		slots:           make(map[string]*Slot),
		maxSlots:        cfg.MaxSlots,
		ttlSeconds:      cfg.TTLSeconds,
		slotMaxBytes:    cfg.SlotMaxBytes,
		cleanupInterval: cfg.CleanupInterval,
	}
}

func (s *Store) evictOldestSlotLocked() bool {
	var oldestKey string
	var oldest time.Time
	for k, slot := range s.slots {
		if oldestKey == "" || slot.lastRequestAt.Before(oldest) {
			oldest = slot.lastRequestAt
			oldestKey = k
		}
	}
	if oldestKey == "" {
		return false
	}
	delete(s.slots, oldestKey)
	s.evictedSlots++
	return true
}

func (s *Store) totalBytesLocked() int {
	total := 0
	for _, slot := range s.slots {
		if slot.buffer != nil {
			total += slot.buffer.Size()
		}
	}
	return total
}

// EnsureSlot opens a slot for sourceKey, or refreshes its request time if present.
func (s *Store) EnsureSlot(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if slot, ok := s.slots[sourceKey]; ok {
		slot.lastRequestAt = time.Now()
		return
	}
	if len(s.slots) >= s.maxSlots {
		s.evictOldestSlotLocked()
	}
	s.slots[sourceKey] = &Slot{lastRequestAt: time.Now(), buffer: NewBoundedBuffer(s.slotMaxBytes)}
}

func (s *Store) RemoveSlot(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.slots, sourceKey)
}

func (s *Store) ClearAllSlots() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slots = make(map[string]*Slot)
}

// ClearSlotBuffer empties a source's buffer but keeps the slot.
func (s *Store) ClearSlotBuffer(sourceKey string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[sourceKey]
	if !ok || slot == nil {
		return false
	}
	slot.lastRequestAt = time.Now()
	if slot.buffer != nil {
		slot.buffer.Clear()
	}
	return true
}

func (s *Store) MaxSlots() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSlots
}

func (s *Store) MemoryStats() MemoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return MemoryStats{
		TotalBytes:          s.totalBytesLocked(),
		MaxSlots:            s.maxSlots,
		SlotMaxBytes:        s.slotMaxBytes,
		SlotTTLSeconds:      s.ttlSeconds,
		MaxTotalBytesBudget: s.maxSlots * s.slotMaxBytes,
	}
}

// Reconfigure applies new cache limits at runtime; a non-positive argument keeps the
// current value. Slots beyond the new maxSlots are pruned (least-recently-used first)
// and every retained buffer is resized to the new slotMaxBytes.
func (s *Store) Reconfigure(maxSlots, slotMaxBytes, ttlSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if maxSlots > 0 {
		s.maxSlots = maxSlots
	}
	if slotMaxBytes > 0 {
		s.slotMaxBytes = slotMaxBytes
	}
	if ttlSeconds > 0 {
		s.ttlSeconds = ttlSeconds
	}
	for len(s.slots) > s.maxSlots {
		if !s.evictOldestSlotLocked() {
			break
		}
	}
	for _, slot := range s.slots {
		if slot.buffer != nil {
			slot.buffer.Resize(s.slotMaxBytes)
		}
	}
}

// AddProfileData appends a chunk to an existing slot; it is a no-op if the slot was
// not opened (the caller gates which sources are stored).
func (s *Store) AddProfileData(sourceKey string, chunk []byte) {
	s.mu.RLock()
	slot, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	if !ok || slot == nil || slot.buffer == nil {
		return
	}
	slot.buffer.Add(chunk)
}

func (s *Store) GetProfileData(sourceKey string) [][]byte {
	return s.snapshot(sourceKey, time.Time{}, true)
}

func (s *Store) SnapshotSince(sourceKey string, since time.Time) [][]byte {
	return s.snapshot(sourceKey, since, false)
}

func (s *Store) snapshot(sourceKey string, since time.Time, bumpRequest bool) [][]byte {
	s.mu.Lock()
	slot, ok := s.slots[sourceKey]
	if ok && slot != nil && bumpRequest {
		slot.lastRequestAt = time.Now()
	}
	s.mu.Unlock()
	if !ok || slot == nil || slot.buffer == nil {
		return nil
	}
	return slot.buffer.SnapshotSince(since)
}

func (s *Store) SnapshotAllSince(since time.Time) [][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out [][]byte
	for _, slot := range s.slots {
		if slot.buffer != nil {
			out = append(out, slot.buffer.SnapshotSince(since)...)
		}
	}
	return out
}

func (s *Store) IsActive(sourceKey string) bool {
	s.mu.RLock()
	_, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	return ok
}

func (s *Store) ActiveSlots() (activeKeys []string, keysWithData []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, slot := range s.slots {
		activeKeys = append(activeKeys, k)
		if slot.buffer != nil && slot.buffer.Size() > 0 {
			keysWithData = append(keysWithData, k)
		}
	}
	return activeKeys, keysWithData
}

func (s *Store) EvictedSlots() uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.evictedSlots
}

// RunCleanup sweeps idle slots every cleanupInterval until ctx is canceled. It
// blocks, so callers run it in the background: go store.RunCleanup(ctx).
func (s *Store) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepNow()
		}
	}
}

func (s *Store) SweepNow() {
	cutoff := time.Now().Add(-time.Duration(s.ttlSeconds) * time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, slot := range s.slots {
		if slot.lastRequestAt.Before(cutoff) {
			delete(s.slots, k)
		}
	}
}
