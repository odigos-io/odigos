package collectorprofiles

import (
	"context"
	"sync"
	"time"
)

// Store limits default to defaults.go; Helm profiling.ui / PROFILES_* env override at runtime.

// Slot holds profile data for one source and last-request time for TTL.
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
	// stopCleanup is the cancel func from RunCleanup's derived context; StopCleanup invokes it to end the TTL goroutine.
	stopCleanup func()
}

// NewProfileStore creates a store with the given limits.
// maxSlots, ttlSeconds, slotMaxBytes use defaults if <= 0. cleanupInterval uses default if <= 0.
func NewProfileStore(maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) *ProfileStore {
	if maxSlots <= 0 {
		maxSlots = DefaultProfilingMaxSlots
	}
	if ttlSeconds <= 0 {
		ttlSeconds = DefaultProfilingSlotTTLSeconds
	}
	if slotMaxBytes <= 0 {
		slotMaxBytes = DefaultProfilingSlotMaxBytes
	}
	if cleanupInterval <= 0 {
		cleanupInterval = time.Duration(DefaultProfilingCleanupIntervalSeconds) * time.Second
	}
	s := &ProfileStore{
		slots:           make(map[string]*Slot),
		maxSlots:        maxSlots,
		ttlSeconds:      ttlSeconds,
		slotMaxBytes:    slotMaxBytes,
		cleanupInterval: cleanupInterval,
	}
	return s
}

// StartViewing ensures a slot exists for the given source key and refreshes LastRequestAt.
// If the store is full, the slot with the oldest LastRequestAt is evicted first.
func (s *ProfileStore) StartViewing(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	if slot, ok := s.slots[sourceKey]; ok {
		slot.LastRequestAt = now
		return
	}

	if len(s.slots) >= s.maxSlots {
		var oldestKey string
		var oldestTime time.Time
		first := true
		for k, slot := range s.slots {
			if first || slot.LastRequestAt.Before(oldestTime) {
				oldestTime = slot.LastRequestAt
				oldestKey = k
				first = false
			}
		}
		if oldestKey != "" {
			delete(s.slots, oldestKey)
		}
	}

	s.slots[sourceKey] = &Slot{
		LastRequestAt: now,
		Buffer:        NewBoundedBuffer(s.slotMaxBytes),
	}
}

// RemoveSlot deletes the slot and buffered OTLP data for sourceKey (e.g. user closed the profiling UI).
// When no slots remain, profile buffer memory returns to ~zero aside from store metadata.
func (s *ProfileStore) RemoveSlot(sourceKey string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.slots, sourceKey)
}

// MaxSlots returns the maximum number of concurrent profiling slots (services).
func (s *ProfileStore) MaxSlots() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSlots
}

// MemoryStats returns total bytes buffered across slots and the configured limits (for UI / debugging).
func (s *ProfileStore) MemoryStats() (totalBytes int, maxSlots int, slotMaxBytes int, ttlSeconds int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, slot := range s.slots {
		if slot.Buffer != nil {
			totalBytes += slot.Buffer.Size()
		}
	}
	return totalBytes, s.maxSlots, s.slotMaxBytes, s.ttlSeconds
}

// AddProfileData appends serialized profile data to the slot for sourceKey if it exists.
// No-op if the source is not in the active set.
// Copies the buffer pointer under the store lock so we do not append after eviction removes
// this key from the map (writes go to the same BoundedBuffer the slot used until TTL/GC).
func (s *ProfileStore) AddProfileData(sourceKey string, chunk []byte) {
	s.mu.Lock()
	slot, ok := s.slots[sourceKey]
	var buf *BoundedBuffer
	if ok && slot != nil {
		buf = slot.Buffer
	}
	s.mu.Unlock()
	if buf == nil {
		return
	}
	buf.Add(chunk)
}

// GetProfileData returns a snapshot of the buffer for the given source key.
// Returns nil if the source has no slot.
// Holds a single write lock for the full operation to prevent the cleanup
// goroutine from evicting the slot between the existence check and the refresh.
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

// IsActive returns true if the source has a slot (and is within TTL if cleanup has run).
func (s *ProfileStore) IsActive(sourceKey string) bool {
	s.mu.RLock()
	_, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	return ok
}

// DebugSlots returns active source keys and which have non-empty buffers (for debugging).
func (s *ProfileStore) DebugSlots() (activeKeys []string, keysWithData []string) {
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

// RunCleanup starts a background goroutine that removes slots not requested in the last ttlSeconds.
// Call the returned cancel to stop.
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

// StopCleanup stops the TTL cleanup goroutine if it was started.
func (s *ProfileStore) StopCleanup() {
	if s.stopCleanup != nil {
		s.stopCleanup()
	}
}
