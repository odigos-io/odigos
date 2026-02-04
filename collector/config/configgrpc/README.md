# OpenTelemetry Collector gRPC Config Module

## Overview

This is a copy of the `go.opentelemetry.io/collector/config/configgrpc` module from the [OpenTelemetry Collector repository](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc) with an additional enhancement from [PR #9673](https://github.com/open-telemetry/opentelemetry-collector/pull/9673). This enhancement enables the collector to reject incoming telemetry data during memory pressure situations before processing and decoding, preventing unnecessary memory allocation.

## Problem Statement

### Baseline Scenario

This fix addresses observations from one of our users experiencing:

- **Large cluster**: ~200 nodes
- **Heavy payload**: Large span payloads due to stack trace collection
- **Downstream issues**: Failures and delays preventing export, causing collector queue buildup
- **Memory exhaustion**: Memory limits reached
- **Continuous data flow**: New data from numerous senders continues flowing in, allocating more memory
- **System failure**: Collector pod dies with Out of Memory (OOM) error

### Root Cause Analysis

The issue occurs when the extra buffer in the memory limiter (between hard limit and OOM) cannot handle the volume of incoming telemetry, especially since each batch requires decoding into memory before rejection.

While memory thresholds are somewhat arbitrary and the process isn't fully hermetic, this appears to be the primary factor affecting collector stability under high load with slow downstream destinations.

### Impact

- Collectors become overwhelmed within seconds of startup
- System crashes with OOM errors
- No metrics reporting, preventing Horizontal Pod Autoscaler (HPA) from adding more collector replicas
- Complete service disruption

## Related Issues and PRs

- **[PR #9673](https://github.com/open-telemetry/opentelemetry-collector/pull/9673)**: Adds rejection capability to gRPC servers (temporarily applied to our fork)
- **[Issue #9591](https://github.com/open-telemetry/opentelemetry-collector/issues/9591)**: Tracking issue for this problem
- **[PR #13265](https://github.com/open-telemetry/opentelemetry-collector/pull/13265)**: Tracking issue for approach under development
- **[PR #9397](https://github.com/open-telemetry/opentelemetry-collector/pull/9397)**: Similar PR for HTTP receiver (for future data collection implementation)

## Future Plans

The OpenTelemetry Collector SIG is working on refactoring how receiver middlewares are applied. As of 06/07/2025, this is still under development with several draft PRs, including [PR #13265](https://github.com/open-telemetry/opentelemetry-collector/pull/13265).

Once the collector adopts the new "middlewares" approach, this module can be retired and removed.

## Maintenance

### How to Update

The code in this module is copied from the collector repository using the current tag (`v0.141.0` as of this writing). When upgrading the collector version, this module must also be updated accordingly.

- The original collector module, contains "replace" statements in it's `go.mod` which link to internally in the collector mono-repo (`../../pdata`). those need to be removed in odigos repo, as they confuse `make go-mod-tidy`

### Current Version

- **Collector Version**: v0.141.0
- **Last Updated**: 22/01/2026

