package ebpf

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/go-logr/logr"
)

// MemoryManager tracks non-heap memory usage and adjusts GOMEMLIMIT accordingly
type MemoryManager struct {
	logger          logr.Logger
	mu              sync.RWMutex
	ebpfMemoryUsage uint64
	lastUpdate      time.Time
	cgroupMemPath   string
	stopCh          chan struct{}
}

// MemoryStats contains memory usage information
type MemoryStats struct {
	TotalLimit     uint64 // Kubernetes memory limit
	TotalUsage     uint64 // Current total memory usage from cgroup
	EBPFUsage      uint64 // eBPF objects memory usage
	OtherNonHeap   uint64 // Other non-heap memory (calculated)
	AvailableHeap  uint64 // Available memory for Go heap
	RecommendedGOMEMLIMIT uint64 // Recommended GOMEMLIMIT value
}

// NewMemoryManager creates a new memory manager instance
func NewMemoryManager(logger logr.Logger) (*MemoryManager, error) {
	cgroupPath, err := findMemoryCgroupPath()
	if err != nil {
		return nil, fmt.Errorf("failed to find memory cgroup path: %w", err)
	}

	mm := &MemoryManager{
		logger:        logger.WithName("memory-manager"),
		cgroupMemPath: cgroupPath,
		stopCh:        make(chan struct{}),
	}

	return mm, nil
}

// Start begins the memory monitoring and GOMEMLIMIT adjustment process
func (mm *MemoryManager) Start() error {
	mm.logger.Info("Starting memory manager")

	// Get initial memory stats
	stats, err := mm.GetMemoryStats()
	if err != nil {
		return fmt.Errorf("failed to get initial memory stats: %w", err)
	}

	mm.logger.Info("Initial memory stats", 
		"totalLimit", stats.TotalLimit,
		"currentUsage", stats.TotalUsage,
		"ebpfUsage", stats.EBPFUsage,
		"recommendedGOMEMLIMIT", stats.RecommendedGOMEMLIMIT)

	// Set initial GOMEMLIMIT using automemlimit with custom calculation
	err = mm.setDynamicGOMEMLIMIT(stats)
	if err != nil {
		mm.logger.Error(err, "Failed to set initial GOMEMLIMIT")
	}

	// Start periodic monitoring
	go mm.monitoringLoop()

	return nil
}

// Stop stops the memory manager
func (mm *MemoryManager) Stop() {
	mm.logger.Info("Stopping memory manager")
	close(mm.stopCh)
}

// TrackEBPFMemory updates the tracked eBPF memory usage
func (mm *MemoryManager) TrackEBPFMemory(bytesAllocated uint64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	
	oldUsage := mm.ebpfMemoryUsage
	mm.ebpfMemoryUsage = bytesAllocated
	mm.lastUpdate = time.Now()
	
	if oldUsage != bytesAllocated {
		mm.logger.V(1).Info("eBPF memory usage updated",
			"oldUsage", oldUsage,
			"newUsage", bytesAllocated,
			"delta", int64(bytesAllocated)-int64(oldUsage))
	}
}

// GetMemoryStats returns current memory statistics
func (mm *MemoryManager) GetMemoryStats() (*MemoryStats, error) {
	// Read memory limit from cgroup
	limit, err := mm.readMemoryLimit()
	if err != nil {
		return nil, fmt.Errorf("failed to read memory limit: %w", err)
	}

	// Read current memory usage from cgroup
	usage, err := mm.readMemoryUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to read memory usage: %w", err)
	}

	mm.mu.RLock()
	ebpfUsage := mm.ebpfMemoryUsage
	mm.mu.RUnlock()

	// Calculate other non-heap memory (total usage - rss - cache - ebpf)
	rss, cache, err := mm.readMemoryBreakdown()
	if err != nil {
		mm.logger.Error(err, "Failed to read memory breakdown, using estimation")
		// Use 20% of total usage as estimation for non-heap overhead
		cache = usage / 5
		rss = usage - cache - ebpfUsage
	}

	// Calculate available memory for Go heap
	// Reserve some memory for: eBPF objects + kernel memory + buffers + safety margin
	nonHeapMemory := ebpfUsage + cache + (limit / 20) // eBPF + cache + 5% safety margin
	availableHeap := uint64(0)
	if limit > nonHeapMemory {
		availableHeap = limit - nonHeapMemory
	}

	// Set GOMEMLIMIT to 80% of available heap memory to trigger GC appropriately
	recommendedGOMEMLIMIT := (availableHeap * 80) / 100

	stats := &MemoryStats{
		TotalLimit:            limit,
		TotalUsage:            usage,
		EBPFUsage:            ebpfUsage,
		OtherNonHeap:         nonHeapMemory - ebpfUsage,
		AvailableHeap:        availableHeap,
		RecommendedGOMEMLIMIT: recommendedGOMEMLIMIT,
	}

	return stats, nil
}

// monitoringLoop periodically updates GOMEMLIMIT based on memory usage
func (mm *MemoryManager) monitoringLoop() {
	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			stats, err := mm.GetMemoryStats()
			if err != nil {
				mm.logger.Error(err, "Failed to get memory stats during monitoring")
				continue
			}

			err = mm.setDynamicGOMEMLIMIT(stats)
			if err != nil {
				mm.logger.Error(err, "Failed to update GOMEMLIMIT")
			}

			// Log stats periodically (every 5 minutes)
			if time.Now().Unix()%300 == 0 {
				mm.logger.Info("Memory usage stats",
					"totalUsage", stats.TotalUsage,
					"totalLimit", stats.TotalLimit,
					"ebpfUsage", stats.EBPFUsage,
					"currentGOMEMLIMIT", os.Getenv("GOMEMLIMIT"),
					"recommendedGOMEMLIMIT", stats.RecommendedGOMEMLIMIT,
					"utilizationPercent", (stats.TotalUsage*100)/stats.TotalLimit)
			}

		case <-mm.stopCh:
			return
		}
	}
}

// setDynamicGOMEMLIMIT updates GOMEMLIMIT using automemlimit with custom calculation
func (mm *MemoryManager) setDynamicGOMEMLIMIT(stats *MemoryStats) error {
	if stats.RecommendedGOMEMLIMIT == 0 {
		mm.logger.V(1).Info("Recommended GOMEMLIMIT is 0, skipping update")
		return nil
	}

	// Create a custom provider that returns our calculated limit
	customProvider := func() (uint64, error) {
		return stats.RecommendedGOMEMLIMIT, nil
	}

	// Use automemlimit with our custom provider
	return memlimit.SetGoMemLimitWithOpts(
		memlimit.WithRatio(1.0), // We already calculated the ratio
		memlimit.WithProvider(customProvider),
		memlimit.WithLogger(mm.logger),
	)
}

// findMemoryCgroupPath finds the memory cgroup path for the current process
func findMemoryCgroupPath() (string, error) {
	// Read /proc/self/cgroup to find memory cgroup
	file, err := os.Open("/proc/self/cgroup")
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			// Check if this is memory cgroup (v1) or unified cgroup (v2)
			if strings.Contains(parts[1], "memory") || parts[1] == "" {
				cgroupPath := parts[2]
				// For cgroup v1
				if strings.Contains(parts[1], "memory") {
					return filepath.Join("/sys/fs/cgroup/memory", cgroupPath), nil
				}
				// For cgroup v2 (unified)
				if parts[1] == "" {
					return filepath.Join("/sys/fs/cgroup", cgroupPath), nil
				}
			}
		}
	}

	return "", fmt.Errorf("memory cgroup not found in /proc/self/cgroup")
}

// readMemoryLimit reads the memory limit from cgroup
func (mm *MemoryManager) readMemoryLimit() (uint64, error) {
	// Try cgroup v2 first
	if val, err := mm.readUint64File(filepath.Join(mm.cgroupMemPath, "memory.max")); err == nil {
		if val == 9223372036854775807 { // max value indicates no limit
			// Fall back to system memory
			if sysVal, err := mm.readSystemMemory(); err == nil {
				return sysVal, nil
			}
		}
		return val, nil
	}

	// Fall back to cgroup v1
	if val, err := mm.readUint64File(filepath.Join(mm.cgroupMemPath, "memory.limit_in_bytes")); err == nil {
		if val == 9223372036854775807 { // max value indicates no limit
			// Fall back to system memory
			if sysVal, err := mm.readSystemMemory(); err == nil {
				return sysVal, nil
			}
		}
		return val, nil
	}

	return 0, fmt.Errorf("could not read memory limit from cgroup")
}

// readMemoryUsage reads current memory usage from cgroup
func (mm *MemoryManager) readMemoryUsage() (uint64, error) {
	// Try cgroup v2 first
	if val, err := mm.readUint64File(filepath.Join(mm.cgroupMemPath, "memory.current")); err == nil {
		return val, nil
	}

	// Fall back to cgroup v1
	if val, err := mm.readUint64File(filepath.Join(mm.cgroupMemPath, "memory.usage_in_bytes")); err == nil {
		return val, nil
	}

	return 0, fmt.Errorf("could not read memory usage from cgroup")
}

// readMemoryBreakdown reads RSS and cache memory from cgroup
func (mm *MemoryManager) readMemoryBreakdown() (rss, cache uint64, err error) {
	// Try cgroup v2 first
	if data, err := os.ReadFile(filepath.Join(mm.cgroupMemPath, "memory.stat")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "anon ") {
				if val, parseErr := strconv.ParseUint(strings.Fields(line)[1], 10, 64); parseErr == nil {
					rss = val
				}
			}
			if strings.HasPrefix(line, "file ") {
				if val, parseErr := strconv.ParseUint(strings.Fields(line)[1], 10, 64); parseErr == nil {
					cache = val
				}
			}
		}
		if rss > 0 || cache > 0 {
			return rss, cache, nil
		}
	}

	// Fall back to cgroup v1 format
	if data, err := os.ReadFile(filepath.Join(mm.cgroupMemPath, "memory.stat")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if fields[0] == "rss" || fields[0] == "total_rss" {
					if val, parseErr := strconv.ParseUint(fields[1], 10, 64); parseErr == nil {
						rss = val
					}
				}
				if fields[0] == "cache" || fields[0] == "total_cache" {
					if val, parseErr := strconv.ParseUint(fields[1], 10, 64); parseErr == nil {
						cache = val
					}
				}
			}
		}
	}

	if rss == 0 && cache == 0 {
		return 0, 0, fmt.Errorf("could not read memory breakdown from cgroup")
	}

	return rss, cache, nil
}

// readSystemMemory reads total system memory as fallback
func (mm *MemoryManager) readSystemMemory() (uint64, error) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "MemTotal:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				val, err := strconv.ParseUint(fields[1], 10, 64)
				if err != nil {
					return 0, err
				}
				return val * 1024, nil // Convert from kB to bytes
			}
		}
	}

	return 0, fmt.Errorf("MemTotal not found in /proc/meminfo")
}

// readUint64File reads a uint64 value from a file
func (mm *MemoryManager) readUint64File(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	val, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return val, nil
}