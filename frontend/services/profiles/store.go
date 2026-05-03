package profiles

import (
	"context"
	"sync"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/frontend/services/common"
)

type Slot struct {
	LastRequestAt time.Time
	Buffer        *BoundedBuffer
}

// ProfileStore holds at most maxSlots source-keyed slots with a TTL.
// Eviction: when full, the slot with the oldest LastRequestAt is removed.
// TTL: slots with no request in the last ttlSeconds are removed by a background goroutine.
type ProfileStore struct {
	mu              sync.RWMutex
	slots           map[string]*Slot
	maxSlots        int
	ttlSeconds      int
	slotMaxBytes    int
	cleanupInterval time.Duration
	// StopCleanup invokes it to end the TTL goroutine.
	stopCleanup func()
}

// evictOldestSlotLocked removes the slot with the smallest LastRequestAt.
func (s *ProfileStore) evictOldestSlotLocked() {
	var oldestKey string
	var oldestTime time.Time
	for k, slot := range s.slots {
		if oldestKey == "" || slot.LastRequestAt.Before(oldestTime) {
			oldestTime = slot.LastRequestAt
			oldestKey = k
		}
	}
	if oldestKey != "" {
		delete(s.slots, oldestKey)
	}
}

func NewProfileStore(maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) *ProfileStore {
	return &ProfileStore{
		slots:           make(map[string]*Slot),
		maxSlots:        maxSlots,
		ttlSeconds:      ttlSeconds,
		slotMaxBytes:    slotMaxBytes,
		cleanupInterval: cleanupInterval,
	}
}

// EnsureSlot opens a slot for sourceKey if one does not already exist,
// or refreshes its LastRequestAt if it does.
func (s *ProfileStore) EnsureSlot(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	if slot, ok := s.slots[sourceKey]; ok {
		slot.LastRequestAt = now
		return
	}

	if len(s.slots) >= s.maxSlots {
		s.evictOldestSlotLocked()
	}

	s.slots[sourceKey] = &Slot{
		LastRequestAt: now,
		Buffer:        NewBoundedBuffer(s.slotMaxBytes),
	}
}

func (s *ProfileStore) RemoveSlot(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.slots, sourceKey)
}

// ClearAllSlots removes every slot (e.g. when cluster profiling is turned off via effective-config).
func (s *ProfileStore) ClearAllSlots() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.slots = make(map[string]*Slot)
}

// ClearSlotBuffer removes all buffered profile chunks for sourceKey but keeps the slot
func (s *ProfileStore) ClearSlotBuffer(sourceKey string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	slot, ok := s.slots[sourceKey]
	if !ok || slot == nil {
		return false
	}
	slot.LastRequestAt = time.Now()
	if slot.Buffer != nil {
		slot.Buffer.Clear()
	}
	return true
}

func (s *ProfileStore) MaxSlots() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSlots
}

// MemoryStats returns total bytes buffered across slots and the configured limits for debugging purposes
func (s *ProfileStore) MemoryStats() common.ProfileMemoryStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var totalBytes int
	for _, slot := range s.slots {
		if slot.Buffer != nil {
			totalBytes += slot.Buffer.Size()
		}
	}
	return common.ProfileMemoryStats{
		TotalBytes:          totalBytes,
		MaxSlots:            s.maxSlots,
		SlotMaxBytes:        s.slotMaxBytes,
		SlotTTLSeconds:      s.ttlSeconds,
		MaxTotalBytesBudget: s.maxSlots * s.slotMaxBytes,
	}
}

// AddProfileData appends serialized profile data to the slot for sourceKey if it exists.
func (s *ProfileStore) AddProfileData(sourceKey string, chunk []byte) {
	s.mu.RLock()
	slot, ok := s.slots[sourceKey]
	var buf *BoundedBuffer
	if ok && slot != nil {
		buf = slot.Buffer
	}
	s.mu.RUnlock()
	if buf == nil {
		return
	}
	if !buf.Add(chunk) {
		commonlogger.LoggerCompat().With("subsystem", "backend-profiling").Warn(
			"profile_chunk_dropped_oversized", "sourceKey", sourceKey,
		)
	}
}

// GetProfileData returns a shallow snapshot of buffered chunks for the given source key (see BoundedBuffer.Snapshot).
func (s *ProfileStore) GetProfileData(sourceKey string) [][]byte {
	s.mu.Lock()
	slot, ok := s.slots[sourceKey]
	if ok {
		slot.LastRequestAt = time.Now()
	}
	s.mu.Unlock()
	if !ok {
		return nil
	}
	return slot.Buffer.Snapshot()
}

func (s *ProfileStore) IsActive(sourceKey string) bool {
	s.mu.RLock()
	_, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	return ok
}

// ActiveSlots returns source keys for all open slots and the subset that have buffered data.
func (s *ProfileStore) ActiveSlots() (activeKeys []string, keysWithData []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for k, slot := range s.slots {
		activeKeys = append(activeKeys, k)
		if slot.Buffer != nil && slot.Buffer.Size() > 0 {
			keysWithData = append(keysWithData, k)
		}
	}
	return activeKeys, keysWithData
}

// RunCleanup is used for ttlSeconds based background goroutine for store slots cleanup.
func (s *ProfileStore) RunCleanup(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.stopCleanup = cancel
	go func() {
		ticker := time.NewTicker(s.cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.cleanupExpired()
			}
		}
	}()
}

func (s *ProfileStore) cleanupExpired() {
	cutoff := time.Now().Add(-time.Duration(s.ttlSeconds) * time.Second)
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, slot := range s.slots {
		if slot.LastRequestAt.Before(cutoff) {
			delete(s.slots, k)
		}
	}
}

// StopCleanup stops the TTL cleanup goroutine
func (s *ProfileStore) StopCleanup() {
	if s.stopCleanup != nil {
		s.stopCleanup()
	}
}
