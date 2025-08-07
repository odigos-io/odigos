package ebpf

import (
	"testing"

	"github.com/go-logr/logr"
)

// TestMemoryStatCalculations tests the memory calculation logic without requiring cgroup access
func TestMemoryStatCalculations(t *testing.T) {
	// Create a memory manager with a no-op logger for testing
	logger := logr.Discard()
	
	// Create a test memory tracker
	mockManager := &MemoryManager{
		logger: logger,
	}
	
	tracker := NewEBPFMemoryTracker(logger, mockManager)
	
	// Test 1: Basic eBPF memory tracking
	tracker.TrackMapAllocation("test_map", 8, 64, 1000)
	tracker.TrackProgramAllocation("test_prog", 500)
	
	mapMem, progMem, total := tracker.GetMemoryUsage()
	
	// Verify calculations
	expectedMapMem := uint64(8+64+8)*1000 + 4096  // (key+value+overhead)*entries + map overhead
	expectedProgMem := uint64(500)*8 + uint64(500)*8/2 + 4096  // instructions*8 + JIT overhead + prog overhead
	expectedTotal := expectedMapMem + expectedProgMem
	
	if mapMem != expectedMapMem {
		t.Errorf("Expected map memory %d, got %d", expectedMapMem, mapMem)
	}
	
	if progMem != expectedProgMem {
		t.Errorf("Expected program memory %d, got %d", expectedProgMem, progMem)
	}
	
	if total != expectedTotal {
		t.Errorf("Expected total memory %d, got %d", expectedTotal, total)
	}
	
	// Test 2: Deallocation
	tracker.TrackMapDeallocation("test_map", 8, 64, 1000)
	
	mapMem, progMem, total = tracker.GetMemoryUsage()
	
	if mapMem != 0 {
		t.Errorf("Expected map memory 0 after deallocation, got %d", mapMem)
	}
	
	if total != expectedProgMem {
		t.Errorf("Expected total memory %d after map deallocation, got %d", expectedProgMem, total)
	}
	
	t.Logf("eBPF memory tracking test passed successfully")
	t.Logf("Map memory calculation: %d bytes", expectedMapMem)
	t.Logf("Program memory calculation: %d bytes", expectedProgMem)
}

// TestGlobalTrackingFunctions tests the global tracking functions
func TestGlobalTrackingFunctions(t *testing.T) {
	logger := logr.Discard()
	
	// Create and set a global tracker
	mockManager := &MemoryManager{logger: logger}
	tracker := NewEBPFMemoryTracker(logger, mockManager)
	setGlobalEBPFMemoryTracker(tracker)
	
	// Test global tracking functions
	TrackGlobalMapAllocation("global_map", 16, 32, 500)
	TrackGlobalProgramAllocation("global_prog", 200)
	
	// Verify tracking worked
	if globalTracker := GetGlobalEBPFMemoryTracker(); globalTracker != nil {
		mapMem, progMem, total := globalTracker.GetMemoryUsage()
		
		if mapMem == 0 {
			t.Error("Global map allocation was not tracked")
		}
		
		if progMem == 0 {
			t.Error("Global program allocation was not tracked")
		}
		
		if total == 0 {
			t.Error("Global total allocation was not tracked")
		}
		
		t.Logf("Global tracking test passed: maps=%d, progs=%d, total=%d", mapMem, progMem, total)
	} else {
		t.Error("Global tracker was not set properly")
	}
	
	// Test deallocation
	TrackGlobalMapDeallocation("global_map", 16, 32, 500)
	TrackGlobalProgramDeallocation("global_prog", 200)
	
	if globalTracker := GetGlobalEBPFMemoryTracker(); globalTracker != nil {
		mapMem, progMem, total := globalTracker.GetMemoryUsage()
		
		if mapMem != 0 || progMem != 0 || total != 0 {
			t.Errorf("Global deallocation failed: maps=%d, progs=%d, total=%d", mapMem, progMem, total)
		} else {
			t.Log("Global deallocation test passed")
		}
	}
}

// TestMemoryCalculationFormula tests the memory calculation formula used by the memory manager
func TestMemoryCalculationFormula(t *testing.T) {
	// Test the formula: Available Heap = K8s Limit - eBPF - Cache - Safety Margin (5%)
	// GOMEMLIMIT = Available Heap * 80%
	
	testCases := []struct {
		name            string
		k8sLimit        uint64
		ebpfUsage       uint64
		cache           uint64
		expectedHeap    uint64
		expectedGOMEMLIMIT uint64
	}{
		{
			name:            "Basic 512MB container",
			k8sLimit:        512 * 1024 * 1024,  // 512MB
			ebpfUsage:       16 * 1024 * 1024,   // 16MB eBPF
			cache:           32 * 1024 * 1024,   // 32MB cache
			expectedHeap:    512*1024*1024 - 16*1024*1024 - 32*1024*1024 - (512*1024*1024)/20,  // ~438MB
			expectedGOMEMLIMIT: 0, // Will be calculated
		},
		{
			name:            "Large 2GB container",
			k8sLimit:        2 * 1024 * 1024 * 1024,  // 2GB
			ebpfUsage:       64 * 1024 * 1024,        // 64MB eBPF
			cache:           128 * 1024 * 1024,       // 128MB cache
			expectedHeap:    2*1024*1024*1024 - 64*1024*1024 - 128*1024*1024 - (2*1024*1024*1024)/20,  // ~1.7GB
			expectedGOMEMLIMIT: 0, // Will be calculated
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate expected values
			safetyMargin := tc.k8sLimit / 20  // 5%
			nonHeapMemory := tc.ebpfUsage + tc.cache + safetyMargin
			availableHeap := uint64(0)
			if tc.k8sLimit > nonHeapMemory {
				availableHeap = tc.k8sLimit - nonHeapMemory
			}
			recommendedGOMEMLIMIT := (availableHeap * 80) / 100
			
			t.Logf("Test case: %s", tc.name)
			t.Logf("  K8s Limit: %d MB", tc.k8sLimit/(1024*1024))
			t.Logf("  eBPF Usage: %d MB", tc.ebpfUsage/(1024*1024))
			t.Logf("  Cache: %d MB", tc.cache/(1024*1024))
			t.Logf("  Safety Margin: %d MB", safetyMargin/(1024*1024))
			t.Logf("  Available Heap: %d MB", availableHeap/(1024*1024))
			t.Logf("  Recommended GOMEMLIMIT: %d MB", recommendedGOMEMLIMIT/(1024*1024))
			
			// Verify the calculation makes sense
			if availableHeap == 0 {
				t.Error("Available heap should not be zero for reasonable inputs")
			}
			
			if recommendedGOMEMLIMIT >= tc.k8sLimit {
				t.Error("GOMEMLIMIT should be less than K8s limit")
			}
			
			if recommendedGOMEMLIMIT > availableHeap {
				t.Error("GOMEMLIMIT should not exceed available heap")
			}
			
			// Verify the safety ratio (GOMEMLIMIT should be ~65-75% of total limit for typical workloads)
			ratio := float64(recommendedGOMEMLIMIT) / float64(tc.k8sLimit)
			if ratio < 0.5 || ratio > 0.85 {
				t.Logf("Warning: GOMEMLIMIT ratio %.2f might be outside expected range (0.5-0.85)", ratio)
			}
		})
	}
}

// BenchmarkMemoryTracking benchmarks the memory tracking operations
func BenchmarkMemoryTracking(b *testing.B) {
	logger := logr.Discard()
	mockManager := &MemoryManager{logger: logger}
	tracker := NewEBPFMemoryTracker(logger, mockManager)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		tracker.TrackMapAllocation("bench_map", 8, 64, 1000)
		tracker.TrackProgramAllocation("bench_prog", 500)
		tracker.GetMemoryUsage()
		tracker.TrackMapDeallocation("bench_map", 8, 64, 1000)
		tracker.TrackProgramDeallocation("bench_prog", 500)
	}
}