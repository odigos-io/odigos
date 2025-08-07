package ebpf

import (
	"fmt"
	"time"
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
)

// TestMemoryManager demonstrates the memory manager functionality
func TestMemoryManager(t *testing.T) {
	// Create a test logger
	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}
	logger := zapr.NewLogger(zapLogger)

	// Create memory manager
	mm, err := NewMemoryManager(logger)
	if err != nil {
		t.Skipf("Skipping test - could not create memory manager: %v", err)
	}

	// Create eBPF memory tracker
	tracker := NewEBPFMemoryTracker(logger, mm)

	// Simulate eBPF object allocations
	fmt.Println("=== Testing Memory Manager ===")

	// Test 1: Track some eBPF map allocations
	tracker.TrackMapAllocation("test_map_1", 8, 64, 1000)   // 8-byte key, 64-byte value, 1000 entries
	tracker.TrackMapAllocation("test_map_2", 16, 128, 2000) // 16-byte key, 128-byte value, 2000 entries

	// Test 2: Track eBPF program allocations
	tracker.TrackProgramAllocation("test_prog_1", 500)  // 500 instructions
	tracker.TrackProgramAllocation("test_prog_2", 1000) // 1000 instructions

	// Test 3: Get memory statistics
	stats, err := mm.GetMemoryStats()
	if err != nil {
		t.Errorf("Failed to get memory stats: %v", err)
		return
	}

	fmt.Printf("Memory Statistics:\n")
	fmt.Printf("  Total Limit: %d MB\n", stats.TotalLimit/(1024*1024))
	fmt.Printf("  Total Usage: %d MB\n", stats.TotalUsage/(1024*1024))
	fmt.Printf("  eBPF Usage: %d KB\n", stats.EBPFUsage/1024)
	fmt.Printf("  Other Non-Heap: %d KB\n", stats.OtherNonHeap/1024)
	fmt.Printf("  Available Heap: %d MB\n", stats.AvailableHeap/(1024*1024))
	fmt.Printf("  Recommended GOMEMLIMIT: %d MB\n", stats.RecommendedGOMEMLIMIT/(1024*1024))

	// Test 4: Simulate deallocation
	tracker.TrackMapDeallocation("test_map_1", 8, 64, 1000)
	tracker.TrackProgramDeallocation("test_prog_1", 500)

	// Get final memory usage
	mapMem, progMem, total := tracker.GetMemoryUsage()
	fmt.Printf("\nFinal eBPF Memory Usage:\n")
	fmt.Printf("  Maps: %d KB\n", mapMem/1024)
	fmt.Printf("  Programs: %d KB\n", progMem/1024)
	fmt.Printf("  Total: %d KB\n", total/1024)

	fmt.Println("=== Test completed successfully ===")
}

// ExampleMemoryManager shows how to use the memory manager
func ExampleMemoryManager() {
	// Create logger
	zapLogger, _ := zap.NewDevelopment()
	logger := zapr.NewLogger(zapLogger)

	// Create and start memory manager
	mm, err := NewMemoryManager(logger)
	if err != nil {
		fmt.Printf("Error creating memory manager: %v\n", err)
		return
	}

	err = mm.Start()
	if err != nil {
		fmt.Printf("Error starting memory manager: %v\n", err)
		return
	}
	defer mm.Stop()

	// Create eBPF memory tracker
	tracker := NewEBPFMemoryTracker(logger, mm)

	// Track eBPF allocations
	tracker.TrackMapAllocation("example_map", 8, 32, 1024)
	tracker.TrackProgramAllocation("example_prog", 256)

	// Get statistics
	stats, _ := mm.GetMemoryStats()
	fmt.Printf("GOMEMLIMIT should be set to: %d MB\n", stats.RecommendedGOMEMLIMIT/(1024*1024))

	// Simulate some time passing for the memory manager to do its work
	time.Sleep(1 * time.Second)
}