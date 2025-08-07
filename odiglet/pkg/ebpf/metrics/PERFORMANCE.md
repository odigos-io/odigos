# eBPF Metrics Collection - Performance Optimization Guide

## Overview

This document outlines the performance optimizations implemented in the eBPF metrics collection system to ensure extremely efficient operation in resource-constrained environments.

## Memory Management

### Strict Memory Limits

- **Fixed Memory Budget**: Configurable maximum memory allocation (default: 10MB, high-performance: 5MB)
- **Pre-allocated Object Pools**: Zero-allocation during runtime using fixed-size pools
- **Memory Exhaustion Protection**: Automatic collection suspension when approaching limits
- **Pool Utilization Metrics**: Track pool usage to prevent overflows

### Configuration Profiles

```go
// High-Performance Profile (Minimal Resource Usage)
MaxMemoryBytes:        5MB
MaxTrackedMaps:        100 objects  
MaxTrackedProgs:       50 objects
MaxTrackedLinks:       25 objects
CollectionInterval:    120 seconds
MaxConcurrentSyscalls: 2

// Production Profile (Balanced)
MaxMemoryBytes:        10MB
MaxTrackedMaps:        500 objects
MaxTrackedProgs:       200 objects  
MaxTrackedLinks:       100 objects
CollectionInterval:    60 seconds
MaxConcurrentSyscalls: 3
```

## Syscall Optimization

### Batched Processing

- **Enumeration Phase**: Single pass to collect all object IDs
- **Batch Processing**: Process objects in configurable batches (default: 50)
- **Rate Limiting**: Limit concurrent syscalls to prevent kernel pressure
- **Early Termination**: Stop processing when pools are exhausted

### Efficient Object Tracking

```go
// Before: Individual syscalls for each object
for each object {
    syscall(get_next_id)
    syscall(get_fd_by_id) 
    syscall(get_info_by_fd)
}

// After: Batched enumeration + processing
objectIDs := enumerate_all_ids()  // Single enumeration pass
process_in_batches(objectIDs, batch_size=50)
```

## Fast /metrics Endpoint Response

### Pre-computed Metrics Cache

- **Atomic Operations**: All metric updates use atomic operations for lock-free reads
- **Aggregated Values**: Pre-calculate totals during collection
- **No Real-time Syscalls**: /metrics endpoint never makes syscalls
- **Sub-millisecond Response**: Cached values provide instant responses

### Minimal Metric Set

Instead of detailed per-object metrics, expose only high-level aggregates:

```prometheus
# Essential metrics only
odigos.ebpf.objects.total{type="map"}
odigos.ebpf.objects.total{type="program"} 
odigos.ebpf.objects.total{type="link"}
odigos.ebpf.total_memory_bytes
odigos.ebpf.collection.memory_limit_hit_total
odigos.ebpf.collection.errors_total
```

## Memory Efficiency Techniques

### Data Structure Optimization

```go
// Before: Map-based tracking (memory overhead)
trackedMaps map[uint32]*EBPFMapInfo

// After: Slice-based tracking (memory efficient)
mapIDs []uint32
mapMemoryUsage []uint64  // Parallel arrays
```

### Zero-Copy Operations

- **Pre-allocated Pools**: Objects reused across collection cycles
- **Slice Reuse**: Reset slice length but keep underlying arrays
- **Atomic Updates**: Direct memory updates without copying

## Resource Limits and Monitoring

### Collection Status Metrics

```prometheus
# Memory management
odigos.ebpf.collector.memory_usage_bytes    # Current collector memory usage
odigos.ebpf.collection.memory_limit_hit_total # Times collection stopped due to limits

# Performance monitoring  
odigos.ebpf.collection.active               # Collection status (1=active, 0=stopped)
odigos.ebpf.collection.errors_total         # Collection errors
```

### Graceful Degradation

- **Memory Pressure**: Stop collection at 90% memory utilization
- **Error Handling**: Log errors but continue with partial data
- **Pool Exhaustion**: Fail gracefully when object pools are full

## Performance Characteristics

### Memory Usage

| Component | Memory Usage | Notes |
|-----------|--------------|-------|
| Object Pools | ~2-8MB | Pre-allocated based on config |
| Metrics Cache | <100KB | Atomic counters only |
| Tracking Arrays | 1-3MB | Efficient slice storage |
| **Total** | **5-10MB** | **Configurable maximum** |

### CPU Usage

- **Collection Overhead**: <1% CPU during 60s intervals
- **Syscall Rate**: <100 syscalls/second (rate limited)
- **/metrics Response**: <1ms (cached values)
- **Memory Allocations**: Zero during steady state

### Scalability Limits

| Metric | High-Performance | Production |
|--------|------------------|------------|
| Max Maps | 100 | 500 |
| Max Programs | 50 | 200 |
| Max Links | 25 | 100 |
| Collection Interval | 120s | 60s |
| Memory Limit | 5MB | 10MB |

## Configuration Examples

### For High-Traffic Environments

```go
config := &EBPFMetricsConfig{
    MaxMemoryBytes:        5 * 1024 * 1024,  // Very conservative
    MaxTrackedMaps:        50,               // Fewer objects
    MaxTrackedProgs:       25,               
    MaxTrackedLinks:       10,               
    CollectionInterval:    180 * time.Second, // Less frequent
    MaxConcurrentSyscalls: 1,                // Minimal syscalls
    EnableSystemStats:     false,            // Disabled for performance
}
```

### For Development/Debugging

```go
config := &EBPFMetricsConfig{
    MaxMemoryBytes:        20 * 1024 * 1024, // More memory available
    MaxTrackedMaps:        1000,             
    MaxTrackedProgs:       500,              
    MaxTrackedLinks:       200,              
    CollectionInterval:    30 * time.Second, // More frequent
    MaxConcurrentSyscalls: 5,                
    EnableSystemStats:     true,             // Full features
}
```

## Monitoring and Alerting

### Key Metrics to Monitor

```prometheus
# Memory pressure
rate(odigos_ebpf_collection_memory_limit_hit_total[5m]) > 0

# Collection health  
odigos_ebpf_collection_active == 0

# Error rate
rate(odigos_ebpf_collection_errors_total[5m]) > 0.1
```

### Performance Tuning

1. **Monitor memory usage** - Adjust pool sizes based on actual usage
2. **Track collection intervals** - Increase interval if CPU usage is high  
3. **Watch error rates** - May indicate need for larger pools or longer intervals
4. **Observe syscall rates** - Should stay well below kernel limits

## Implementation Details

### Lock-Free Design

- **Atomic Operations**: All counters use atomic operations
- **Read-Only /metrics**: No locks needed for metric exposition
- **Pool Management**: Lock-free pool allocation using CAS operations

### Error Recovery

- **Partial Failures**: Continue collection even if some objects fail
- **Memory Recovery**: Automatic pool reset between collection cycles
- **Graceful Shutdown**: Clean resource cleanup on termination

### Future Optimizations

- **eBPF-based Collection**: Use eBPF programs for kernel-side aggregation
- **Incremental Updates**: Track only changed objects between cycles
- **Compression**: Compress historical data for longer retention