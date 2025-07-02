# Odigos Stress Testing Framework

A comprehensive stress testing framework for the OpenTelemetry Collector in Odigos, designed to test the gateway collector under extreme load conditions with configurable parameters and automated resource monitoring.

## Overview

This stress testing framework provides:

- **Configurable Load Generation**: Test different spans/sec rates, backend ingestion speeds, and processor configurations
- **Resource Monitoring**: Real-time CPU and memory monitoring with configurable thresholds
- **Automated Execution**: Can run manually or via scheduled jobs (nightly)
- **Comprehensive Reporting**: Generates multiple graphs and performance metrics
- **Capacity Planning**: Determines spans/sec capacity for given CPU/memory allocations
- **Scenario Testing**: Pre-configured scenarios for common use cases

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│  Load Generator │────│ Gateway Collector │────│ Mock Backends   │
│  (spans/sec)    │    │ (Under Test)     │    │ (configurable)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         │              ┌────────────────┐              │
         └──────────────│   Monitoring   │──────────────┘
                        │  (Prometheus)  │
                        └────────────────┘
                                 │
                        ┌────────────────┐
                        │   Grafana +    │
                        │  Visualization │
                        └────────────────┘
```

## Features

### 🚀 Load Generation
- Configurable spans per second (100 - 100,000+)
- Variable trace complexity (spans per trace, attributes, etc.)
- Support for different data patterns (constant, burst, ramp-up)
- Multi-protocol support (OTLP, Jaeger, Zipkin)

### 📊 Monitoring & Metrics
- Real-time CPU and memory monitoring
- Throughput and latency measurements
- Error rate tracking
- Resource utilization analysis
- Queue depth monitoring

### 🎯 Scenario Testing
- **Basic Load**: Simple throughput testing
- **Burst Traffic**: Sudden traffic spikes
- **Sustained Load**: Long-running capacity tests
- **Resource Limits**: OOM and CPU threshold testing
- **Backend Backpressure**: Slow backend simulation

### 📈 Visualization
- Performance graphs over time
- Resource utilization charts
- Throughput vs. resource usage correlation
- Error rate analysis
- Capacity planning recommendations

## Quick Start

### Prerequisites
- Kubernetes cluster (local or remote)
- kubectl configured
- Docker installed
- Prometheus and Grafana (optional, can be auto-deployed)

### Basic Usage

```bash
# Run a basic stress test
make stress-test SCENARIO=basic-load DURATION=10m

# Run with custom parameters
make stress-test SPANS_PER_SEC=5000 DURATION=30m CPU_LIMIT=2 MEMORY_LIMIT=4Gi

# Run nightly test suite
make stress-test-nightly
```

### Configuration Files

Tests are configured using YAML files in `tests/stress/scenarios/`:

```yaml
# Example: basic-load.yaml
name: "Basic Load Test"
description: "Test basic throughput with 1000 spans/sec"
duration: "10m"
load:
  spans_per_second: 1000
  trace_complexity: simple
  protocols: [otlp]
resources:
  cpu_limit: "1000m"
  memory_limit: "2Gi"
  cpu_threshold: 80
  memory_threshold: 80
backends:
  - name: "fast-backend"
    delay: "10ms"
    success_rate: 99.9
monitoring:
  prometheus_enabled: true
  grafana_enabled: true
  export_metrics: true
```

## Test Scenarios

### 1. Basic Load Testing
```bash
make stress-test SCENARIO=basic-load-1k      # 1,000 spans/sec
make stress-test SCENARIO=basic-load-5k      # 5,000 spans/sec
make stress-test SCENARIO=basic-load-10k     # 10,000 spans/sec
make stress-test SCENARIO=basic-load-50k     # 50,000 spans/sec
```

### 2. Resource Constraint Testing
```bash
make stress-test SCENARIO=cpu-limited        # Limited CPU resources
make stress-test SCENARIO=memory-limited     # Limited memory resources
make stress-test SCENARIO=oom-prevention     # OOM prevention testing
```

### 3. Backend Scenarios
```bash
make stress-test SCENARIO=slow-backend       # Slow backend response
make stress-test SCENARIO=backend-failure    # Backend failure simulation
make stress-test SCENARIO=backpressure       # Backpressure handling
```

### 4. Complex Processing
```bash
make stress-test SCENARIO=heavy-processing   # Multiple processors
make stress-test SCENARIO=sampling-load      # With sampling
make stress-test SCENARIO=filtering-load     # With filtering
```

## Monitoring and Alerting

The framework includes comprehensive monitoring:

### Prometheus Metrics
- `odigos_stress_spans_generated_total`
- `odigos_stress_spans_processed_total`
- `odigos_stress_processing_duration_seconds`
- `odigos_stress_memory_usage_bytes`
- `odigos_stress_cpu_usage_percent`
- `odigos_stress_errors_total`

### Grafana Dashboards
- Real-time performance monitoring
- Resource utilization tracking
- Capacity planning insights
- Comparison between test runs

### Alerting Rules
- CPU usage > 80%
- Memory usage > 80%
- Error rate > 1%
- Processing latency > 1s
- Queue depth > 10,000

## Results and Reporting

### Automated Reports
After each test run, the framework generates:

1. **Performance Summary**: Key metrics and recommendations
2. **Resource Utilization Graphs**: CPU/Memory over time
3. **Throughput Analysis**: Spans/sec vs. resource usage
4. **Error Analysis**: Error rates and patterns
5. **Capacity Recommendations**: Optimal resource allocation

### Output Files
```
tests/stress/results/
├── 2024-01-15_basic-load-5k/
│   ├── summary.json
│   ├── metrics.json
│   ├── graphs/
│   │   ├── cpu-usage.png
│   │   ├── memory-usage.png
│   │   ├── throughput.png
│   │   └── latency.png
│   └── recommendations.md
```

## Advanced Configuration

### Custom Load Patterns
```yaml
load:
  pattern: "ramp-up"
  start_rate: 100
  end_rate: 10000
  ramp_duration: "5m"
  steady_duration: "10m"
```

### Processor Configuration
```yaml
processors:
  - name: "batch"
    config:
      send_batch_size: 1000
      timeout: "1s"
  - name: "memory_limiter"
    config:
      limit_mib: 1024
      spike_limit_mib: 256
```

### Backend Simulation
```yaml
backends:
  - name: "primary"
    endpoint: "http://mock-backend:14318"
    delay: "50ms"
    success_rate: 99.5
    backpressure_threshold: 1000
```

## Continuous Integration

### Nightly Testing
```bash
# Add to your CI/CD pipeline
make stress-test-nightly
```

### GitHub Actions Integration
```yaml
name: Nightly Stress Tests
on:
  schedule:
    - cron: '0 2 * * *'  # Run at 2 AM UTC
jobs:
  stress-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - name: Run Stress Tests
        run: make stress-test-nightly
      - name: Upload Results
        uses: actions/upload-artifact@v3
        with:
          name: stress-test-results
          path: tests/stress/results/
```

## Troubleshooting

### Common Issues

1. **OOM Kills**: Increase memory limits or reduce load
2. **High CPU Usage**: Optimize processor configuration
3. **Backend Timeouts**: Adjust backend simulation parameters
4. **Missing Metrics**: Ensure Prometheus scraping is configured

### Debug Mode
```bash
make stress-test SCENARIO=basic-load DEBUG=true
```

## Performance Baselines

The framework establishes baseline performance metrics:

| Scenario | Spans/sec | CPU (cores) | Memory (GB) | Latency (p99) |
|----------|-----------|-------------|-------------|---------------|
| Basic    | 1,000     | 0.2         | 0.5         | 50ms          |
| Medium   | 5,000     | 0.8         | 1.2         | 100ms         |
| High     | 10,000    | 1.5         | 2.5         | 200ms         |
| Extreme  | 50,000    | 4.0         | 8.0         | 500ms         |

## Contributing

1. Add new scenarios in `tests/stress/scenarios/`
2. Extend load generators in `tests/stress/loadgen/`
3. Add new metrics in `tests/stress/monitoring/`
4. Update documentation for new features

## License

This stress testing framework is part of the Odigos project and follows the same license terms.