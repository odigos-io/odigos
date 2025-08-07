# eBPF Metrics Collection for Odiglet

This package provides comprehensive metrics collection for eBPF objects managed by odiglet. These metrics are essential for debugging memory usage, CPU performance, and resource allocation issues in production environments with thousands of processes.

## Overview

The eBPF metrics collector provides real-time visibility into:
- **eBPF Maps**: Memory usage, entry counts, types, and names
- **eBPF Programs**: Load times, instruction counts, runtime statistics, memory usage
- **eBPF Links**: Attachment points and target information
- **System Resources**: Total memory allocated to eBPF objects and resource limits

## Metrics Exposed

### Object Counts and Types

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.objects.total` | UpDownCounter | Total number of eBPF objects allocated | `object_type`, `object_name` |
| `odigos.ebpf.instrumentation.manager.loaded_programs` | UpDownCounter | eBPF programs loaded by instrumentation manager | |
| `odigos.ebpf.instrumentation.manager.loaded_maps` | UpDownCounter | eBPF maps loaded by instrumentation manager | |
| `odigos.ebpf.instrumentation.manager.loaded_links` | UpDownCounter | eBPF links loaded by instrumentation manager | |

### Memory Usage Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.map.memory_bytes` | UpDownCounter | Memory used by eBPF maps | `map_id`, `map_name`, `map_type` |
| `odigos.ebpf.program.memory_bytes` | UpDownCounter | Memory used by eBPF programs | `prog_id`, `prog_name`, `prog_type` |
| `odigos.ebpf.total_memory_bytes` | UpDownCounter | Total memory allocated to eBPF objects | `resource_type` |

### Performance Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.program.runtime_ns_total` | Counter | Total runtime of eBPF programs in nanoseconds | `prog_id`, `prog_name`, `prog_type` |
| `odigos.ebpf.program.runs_total` | Counter | Total number of eBPF program executions | `prog_id`, `prog_name`, `prog_type` |
| `odigos.ebpf.program.instructions` | UpDownCounter | Number of instructions in eBPF programs | `prog_id`, `prog_name`, `prog_type` |
| `odigos.ebpf.program.verification_duration_ms` | Histogram | Time spent verifying eBPF programs | |
| `odigos.ebpf.instrumentation.manager.program_load_duration_ms` | Histogram | Time taken to load eBPF programs | |

### Map Operations

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.map.entries` | UpDownCounter | Number of entries in eBPF maps | `map_id`, `map_name`, `map_type` |
| `odigos.ebpf.map.operations_total` | Counter | Total number of eBPF map operations | |
| `odigos.ebpf.instrumentation.manager.map_lookups_total` | Counter | Total eBPF map lookups performed | |
| `odigos.ebpf.instrumentation.manager.map_updates_total` | Counter | Total eBPF map updates performed | |

### Buffer Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.perf_buffer.events_total` | Counter | Total events processed through perf buffers | |
| `odigos.ebpf.ring_buffer.events_total` | Counter | Total events processed through ring buffers | |
| `odigos.ebpf.perf_buffer.lost_events_total` | Counter | Total events lost in perf buffers | |

### Error and System Metrics

| Metric Name | Type | Description | Labels |
|-------------|------|-------------|--------|
| `odigos.ebpf.instrumentation.manager.program_errors_total` | Counter | Total eBPF program execution errors | |
| `odigos.ebpf.system.resource_usage_percent` | UpDownCounter | eBPF system resource usage percentage | `resource` |

## Production Debugging Use Cases

### Memory Leak Detection

Monitor `odigos.ebpf.total_memory_bytes` and `odigos.ebpf.map.memory_bytes` to identify:
- Maps that continuously grow without bounds
- Programs consuming excessive memory
- Memory not being freed when processes exit

**Example Alert:**
```yaml
- alert: EBPFMemoryLeak
  expr: rate(odigos_ebpf_total_memory_bytes[5m]) > 0
  for: 10m
  annotations:
    summary: "eBPF memory continuously increasing"
```

### Performance Issues

Use `odigos.ebpf.program.runtime_ns_total` and `odigos.ebpf.program.runs_total` to:
- Identify hot eBPF programs consuming excessive CPU
- Detect programs with unexpectedly long execution times
- Monitor execution frequency patterns

**Example Query:**
```promql
# Average execution time per program
rate(odigos_ebpf_program_runtime_ns_total[5m]) / rate(odigos_ebpf_program_runs_total[5m])
```

### Resource Exhaustion

Monitor `odigos.ebpf.objects.total` and `odigos.ebpf.system.resource_usage_percent` to:
- Track total number of eBPF objects allocated
- Prevent hitting kernel limits on eBPF resources
- Identify runaway object creation

### Map Efficiency

Use `odigos.ebpf.map.entries` and operation counters to:
- Monitor map utilization rates
- Identify oversized or undersized maps
- Track lookup/update patterns for optimization

## Configuration

### Collection Interval

The default collection interval is 30 seconds. This can be adjusted:

```go
collector.SetCollectionInterval(60 * time.Second)
```

### System Statistics

System-level statistics collection can be enabled/disabled:

```go
collector.EnableSystemStats(false)
```

## Implementation Details

### Data Collection

The collector uses Linux BPF syscalls to enumerate and inspect:
- **Maps**: Via `BPF_MAP_GET_NEXT_ID` and `BPF_OBJ_GET_INFO_BY_FD`
- **Programs**: Via `BPF_PROG_GET_NEXT_ID` and `BPF_OBJ_GET_INFO_BY_FD`
- **Links**: Via `BPF_LINK_GET_NEXT_ID` and `BPF_OBJ_GET_INFO_BY_FD`

### Memory Calculation

Memory usage is calculated as:
- **Maps**: `(key_size + value_size) * max_entries`
- **Programs**: `jited_prog_len + xlated_prog_len`

### Error Handling

The collector is designed to be resilient:
- Individual object collection failures don't stop the entire collection
- Permissions errors are logged but don't crash the collector
- Missing syscalls/features are gracefully handled

## Troubleshooting

### Permission Issues

Ensure odiglet has the required capabilities:
- `CAP_BPF` (Linux 5.8+) or `CAP_SYS_ADMIN` (older kernels)
- `CAP_PERFMON` for accessing performance data

### High Memory Usage

If metrics collection itself uses too much memory:
1. Increase collection interval
2. Disable system statistics collection
3. Filter metrics by program/map names if needed

### Missing Metrics

If certain metrics are not appearing:
1. Check kernel version compatibility (minimum 5.4 recommended)
2. Verify BPF syscall support
3. Check dmesg for BPF-related errors

## Integration with Monitoring Systems

### Prometheus

Metrics are automatically exposed via the odiglet `/metrics` endpoint when controller-runtime metrics are enabled.

### Grafana Dashboard

Key visualizations for production monitoring:

```json
{
  "title": "eBPF Memory Usage",
  "targets": [{
    "expr": "odigos_ebpf_total_memory_bytes",
    "legendFormat": "Total eBPF Memory"
  }]
}
```

### Alerting Rules

```yaml
groups:
- name: ebpf.rules
  rules:
  - alert: EBPFHighMemoryUsage
    expr: odigos_ebpf_total_memory_bytes > 1e9  # 1GB
    for: 5m
    annotations:
      summary: "eBPF using more than 1GB memory"
      
  - alert: EBPFMapGrowth
    expr: rate(odigos_ebpf_map_entries[10m]) > 1000
    for: 5m
    annotations:
      summary: "eBPF map growing rapidly"
```

## Performance Impact

The metrics collector is designed to have minimal performance impact:
- Uses efficient BPF syscalls for enumeration
- Collection runs in separate goroutine
- Configurable collection intervals
- Graceful error handling prevents blocking

Expected overhead:
- **CPU**: < 1% additional CPU usage
- **Memory**: < 50MB additional memory for tracking
- **Network**: Minimal (only metric export to Prometheus)

## Future Enhancements

Planned improvements include:
- Per-CPU map statistics
- BTF information tracking
- Real-time map entry counting (vs. max_entries estimation)
- Integration with kernel tracepoints for more detailed metrics
- Support for additional eBPF object types (BTF, cgroup storage, etc.)