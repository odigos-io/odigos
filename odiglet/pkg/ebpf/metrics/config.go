package metrics

import (
	"time"
)

// EBPFMetricsConfig provides configuration for eBPF metrics collection
type EBPFMetricsConfig struct {
	// Memory Management
	MaxMemoryBytes        int64  `json:"max_memory_bytes" yaml:"max_memory_bytes"`                // Maximum memory for metrics collection
	MaxTrackedMaps        int    `json:"max_tracked_maps" yaml:"max_tracked_maps"`               // Maximum number of maps to track
	MaxTrackedProgs       int    `json:"max_tracked_progs" yaml:"max_tracked_progs"`             // Maximum number of programs to track
	MaxTrackedLinks       int    `json:"max_tracked_links" yaml:"max_tracked_links"`             // Maximum number of links to track
	
	// Performance Tuning
	CollectionInterval    time.Duration `json:"collection_interval" yaml:"collection_interval"`     // How often to collect metrics
	MaxBatchSize         int    `json:"max_batch_size" yaml:"max_batch_size"`                    // Batch size for syscall processing
	MaxConcurrentSyscalls int    `json:"max_concurrent_syscalls" yaml:"max_concurrent_syscalls"` // Limit concurrent syscalls
	
	// Feature Toggles
	EnableSystemStats     bool   `json:"enable_system_stats" yaml:"enable_system_stats"`         // Enable system-level statistics
	EnableDetailedMetrics bool   `json:"enable_detailed_metrics" yaml:"enable_detailed_metrics"` // Enable per-object detailed metrics
	EnablePinnedPaths     bool   `json:"enable_pinned_paths" yaml:"enable_pinned_paths"`         // Enable pinned path discovery
	
	// Rate Limiting
	MaxSyscallsPerSecond  int    `json:"max_syscalls_per_second" yaml:"max_syscalls_per_second"` // Rate limit for syscalls
	SyscallBurstLimit     int    `json:"syscall_burst_limit" yaml:"syscall_burst_limit"`         // Burst limit for syscalls
}

// DefaultEBPFMetricsConfig returns a configuration optimized for production use
func DefaultEBPFMetricsConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		// Conservative memory limits for production
		MaxMemoryBytes:        10 * 1024 * 1024, // 10MB maximum
		MaxTrackedMaps:        500,               // Track up to 500 maps
		MaxTrackedProgs:       200,               // Track up to 200 programs
		MaxTrackedLinks:       100,               // Track up to 100 links
		
		// Performance optimized for production workloads
		CollectionInterval:    60 * time.Second,  // Collect every minute
		MaxBatchSize:         50,                // Process 50 objects per batch
		MaxConcurrentSyscalls: 3,                // Limit concurrent syscalls
		
		// Minimal features for production efficiency
		EnableSystemStats:     false,            // Disabled for performance
		EnableDetailedMetrics: false,            // High-level aggregates only
		EnablePinnedPaths:     false,            // Skip expensive path discovery
		
		// Conservative syscall rate limiting
		MaxSyscallsPerSecond:  100,              // Max 100 syscalls/second
		SyscallBurstLimit:     20,               // Allow bursts of 20
	}
}

// HighPerformanceConfig returns a configuration optimized for minimal overhead
func HighPerformanceConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		MaxMemoryBytes:        5 * 1024 * 1024,  // 5MB maximum (very conservative)
		MaxTrackedMaps:        100,              // Fewer tracked objects
		MaxTrackedProgs:       50,               
		MaxTrackedLinks:       25,               
		
		CollectionInterval:    120 * time.Second, // Collect every 2 minutes
		MaxBatchSize:         100,               // Larger batches for efficiency
		MaxConcurrentSyscalls: 2,                // Fewer concurrent syscalls
		
		EnableSystemStats:     false,            
		EnableDetailedMetrics: false,            
		EnablePinnedPaths:     false,            
		
		MaxSyscallsPerSecond:  50,               // Very conservative rate limit
		SyscallBurstLimit:     10,               
	}
}

// DevelopmentConfig returns a configuration suitable for development/debugging
func DevelopmentConfig() *EBPFMetricsConfig {
	return &EBPFMetricsConfig{
		MaxMemoryBytes:        50 * 1024 * 1024, // 50MB for development
		MaxTrackedMaps:        1000,             
		MaxTrackedProgs:       500,              
		MaxTrackedLinks:       200,              
		
		CollectionInterval:    10 * time.Second, // More frequent collection
		MaxBatchSize:         25,                // Smaller batches for testing
		MaxConcurrentSyscalls: 5,                
		
		EnableSystemStats:     true,             // Enable all features
		EnableDetailedMetrics: true,             
		EnablePinnedPaths:     true,             
		
		MaxSyscallsPerSecond:  200,              
		SyscallBurstLimit:     50,               
	}
}

// Validate checks if the configuration is valid and applies reasonable defaults
func (c *EBPFMetricsConfig) Validate() error {
	// Apply minimum memory limit
	if c.MaxMemoryBytes < 1*1024*1024 {
		c.MaxMemoryBytes = 1 * 1024 * 1024 // Minimum 1MB
	}
	
	// Apply minimum collection interval
	if c.CollectionInterval < 10*time.Second {
		c.CollectionInterval = 10 * time.Second
	}
	
	// Apply reasonable batch size limits
	if c.MaxBatchSize < 10 {
		c.MaxBatchSize = 10
	}
	if c.MaxBatchSize > 200 {
		c.MaxBatchSize = 200
	}
	
	// Apply reasonable concurrent syscall limits
	if c.MaxConcurrentSyscalls < 1 {
		c.MaxConcurrentSyscalls = 1
	}
	if c.MaxConcurrentSyscalls > 10 {
		c.MaxConcurrentSyscalls = 10
	}
	
	// Apply reasonable tracking limits
	if c.MaxTrackedMaps < 10 {
		c.MaxTrackedMaps = 10
	}
	if c.MaxTrackedProgs < 5 {
		c.MaxTrackedProgs = 5
	}
	if c.MaxTrackedLinks < 5 {
		c.MaxTrackedLinks = 5
	}
	
	// Apply reasonable syscall rate limits
	if c.MaxSyscallsPerSecond < 10 {
		c.MaxSyscallsPerSecond = 10
	}
	if c.SyscallBurstLimit < 5 {
		c.SyscallBurstLimit = 5
	}
	
	return nil
}

// GetMemoryUtilizationPercent returns current memory utilization as a percentage
func (c *EBPFMetricsConfig) GetMemoryUtilizationPercent(currentUsage int64) float64 {
	if c.MaxMemoryBytes <= 0 {
		return 0
	}
	return float64(currentUsage) / float64(c.MaxMemoryBytes) * 100.0
}

// ShouldThrottleCollection returns true if collection should be throttled based on memory usage
func (c *EBPFMetricsConfig) ShouldThrottleCollection(currentUsage int64) bool {
	return c.GetMemoryUtilizationPercent(currentUsage) > 90.0 // Throttle at 90%
}

// EstimatedObjectMemoryUsage returns estimated memory usage for tracking N objects
func (c *EBPFMetricsConfig) EstimatedObjectMemoryUsage() int64 {
	const (
		mapInfoSize  = 200  // Estimated bytes per map info
		progInfoSize = 150  // Estimated bytes per prog info
		linkInfoSize = 100  // Estimated bytes per link info
	)
	
	return int64(c.MaxTrackedMaps*mapInfoSize + 
		c.MaxTrackedProgs*progInfoSize + 
		c.MaxTrackedLinks*linkInfoSize)
}