package ebpf

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// EBPFMemoryTracker tracks memory usage of eBPF objects
type EBPFMemoryTracker struct {
	logger            logr.Logger
	memoryManager     *MemoryManager
	mu                sync.RWMutex
	mapMemoryUsage    uint64
	progMemoryUsage   uint64
	totalAllocated    uint64
	lastUpdate        time.Time
}

// NewEBPFMemoryTracker creates a new eBPF memory tracker
func NewEBPFMemoryTracker(logger logr.Logger, memoryManager *MemoryManager) *EBPFMemoryTracker {
	return &EBPFMemoryTracker{
		logger:        logger.WithName("ebpf-memory-tracker"),
		memoryManager: memoryManager,
	}
}

// TrackMapAllocation tracks memory allocated for eBPF maps
func (t *EBPFMemoryTracker) TrackMapAllocation(mapName string, keySize, valueSize, maxEntries uint32) {
	// Calculate approximate memory usage for the map
	// This is a simplified calculation; actual memory usage may vary
	memoryPerEntry := uint64(keySize + valueSize + 8) // 8 bytes overhead per entry
	totalMapMemory := memoryPerEntry * uint64(maxEntries)
	
	// Add additional overhead for map metadata (approximate)
	mapOverhead := uint64(4096) // 4KB overhead for map structure
	totalMapMemory += mapOverhead

	t.mu.Lock()
	t.mapMemoryUsage += totalMapMemory
	t.updateTotalAndNotify()
	t.mu.Unlock()

	t.logger.V(1).Info("Tracked eBPF map allocation",
		"mapName", mapName,
		"keySize", keySize,
		"valueSize", valueSize,
		"maxEntries", maxEntries,
		"estimatedMemory", totalMapMemory,
		"totalMapMemory", t.mapMemoryUsage)
}

// TrackMapDeallocation tracks memory deallocated for eBPF maps
func (t *EBPFMemoryTracker) TrackMapDeallocation(mapName string, keySize, valueSize, maxEntries uint32) {
	memoryPerEntry := uint64(keySize + valueSize + 8)
	totalMapMemory := memoryPerEntry * uint64(maxEntries) + 4096

	t.mu.Lock()
	if t.mapMemoryUsage >= totalMapMemory {
		t.mapMemoryUsage -= totalMapMemory
	} else {
		t.mapMemoryUsage = 0
	}
	t.updateTotalAndNotify()
	t.mu.Unlock()

	t.logger.V(1).Info("Tracked eBPF map deallocation",
		"mapName", mapName,
		"estimatedMemory", totalMapMemory,
		"totalMapMemory", t.mapMemoryUsage)
}

// TrackProgramAllocation tracks memory allocated for eBPF programs
func (t *EBPFMemoryTracker) TrackProgramAllocation(progName string, instructionCount uint32) {
	// Each eBPF instruction is typically 8 bytes
	// Add overhead for program metadata and JIT compilation
	progMemory := uint64(instructionCount) * 8
	jitOverhead := progMemory / 2 // JIT compilation roughly doubles memory usage
	totalProgMemory := progMemory + jitOverhead + 4096 // 4KB overhead

	t.mu.Lock()
	t.progMemoryUsage += totalProgMemory
	t.updateTotalAndNotify()
	t.mu.Unlock()

	t.logger.V(1).Info("Tracked eBPF program allocation",
		"progName", progName,
		"instructionCount", instructionCount,
		"estimatedMemory", totalProgMemory,
		"totalProgMemory", t.progMemoryUsage)
}

// TrackProgramDeallocation tracks memory deallocated for eBPF programs
func (t *EBPFMemoryTracker) TrackProgramDeallocation(progName string, instructionCount uint32) {
	progMemory := uint64(instructionCount) * 8
	jitOverhead := progMemory / 2
	totalProgMemory := progMemory + jitOverhead + 4096

	t.mu.Lock()
	if t.progMemoryUsage >= totalProgMemory {
		t.progMemoryUsage -= totalProgMemory
	} else {
		t.progMemoryUsage = 0
	}
	t.updateTotalAndNotify()
	t.mu.Unlock()

	t.logger.V(1).Info("Tracked eBPF program deallocation",
		"progName", progName,
		"estimatedMemory", totalProgMemory,
		"totalProgMemory", t.progMemoryUsage)
}

// ReadEBPFMemoryUsageFromKernel attempts to read actual eBPF memory usage from kernel
func (t *EBPFMemoryTracker) ReadEBPFMemoryUsageFromKernel() uint64 {
	// Try to read from /proc/meminfo or /sys/kernel/debug/bpf if available
	if memUsage := t.readFromProcMeminfo(); memUsage > 0 {
		return memUsage
	}

	if memUsage := t.readFromBPFFS(); memUsage > 0 {
		return memUsage
	}

	// Fallback to our tracked usage
	t.mu.RLock()
	total := t.totalAllocated
	t.mu.RUnlock()

	return total
}

// GetMemoryUsage returns current eBPF memory usage
func (t *EBPFMemoryTracker) GetMemoryUsage() (mapMem, progMem, total uint64) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.mapMemoryUsage, t.progMemoryUsage, t.totalAllocated
}

// updateTotalAndNotify updates total memory usage and notifies memory manager
func (t *EBPFMemoryTracker) updateTotalAndNotify() {
	// This function assumes the mutex is already held
	oldTotal := t.totalAllocated
	t.totalAllocated = t.mapMemoryUsage + t.progMemoryUsage
	t.lastUpdate = time.Now()

	// Try to get more accurate reading from kernel
	if kernelUsage := t.ReadEBPFMemoryUsageFromKernel(); kernelUsage > t.totalAllocated {
		t.totalAllocated = kernelUsage
	}

	// Notify memory manager of the update
	if t.memoryManager != nil && oldTotal != t.totalAllocated {
		t.memoryManager.TrackEBPFMemory(t.totalAllocated)
	}
}

// readFromProcMeminfo tries to read eBPF memory usage from /proc/meminfo
func (t *EBPFMemoryTracker) readFromProcMeminfo() uint64 {
	// This is a placeholder - in practice, eBPF memory might not be directly
	// exposed in /proc/meminfo, but we can try to infer it from kernel memory usage
	return 0
}

// readFromBPFFS tries to read eBPF memory usage from /sys/fs/bpf or /sys/kernel/debug/bpf
func (t *EBPFMemoryTracker) readFromBPFFS() uint64 {
	// Try to read from debugfs bpf stats if available
	debugPath := "/sys/kernel/debug/bpf"
	if _, err := os.Stat(debugPath); err == nil {
		// Try to read memory statistics from debugfs
		return t.readBPFDebugStats(debugPath)
	}

	// Alternative: try to estimate from /proc/slabinfo for bpf-related slabs
	return t.readFromSlabinfo()
}

// readBPFDebugStats reads eBPF memory stats from debugfs
func (t *EBPFMemoryTracker) readBPFDebugStats(debugPath string) uint64 {
	// Read various eBPF-related files in debugfs
	var totalMemory uint64

	// Try to read from memory usage files if they exist
	files := []string{
		filepath.Join(debugPath, "memory"),
		filepath.Join(debugPath, "progs"),
		filepath.Join(debugPath, "maps"),
	}

	for _, file := range files {
		if data, err := os.ReadFile(file); err == nil {
			// Parse the file content for memory information
			if mem := t.parseMemoryFromDebugFile(string(data)); mem > 0 {
				totalMemory += mem
			}
		}
	}

	return totalMemory
}

// parseMemoryFromDebugFile parses memory information from debug files
func (t *EBPFMemoryTracker) parseMemoryFromDebugFile(content string) uint64 {
	lines := strings.Split(content, "\n")
	var totalMemory uint64

	for _, line := range lines {
		// Look for memory-related fields
		if strings.Contains(line, "memory") || strings.Contains(line, "size") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if (strings.Contains(field, "memory") || strings.Contains(field, "size")) && i+1 < len(fields) {
					if mem, err := strconv.ParseUint(fields[i+1], 10, 64); err == nil {
						totalMemory += mem
					}
				}
			}
		}
	}

	return totalMemory
}

// readFromSlabinfo tries to estimate eBPF memory usage from slab allocator statistics
func (t *EBPFMemoryTracker) readFromSlabinfo() uint64 {
	data, err := os.ReadFile("/proc/slabinfo")
	if err != nil {
		return 0
	}

	lines := strings.Split(string(data), "\n")
	var totalBPFMemory uint64

	for _, line := range lines {
		if strings.Contains(line, "bpf") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				// Extract active objects and object size
				if activeObjs, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					if objSize, err := strconv.ParseUint(fields[3], 10, 64); err == nil {
						totalBPFMemory += activeObjs * objSize
					}
				}
			}
		}
	}

	return totalBPFMemory
}