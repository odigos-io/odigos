# Odigos Collector Component Usage Audit

This document identifies deprecated and potentially unused components in the Odigos collector builder configuration.

**Audit Date:** 2026-01-20  
**Collector Version:** v0.130.0

---

## Summary

- **Deprecated Destinations:** 1
- **Unused Exporters (No Destination Config):** 12
- **Deprecated Configuration Fields:** 3
- **Utility Exporters (Not Destinations):** 4
- **Potentially Unused Processors:** 2
- **Actively Used Receivers:** 6 core receivers
- **Legacy/Uncertain Receivers:** 1 (zipkinreceiver - needs verification)

---

## 1. Deprecated Destinations

### Splunk (SAPM Protocol)
**Status:** ⚠️ **DEPRECATED**  
**File:** `destinations/data/splunk.yaml`  
**Display Name:** "Splunk (SAPM) (Deprecated)"  
**Exporter Used:** `sapmexporter`  
**Recommendation:** Users should migrate to `splunkotlp` destination which uses modern OTLP protocol  
**Action:** Consider removing in future major version after migration period

---

## 2. Exporters Without Corresponding Destinations

These exporters are included in the collector but have no corresponding destination configuration in `/workspace/destinations/data/` and are not referenced in the codebase:

### 2.1 Database Exporters

#### `cassandraexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Cassandra destination  
**Risk Level:** Low (can be added back if needed)

#### `carbonexporter` (Graphite/Carbon)
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Graphite/Carbon destination  
**Risk Level:** Low (legacy protocol, low demand)

#### `influxdbexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add InfluxDB destination  
**Risk Level:** Medium (InfluxDB is popular for time-series data)

#### `azuredataexplorerexporter` (Azure Kusto)
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Azure Data Explorer destination  
**Risk Level:** Low

### 2.2 SaaS Platform Exporters

#### `mezmoexporter` (formerly LogDNA)
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Mezmo destination  
**Risk Level:** Low

#### `logicmonitorexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add LogicMonitor destination  
**Risk Level:** Low

### 2.3 Message Queue Exporters

#### `pulsarexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists (Kafka exporter exists, but not Pulsar)  
**Recommendation:** Remove unless there's a plan to add Pulsar destination  
**Risk Level:** Low (Kafka covers most use cases)

### 2.4 Protocol Exporters

#### `syslogexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Syslog destination  
**Risk Level:** Medium (some enterprises still use syslog)

#### `tencentcloudlogserviceexporter`
**Status:** ❌ **NOT USED**  
**Reason:** No destination config exists  
**Recommendation:** Remove unless there's a plan to add Tencent Cloud destination  
**Risk Level:** Low (regional cloud provider)

#### `zipkinexporter`
**Status:** ⚠️ **POTENTIALLY UNUSED**  
**Reason:** No destination config exists, Jaeger destination uses OTLP exporter instead  
**Recommendation:** Keep if supporting legacy Zipkin protocol ingestion via `zipkinreceiver`  
**Risk Level:** Low (legacy protocol, but might be used for compatibility)

### 2.5 Legacy Exporters

#### `opencensusexporter`
**Status:** ⚠️ **LEGACY PROTOCOL**  
**Reason:** OpenCensus is deprecated in favor of OpenTelemetry  
**Recommendation:** Remove unless supporting legacy OpenCensus backends  
**Risk Level:** Low (OpenTelemetry OTLP is the standard)

### 2.6 Specialized Exporters

#### `sapmexporter` (SignalFx APM Protocol)
**Status:** ⚠️ **USED ONLY BY DEPRECATED DESTINATION**  
**Reason:** Only used by deprecated `splunk` destination  
**Recommendation:** Remove after deprecating Splunk (SAPM) destination  
**Risk Level:** Low (SignalFx destination uses `signalfxexporter` instead)

---

## 3. Utility Exporters (Intentionally Not Destinations)

These are NOT unused - they serve specific purposes but aren't user-facing destinations:

### `debugexporter`
**Status:** ✅ **KEEP**  
**Purpose:** Debugging - logs telemetry to stdout  
**Used For:** Troubleshooting collector configurations

### `nopexporter`
**Status:** ✅ **KEEP**  
**Purpose:** Testing/placeholder - discards data  
**Used For:** Testing pipelines without sending data

### `fileexporter`
**Status:** ✅ **KEEP**  
**Purpose:** Debugging/testing - writes to local files  
**Used For:** Local debugging and file-based backends

### `loadbalancingexporter`
**Status:** ✅ **KEEP**  
**Purpose:** Load balancing across multiple backends  
**Used For:** Internal collector architecture (not a destination)

---

## 4. Deprecated Configuration Fields

### 4.1 AWS S3 - `S3_PARTITION`
**File:** `destinations/data/awss3.yaml`  
**Status:** ⚠️ **DEPRECATED**  
**Replacement:** `S3_PARTITION_FORMAT`  
**Message:** "Deprecated field for time granularity. Use S3_PARTITION_FORMAT for custom formats."  
**Action:** Update documentation, consider removing in future major version

### 4.2 Prometheus - `PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS`
**File:** `destinations/data/prometheus.yaml`  
**Status:** ⚠️ **DEPRECATED**  
**Message:** "Notice: deprecated. not used anymore. will be removed soon."  
**Action:** Can be removed in next minor version

### 4.3 Loki/Grafana Cloud Loki - Path Change
**File:** `destinations/data/loki.yaml`, `destinations/data/grafanacloudloki.yaml`  
**Status:** ⚠️ **MIGRATION NOTICE**  
**Message:** "The path `/loki/api/v1/push` has been deprecated and replaced with `/otlp`"  
**Action:** Documentation update only (user-facing path change)

---

## 5. Potentially Unused Processors

### `sumologicprocessor`
**Status:** ⚠️ **BACKEND-SPECIFIC**  
**Usage:** Not referenced in common config code  
**Destination:** Sumo Logic destination exists (`sumologic`)  
**Recommendation:** Keep - likely used in Sumo Logic-specific pipeline configuration  
**Action:** Verify if actually used in Sumo Logic destination config

### `remotetapprocessor`
**Status:** ⚠️ **DEBUGGING TOOL**  
**Usage:** Only referenced in `components.go` (included in build)  
**Recommendation:** Keep - useful for remote debugging, even if not actively configured  
**Action:** No action needed

---

## 6. Receiver Usage Analysis

**CORRECTION:** Initial audit incorrectly identified several receivers as "potentially unused." After thorough code analysis, all receivers except `zipkinreceiver` are confirmed to be actively used in Odigos.

### Actively Used Receivers

### `otlpreceiver`
**Status:** ✅ **CORE - ACTIVELY USED**  
**Purpose:** Primary receiver for OTLP data from instrumented applications  
**Usage:** Receives telemetry from auto-instrumented applications via gRPC and HTTP  
**Location:** Used across all collector types

### `odigosebpfreceiver`
**Status:** ✅ **CORE - ACTIVELY USED**  
**Purpose:** Custom receiver for eBPF-based instrumentation  
**Usage:** Critical for Odigos's eBPF instrumentation, especially for Go applications  
**Location:** Node collector metrics pipeline

### `filelogreceiver`
**Status:** ✅ **ACTIVELY USED**  
**Purpose:** Reads logs from container log files  
**Usage:** Collects application logs from file system in node collectors  
**Location:** `autoscaler/controllers/nodecollector/collectorconfig/logs.go`  
**Configuration:** Configured with `filelogReceiverName` in logs pipeline

### `kubeletstatsreceiver`
**Status:** ✅ **ACTIVELY USED**  
**Purpose:** Collects Kubernetes pod/container metrics from Kubelet API  
**Usage:** Gathers pod, container, and node metrics including CPU, memory, disk, and network  
**Location:** `autoscaler/controllers/nodecollector/collectorconfig/metrics.go`  
**Configuration:** Configurable via `MetricsSources.KubeletStats` in OdigosConfiguration  
**Endpoint:** `https://${NODE_IP}:10250` with ServiceAccount authentication

### `hostmetricsreceiver`
**Status:** ✅ **ACTIVELY USED**  
**Purpose:** Collects host system metrics (CPU, memory, disk, filesystem, network, processes)  
**Usage:** Infrastructure monitoring for node-level metrics  
**Location:** `autoscaler/controllers/nodecollector/collectorconfig/metrics.go`  
**Configuration:** Configurable via `MetricsSources.HostMetrics` in OdigosConfiguration  
**Scrapers:** CPU utilization, memory, disk, filesystem (with kubelet exclusions), network, processes, paging

### `prometheusreceiver`
**Status:** ✅ **ACTIVELY USED**  
**Purpose:** Scrapes Prometheus metrics endpoints  
**Usage:** Self-monitoring and service graph metrics scraping  
**Location:** `common/pipelinegen/config_builder.go`  
**Configuration:** `prometheus/self-metrics` receiver for internal collector metrics and service graph connector

### `zipkinreceiver`
**Status:** ⚠️ **INCLUDED BUT USAGE UNCLEAR**  
**Purpose:** Receives Zipkin-format traces  
**Recommendation:** Verify if used for legacy Zipkin application support or can be removed  
**Action:** Needs further investigation to confirm active usage

---

## 7. Recommendations Summary

### High Priority Actions

1. **Remove Unused Exporters** - Consider removing the 12 exporters with no destination configs:
   - `cassandraexporter`
   - `carbonexporter`
   - `influxdbexporter`
   - `mezmoexporter`
   - `opencensusexporter`
   - `pulsarexporter`
   - `syslogexporter`
   - `tencentcloudlogserviceexporter`
   - `zipkinexporter`
   - `azuredataexplorerexporter`
   - `logicmonitorexporter`
   - `sapmexporter` (after deprecating Splunk SAPM destination)

2. **Deprecate & Migrate** - Mark Splunk (SAPM) destination for removal:
   - Add migration guide to `splunkotlp`
   - Set sunset date for SAPM support

3. **Clean Up Deprecated Fields**:
   - Remove `PROMETHEUS_RESOURCE_ATTRIBUTES_LABELS` field
   - Update S3 documentation to prefer `S3_PARTITION_FORMAT`

### Medium Priority Actions

4. **Verify Zipkin Receiver Usage** - Investigate whether `zipkinreceiver` is actively used or can be safely removed

5. **Audit Usage** - Before removing any component, search for:
   - User documentation references
   - Example configurations
   - Community requests/issues

### Low Priority Actions

6. **Binary Size Reduction** - Removing 12 unused exporters could significantly reduce collector binary size

7. **Maintenance Burden** - Fewer components = easier upgrades when updating OpenTelemetry versions

---

## 8. Impact Analysis

### Binary Size Savings
Removing 12 unused exporters could reduce binary size by approximately 10-20 MB (varies by exporter dependencies).

### Breaking Changes
Removing unused exporters would NOT break existing Odigos installations since:
- No destination configs reference them
- No code uses them
- Users haven't configured them

### Risk Mitigation
If any exporter needs to be restored later:
- Can be re-added to `builder-config.yaml` easily
- No data loss risk (these aren't currently used)
- Quick rollback possible

---

## 9. Next Steps

1. **Validate Findings**: Review this audit with team
2. **Check User Requests**: Search GitHub issues for requests for removed exporters
3. **Create Removal PR**: If approved, create PR to remove unused exporters
4. **Update Documentation**: Document available vs. auto-configured components
5. **Deprecation Notice**: If removing exporters, add notice to release notes

---

## 10. Questions for Review

1. Are any of the 12 unused exporters planned for future destinations?
2. Should we keep `zipkinreceiver` for legacy Zipkin application support or is it safe to remove?
3. Should we maintain `opencensusexporter` for legacy compatibility?
4. What's the timeline for removing deprecated Splunk (SAPM) destination?
5. Are there plans to add destinations for any of the unused exporters (Cassandra, InfluxDB, etc.)?

---

*This audit was generated by analyzing the builder configuration, destination configs, and codebase references. Manual verification recommended before taking action.*
