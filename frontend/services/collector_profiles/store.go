package collectorprofiles

import (
	"context"
	"log"
	"sync"
	"time"
)

const (
	// defaultMaxSlots: max number of services that can have profiling enabled at once (configurable via PROFILES_MAX_SLOTS).
	defaultMaxSlots           = 10
	defaultSlotTTLSeconds     = 30
	defaultCleanupInt         = 15 * time.Second
)

// Slot holds profile data for one source and last-request time for TTL.
type Slot struct {
	LastRequestAt time.Time
	Buffer        *BoundedBuffer
}

// ProfileStore holds at most maxSlots source-keyed slots with a TTL.
// Eviction: when full, the slot with the oldest LastRequestAt is removed.
// TTL: slots with no request in the last ttlSeconds are removed by a background goroutine.
type ProfileStore struct {
	mu               sync.RWMutex
	slots            map[string]*Slot
	maxSlots         int
	ttlSeconds       int
	slotMaxBytes     int
	cleanupInterval  time.Duration
	stopCleanup      func()
}

// NewProfileStore creates a store with the given limits.
// maxSlots, ttlSeconds, slotMaxBytes use defaults if <= 0. cleanupInterval uses default if <= 0.
func NewProfileStore(maxSlots, ttlSeconds, slotMaxBytes int, cleanupInterval time.Duration) *ProfileStore {
	if maxSlots <= 0 {
		maxSlots = defaultMaxSlots
	}
	if ttlSeconds <= 0 {
		ttlSeconds = defaultSlotTTLSeconds
	}
	if slotMaxBytes <= 0 {
		slotMaxBytes = defaultSlotMaxBytes
	}
	if cleanupInterval <= 0 {
		cleanupInterval = defaultCleanupInt
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
		profilingDebugLog("[profiling] store: refresh slot sourceKey=%q", sourceKey)
		return
	}
	log.Printf("[profiling] store: new slot sourceKey=%q activeSlots=%d", sourceKey, len(s.slots)+1)
	profilingDebugLog("[profiling] store: new slot sourceKey=%q (active=%d)", sourceKey, len(s.slots)+1)

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

// MaxSlots returns the maximum number of concurrent profiling slots (services).
func (s *ProfileStore) MaxSlots() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.maxSlots
}

// AddProfileData appends serialized profile data to the slot for sourceKey if it exists.
// No-op if the source is not in the active set.
func (s *ProfileStore) AddProfileData(sourceKey string, chunk []byte) {
	s.mu.RLock()
	slot, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	if !ok {
		return
	}
	slot.Buffer.Add(chunk)
}

// GetProfileData returns a snapshot of the buffer for the given source key.
// Returns nil if the source has no slot.
func (s *ProfileStore) GetProfileData(sourceKey string) [][]byte {
	s.mu.RLock()
	slot, ok := s.slots[sourceKey]
	s.mu.RUnlock()
	if !ok {
		return nil
	}
	// Refresh last request when reading (viewer is active)
	s.mu.Lock()
	slot.LastRequestAt = time.Now()
	s.mu.Unlock()
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
			profilingDebugLog("[profiling] store: evicted slot sourceKey=%q (TTL)", k)
		}
	}
}

// StopCleanup stops the TTL cleanup goroutine if it was started.
func (s *ProfileStore) StopCleanup() {
	if s.stopCleanup != nil {
		s.stopCleanup()
	}
}
