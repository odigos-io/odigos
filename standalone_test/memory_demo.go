package main

import (
	"fmt"
	"log"
)

// Simulate the memory calculation logic from the implementation
type MemoryStats struct {
	TotalLimit            uint64
	TotalUsage            uint64
	EBPFUsage            uint64
	OtherNonHeap         uint64
	AvailableHeap        uint64
	RecommendedGOMEMLIMIT uint64
}

// simulateMemoryCalculation demonstrates the memory calculation formula
func simulateMemoryCalculation(k8sLimit, currentUsage, ebpfUsage, cache uint64) *MemoryStats {
	// Calculate available memory for Go heap
	// Reserve some memory for: eBPF objects + kernel memory + buffers + safety margin
	safetyMargin := k8sLimit / 20 // 5% safety margin
	nonHeapMemory := ebpfUsage + cache + safetyMargin
	
	availableHeap := uint64(0)
	if k8sLimit > nonHeapMemory {
		availableHeap = k8sLimit - nonHeapMemory
	}

	// Set GOMEMLIMIT to 80% of available heap memory to trigger GC appropriately
	recommendedGOMEMLIMIT := (availableHeap * 80) / 100

	return &MemoryStats{
		TotalLimit:            k8sLimit,
		TotalUsage:            currentUsage,
		EBPFUsage:            ebpfUsage,
		OtherNonHeap:         nonHeapMemory - ebpfUsage,
		AvailableHeap:        availableHeap,
		RecommendedGOMEMLIMIT: recommendedGOMEMLIMIT,
	}
}

func main() {
	fmt.Println("=== Odiglet Dynamic GOMEMLIMIT Calculation Test ===")
	
	testCases := []struct {
		name        string
		k8sLimit    uint64 // MB
		currentUsage uint64 // MB
		ebpfUsage   uint64 // MB
		cache       uint64 // MB
	}{
		{
			name:        "Small container (512MB)",
			k8sLimit:    512,
			currentUsage: 256,
			ebpfUsage:   16,
			cache:       32,
		},
		{
			name:        "Medium container (1GB)",
			k8sLimit:    1024,
			currentUsage: 512,
			ebpfUsage:   32,
			cache:       64,
		},
		{
			name:        "Large container (2GB)",
			k8sLimit:    2048,
			currentUsage: 1024,
			ebpfUsage:   64,
			cache:       128,
		},
		{
			name:        "Memory-constrained scenario",
			k8sLimit:    256,
			currentUsage: 200,
			ebpfUsage:   32,
			cache:       16,
		},
	}

	for i, tc := range testCases {
		fmt.Printf("\n--- Test Case %d: %s ---\n", i+1, tc.name)
		
		// Convert MB to bytes for calculation
		k8sLimitBytes := tc.k8sLimit * 1024 * 1024
		currentUsageBytes := tc.currentUsage * 1024 * 1024
		ebpfUsageBytes := tc.ebpfUsage * 1024 * 1024
		cacheBytes := tc.cache * 1024 * 1024
		
		// Calculate memory statistics
		stats := simulateMemoryCalculation(k8sLimitBytes, currentUsageBytes, ebpfUsageBytes, cacheBytes)
		
		// Display results
		fmt.Printf("Input:\n")
		fmt.Printf("  Kubernetes Memory Limit: %d MB\n", tc.k8sLimit)
		fmt.Printf("  Current Memory Usage: %d MB\n", tc.currentUsage)
		fmt.Printf("  eBPF Memory Usage: %d MB\n", tc.ebpfUsage)
		fmt.Printf("  Cache Memory: %d MB\n", tc.cache)
		
		fmt.Printf("\nCalculation:\n")
		fmt.Printf("  Safety Margin (5%%): %d MB\n", (stats.TotalLimit/20)/(1024*1024))
		fmt.Printf("  Total Non-Heap Memory: %d MB\n", (stats.EBPFUsage+stats.OtherNonHeap)/(1024*1024))
		fmt.Printf("  Available for Go Heap: %d MB\n", stats.AvailableHeap/(1024*1024))
		
		fmt.Printf("\nResult:\n")
		fmt.Printf("  OLD GOMEMLIMIT (80%% of limit): %d MB\n", (stats.TotalLimit*80/100)/(1024*1024))
		fmt.Printf("  NEW GOMEMLIMIT (80%% of available): %d MB\n", stats.RecommendedGOMEMLIMIT/(1024*1024))
		
		// Calculate improvement
		oldGOMEMLIMIT := (stats.TotalLimit * 80) / 100
		improvement := float64(oldGOMEMLIMIT-stats.RecommendedGOMEMLIMIT) / float64(oldGOMEMLIMIT) * 100
		
		fmt.Printf("  Memory saved for eBPF/cache: %.1f%% (%d MB)\n", 
			improvement, 
			(oldGOMEMLIMIT-stats.RecommendedGOMEMLIMIT)/(1024*1024))
		
		// Validate the calculation makes sense
		if stats.RecommendedGOMEMLIMIT > stats.TotalLimit {
			log.Printf("WARNING: GOMEMLIMIT exceeds container limit!")
		}
		
		utilizationRatio := float64(stats.RecommendedGOMEMLIMIT) / float64(stats.TotalLimit)
		fmt.Printf("  Memory utilization ratio: %.2f\n", utilizationRatio)
		
		if utilizationRatio < 0.4 {
			fmt.Printf("  ⚠️  Low ratio - might be over-conservative\n")
		} else if utilizationRatio > 0.8 {
			fmt.Printf("  ⚠️  High ratio - might be risky\n")
		} else {
			fmt.Printf("  ✅ Good ratio for memory safety\n")
		}
	}
	
	fmt.Println("\n=== Benefits of Dynamic GOMEMLIMIT ===")
	fmt.Println("1. Prevents OOM by accounting for eBPF and non-heap memory")
	fmt.Println("2. Triggers GC at appropriate times based on actual available heap")
	fmt.Println("3. Automatically adapts to changing memory usage patterns")
	fmt.Println("4. Provides safety margins while maximizing memory utilization")
	
	fmt.Println("\n=== Test completed successfully ===")
}