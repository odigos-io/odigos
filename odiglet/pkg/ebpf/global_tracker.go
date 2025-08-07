package ebpf

import (
	"sync"
)

var (
	globalTracker *EBPFMemoryTracker
	globalMemoryManager *MemoryManager
	trackerMutex  sync.RWMutex
)

// setGlobalEBPFMemoryTracker sets the global eBPF memory tracker
func setGlobalEBPFMemoryTracker(tracker *EBPFMemoryTracker) {
	trackerMutex.Lock()
	defer trackerMutex.Unlock()
	globalTracker = tracker
}

// setGlobalMemoryManager sets the global memory manager
func setGlobalMemoryManager(manager *MemoryManager) {
	trackerMutex.Lock()
	defer trackerMutex.Unlock()
	globalMemoryManager = manager
}

// GetGlobalMemoryManager returns the global memory manager
func GetGlobalMemoryManager() *MemoryManager {
	trackerMutex.RLock()
	defer trackerMutex.RUnlock()
	return globalMemoryManager
}

// GetGlobalEBPFMemoryTracker returns the global eBPF memory tracker
func GetGlobalEBPFMemoryTracker() *EBPFMemoryTracker {
	trackerMutex.RLock()
	defer trackerMutex.RUnlock()
	return globalTracker
}

// TrackGlobalMapAllocation tracks eBPF map allocation globally
func TrackGlobalMapAllocation(mapName string, keySize, valueSize, maxEntries uint32) {
	trackerMutex.RLock()
	tracker := globalTracker
	trackerMutex.RUnlock()
	
	if tracker != nil {
		tracker.TrackMapAllocation(mapName, keySize, valueSize, maxEntries)
	}
}

// TrackGlobalMapDeallocation tracks eBPF map deallocation globally
func TrackGlobalMapDeallocation(mapName string, keySize, valueSize, maxEntries uint32) {
	trackerMutex.RLock()
	tracker := globalTracker
	trackerMutex.RUnlock()
	
	if tracker != nil {
		tracker.TrackMapDeallocation(mapName, keySize, valueSize, maxEntries)
	}
}

// TrackGlobalProgramAllocation tracks eBPF program allocation globally
func TrackGlobalProgramAllocation(progName string, instructionCount uint32) {
	trackerMutex.RLock()
	tracker := globalTracker
	trackerMutex.RUnlock()
	
	if tracker != nil {
		tracker.TrackProgramAllocation(progName, instructionCount)
	}
}

// TrackGlobalProgramDeallocation tracks eBPF program deallocation globally
func TrackGlobalProgramDeallocation(progName string, instructionCount uint32) {
	trackerMutex.RLock()
	tracker := globalTracker
	trackerMutex.RUnlock()
	
	if tracker != nil {
		tracker.TrackProgramDeallocation(progName, instructionCount)
	}
}