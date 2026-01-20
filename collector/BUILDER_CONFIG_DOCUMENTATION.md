# Odigos Collector Builder Configuration Documentation

This document provides a comprehensive audit of all components included in the Odigos OpenTelemetry Collector builder configuration (`builder-config.yaml`). It explains why each component is included and how it's used in Odigos.

## Table of Contents
- [Overview](#overview)
- [Extensions](#extensions)
- [Exporters](#exporters)
- [Processors](#processors)
- [Receivers](#receivers)
- [Connectors](#connectors)
- [Providers](#providers)

---

## Overview

The Odigos collector is a custom-built OpenTelemetry Collector distribution tailored for Odigos's needs. It focuses on:
- Supporting multiple observability backends (vendor-agnostic approach)
- Processing telemetry data from auto-instrumented applications
- Kubernetes-native operations
- Custom processors for Odigos-specific functionality

---

## Extensions

Extensions provide capabilities that can be added to the collector but do not require direct access to telemetry data.

### Core Extensions

#### `zpagesextension`
**Package:** `go.opentelemetry.io/collector/extension/zpagesextension`  
**Purpose:** Provides live debugging endpoints for the collector  
**Usage in Odigos:** Enables operators to debug collector health and performance at runtime through HTTP endpoints (e.g., `/debug/tracez`, `/debug/pipelinez`)

#### `memorylimiterextension`
**Package:** `go.opentelemetry.io/collector/extension/memorylimiterextension`  
**Purpose:** Prevents the collector from exceeding memory limits  
**Usage in Odigos:** Critical for Kubernetes environments where resource limits are enforced. Prevents OOM kills by applying backpressure when memory usage is high.

#### `healthcheckextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension`  
**Purpose:** Exposes health check endpoints for liveness/readiness probes  
**Usage in Odigos:** Integrates with Kubernetes health checks to ensure only healthy collectors receive traffic

#### `pprofextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/pprofextension`  
**Purpose:** Enables Go pprof profiling endpoints  
**Usage in Odigos:** Allows performance profiling and debugging of the collector in production

### Authentication Extensions

Authentication extensions are needed to support various backend destinations that require different auth methods.

#### `basicauthextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/basicauthextension`  
**Purpose:** Provides HTTP Basic Authentication  
**Usage in Odigos:** Supports backends that use basic auth (e.g., some Prometheus endpoints, custom HTTP endpoints)

#### `bearertokenauthextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/bearertokenauthextension`  
**Purpose:** Provides Bearer token authentication  
**Usage in Odigos:** Supports backends requiring bearer tokens (e.g., many SaaS observability platforms)

#### `oauth2clientauthextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/oauth2clientauthextension`  
**Purpose:** Provides OAuth2 client credentials authentication  
**Usage in Odigos:** Supports backends using OAuth2 (e.g., some enterprise observability platforms)

#### `googleclientauthextension`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/extension/googleclientauthextension`  
**Purpose:** Provides Google Cloud authentication  
**Usage in Odigos:** Required for authenticating to Google Cloud backends (Google Cloud Trace, Google Cloud Logging, GCS)

---

## Exporters

Exporters send telemetry data to backends. Odigos supports a wide range of backends to be vendor-agnostic.

### Core/Debug Exporters

#### `debugexporter`
**Package:** `go.opentelemetry.io/collector/exporter/debugexporter`  
**Purpose:** Logs telemetry data to stdout  
**Usage in Odigos:** Used for debugging and troubleshooting configurations

#### `nopexporter`
**Package:** `go.opentelemetry.io/collector/exporter/nopexporter`  
**Purpose:** Discards telemetry data (no operation)  
**Usage in Odigos:** Used for testing and as a placeholder when data should be intentionally dropped

### OTLP Exporters

#### `otlpexporter`
**Package:** `go.opentelemetry.io/collector/exporter/otlpexporter`  
**Purpose:** Exports data using OTLP/gRPC protocol  
**Usage in Odigos:** Primary exporter for backends supporting OTLP (Grafana, Tempo, custom OTLP endpoints)

#### `otlphttpexporter`
**Package:** `go.opentelemetry.io/collector/exporter/otlphttpexporter`  
**Purpose:** Exports data using OTLP/HTTP protocol  
**Usage in Odigos:** Alternative OTLP exporter for backends that prefer HTTP over gRPC

### Odigos Custom Exporters

#### `azureblobstorageexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/azureblobstorageexporter`  
**Purpose:** Exports telemetry data to Azure Blob Storage  
**Usage in Odigos:** Custom exporter for long-term storage of telemetry data in Azure

#### `googlecloudstorageexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/googlecloudstorageexporter`  
**Purpose:** Exports telemetry data to Google Cloud Storage  
**Usage in Odigos:** Custom exporter for long-term storage of telemetry data in GCS

#### `mockdestinationexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/exporter/mockdestinationexporter`  
**Purpose:** Mock exporter for testing  
**Usage in Odigos:** Used in development and testing to simulate backend destinations

### AWS Exporters

#### `awscloudwatchlogsexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awscloudwatchlogsexporter`  
**Purpose:** Exports logs to AWS CloudWatch Logs  
**Usage in Odigos:** Supports AWS CloudWatch Logs as a destination

#### `awsemfexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsemfexporter`  
**Purpose:** Exports metrics in AWS EMF (Embedded Metric Format) to CloudWatch  
**Usage in Odigos:** Supports AWS CloudWatch Metrics as a destination

#### `awss3exporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awss3exporter`  
**Purpose:** Exports telemetry data to AWS S3  
**Usage in Odigos:** Supports S3 for long-term storage and data lake use cases

#### `awsxrayexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter`  
**Purpose:** Exports traces to AWS X-Ray  
**Usage in Odigos:** Supports AWS X-Ray as a distributed tracing backend

### Azure Exporters

#### `azuredataexplorerexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuredataexplorerexporter`  
**Purpose:** Exports telemetry to Azure Data Explorer (Kusto)  
**Usage in Odigos:** Supports Azure Data Explorer for analytics

#### `azuremonitorexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/azuremonitorexporter`  
**Purpose:** Exports telemetry to Azure Monitor  
**Usage in Odigos:** Supports Azure Monitor as an observability backend

### Database Exporters

#### `cassandraexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/cassandraexporter`  
**Purpose:** Exports telemetry to Cassandra  
**Usage in Odigos:** Supports Cassandra for storing telemetry data

#### `clickhouseexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/clickhouseexporter`  
**Purpose:** Exports telemetry to ClickHouse  
**Usage in Odigos:** Supports ClickHouse as a backend (popular for logs and traces)

#### `elasticsearchexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticsearchexporter`  
**Purpose:** Exports telemetry to Elasticsearch  
**Usage in Odigos:** Supports Elasticsearch/ELK stack

#### `opensearchexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opensearchexporter`  
**Purpose:** Exports telemetry to OpenSearch  
**Usage in Odigos:** Supports OpenSearch (Elasticsearch fork)

### SaaS Platform Exporters

#### `coralogixexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/coralogixexporter`  
**Purpose:** Exports telemetry to Coralogix  
**Usage in Odigos:** Supports Coralogix as an observability platform

#### `datadogexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter`  
**Purpose:** Exports telemetry to Datadog  
**Usage in Odigos:** Supports Datadog as an APM/observability platform

#### `datasetexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datasetexporter`  
**Purpose:** Exports telemetry to Scalyr Dataset  
**Usage in Odigos:** Supports Dataset/Scalyr platform

#### `honeycombmarkerexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/honeycombmarkerexporter`  
**Purpose:** Creates markers in Honeycomb  
**Usage in Odigos:** Supports Honeycomb deployment markers

#### `logicmonitorexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logicmonitorexporter`  
**Purpose:** Exports telemetry to LogicMonitor  
**Usage in Odigos:** Supports LogicMonitor platform

#### `logzioexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/logzioexporter`  
**Purpose:** Exports telemetry to Logz.io  
**Usage in Odigos:** Supports Logz.io platform

#### `mezmoexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/mezmoexporter`  
**Purpose:** Exports telemetry to Mezmo (formerly LogDNA)  
**Usage in Odigos:** Supports Mezmo platform

#### `sentryexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sentryexporter`  
**Purpose:** Exports error data to Sentry  
**Usage in Odigos:** Supports Sentry for error tracking

#### `signalfxexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/signalfxexporter`  
**Purpose:** Exports telemetry to SignalFx (Splunk)  
**Usage in Odigos:** Supports SignalFx/Splunk Observability Cloud

#### `splunkhecexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/splunkhecexporter`  
**Purpose:** Exports data to Splunk via HEC (HTTP Event Collector)  
**Usage in Odigos:** Supports Splunk Enterprise/Cloud

#### `sumologicexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sumologicexporter`  
**Purpose:** Exports telemetry to Sumo Logic  
**Usage in Odigos:** Supports Sumo Logic platform

### Google Cloud Exporters

#### `googlecloudexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudexporter`  
**Purpose:** Exports to Google Cloud Trace and Logging  
**Usage in Odigos:** Supports Google Cloud operations suite

#### `googlecloudpubsubexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter`  
**Purpose:** Exports telemetry to Google Cloud Pub/Sub  
**Usage in Odigos:** Supports async telemetry pipelines via Pub/Sub

#### `googlemanagedprometheusexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlemanagedprometheusexporter`  
**Purpose:** Exports metrics to Google Cloud Managed Service for Prometheus  
**Usage in Odigos:** Supports GMP for Prometheus-compatible metrics

### Prometheus Exporters

#### `prometheusexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter`  
**Purpose:** Exposes metrics in Prometheus format via HTTP endpoint  
**Usage in Odigos:** Allows Prometheus to scrape metrics from the collector

#### `prometheusremotewriteexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusremotewriteexporter`  
**Purpose:** Pushes metrics via Prometheus Remote Write protocol  
**Usage in Odigos:** Supports Prometheus, Cortex, Thanos, Mimir, VictoriaMetrics

### Time Series DB Exporters

#### `carbonexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/carbonexporter`  
**Purpose:** Exports metrics in Carbon/Graphite format  
**Usage in Odigos:** Supports Graphite and Carbon-compatible backends

#### `influxdbexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/influxdbexporter`  
**Purpose:** Exports metrics to InfluxDB  
**Usage in Odigos:** Supports InfluxDB time series database

### Logging Exporters

#### `lokiexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/lokiexporter`  
**Purpose:** Exports logs to Grafana Loki  
**Usage in Odigos:** Supports Loki for log aggregation

#### `syslogexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/syslogexporter`  
**Purpose:** Exports logs via syslog protocol  
**Usage in Odigos:** Supports traditional syslog servers

### Message Queue Exporters

#### `kafkaexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/kafkaexporter`  
**Purpose:** Exports telemetry to Kafka topics  
**Usage in Odigos:** Supports Kafka for streaming telemetry pipelines

#### `pulsarexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/pulsarexporter`  
**Purpose:** Exports telemetry to Apache Pulsar  
**Usage in Odigos:** Supports Pulsar for streaming telemetry

### Specialized Exporters

#### `fileexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/fileexporter`  
**Purpose:** Writes telemetry to local files  
**Usage in Odigos:** Used for debugging, testing, and file-based backends

#### `loadbalancingexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/loadbalancingexporter`  
**Purpose:** Load balances telemetry across multiple backend endpoints  
**Usage in Odigos:** Distributes load across multiple collector instances or backends

#### `opencensusexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/opencensusexporter`  
**Purpose:** Exports data in OpenCensus format  
**Usage in Odigos:** Supports legacy OpenCensus backends

#### `sapmexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/sapmexporter`  
**Purpose:** Exports traces in SignalFx APM format  
**Usage in Odigos:** Supports SignalFx APM protocol

#### `tencentcloudlogserviceexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/tencentcloudlogserviceexporter`  
**Purpose:** Exports logs to Tencent Cloud Log Service  
**Usage in Odigos:** Supports Tencent Cloud in Asian markets

#### `zipkinexporter`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/exporter/zipkinexporter`  
**Purpose:** Exports traces in Zipkin format  
**Usage in Odigos:** Supports Zipkin and Zipkin-compatible backends

---

## Processors

Processors transform, filter, and enrich telemetry data.

### Odigos Custom Processors

#### `odigossamplingprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossamplingprocessor`  
**Purpose:** Custom sampling logic for Odigos  
**Usage in Odigos:** Implements Odigos-specific sampling strategies based on service names, errors, latency, and span attributes

#### `odigosconditionalattributes`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigosconditionalattributes`  
**Purpose:** Conditionally adds/modifies attributes  
**Usage in Odigos:** Applies Odigos-specific attribute transformations based on conditions

#### `odigossqldboperationprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigossqldboperationprocessor`  
**Purpose:** Processes and normalizes SQL database operations  
**Usage in Odigos:** Extracts database operation information from SQL queries

#### `odigostrafficmetrics`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/odigos/processor/odigostrafficmetrics`  
**Purpose:** Generates traffic metrics from traces  
**Usage in Odigos:** Creates service-to-service traffic metrics for network observability

#### `odigosurltemplateprocessor`
**Package:** `github.com/odigos-io/odigos/collector/processor/odigosurltemplateprocessor`  
**Purpose:** Templatizes URLs to reduce cardinality  
**Usage in Odigos:** Converts URLs like `/users/123` to `/users/{id}` to prevent high-cardinality issues

### Core Processors

#### `batchprocessor`
**Package:** `go.opentelemetry.io/collector/processor/batchprocessor`  
**Purpose:** Batches telemetry data before export  
**Usage in Odigos:** Improves throughput and reduces network overhead by batching data

#### `memorylimiterprocessor`
**Package:** `go.opentelemetry.io/collector/processor/memorylimiterprocessor`  
**Purpose:** Prevents memory overuse in the processing pipeline  
**Usage in Odigos:** Works with memorylimiterextension to enforce memory limits

### Attribute Processors

#### `attributesprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/attributesprocessor`  
**Purpose:** Adds, updates, or deletes attributes  
**Usage in Odigos:** Allows users to customize attributes via configuration

#### `resourceprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor`  
**Purpose:** Modifies resource attributes  
**Usage in Odigos:** Updates resource-level attributes (e.g., service name, environment)

#### `k8sattributesprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/k8sattributesprocessor`  
**Purpose:** Enriches telemetry with Kubernetes metadata  
**Usage in Odigos:** Critical for Kubernetes environments - adds pod, namespace, deployment info

#### `resourcedetectionprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor`  
**Purpose:** Detects resource information from environment  
**Usage in Odigos:** Auto-detects cloud provider, Kubernetes, and host information

### Filtering Processors

#### `filterprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor`  
**Purpose:** Filters telemetry based on conditions  
**Usage in Odigos:** Allows users to drop unwanted telemetry (e.g., health checks)

#### `redactionprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/redactionprocessor`  
**Purpose:** Redacts sensitive information  
**Usage in Odigos:** Removes PII and sensitive data from telemetry

### Grouping Processors

#### `groupbyattrsprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbyattrsprocessor`  
**Purpose:** Groups spans by attributes  
**Usage in Odigos:** Reorganizes spans for better backend processing

#### `groupbytraceprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/groupbytraceprocessor`  
**Purpose:** Groups spans by trace ID  
**Usage in Odigos:** Ensures all spans from a trace are processed together

### Metrics Processors

#### `cumulativetodeltaprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/cumulativetodeltaprocessor`  
**Purpose:** Converts cumulative metrics to delta  
**Usage in Odigos:** Required for backends that only accept delta metrics

#### `deltatorateprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/deltatorateprocessor`  
**Purpose:** Converts delta metrics to rate metrics  
**Usage in Odigos:** Calculates rates from delta metrics

#### `metricsgenerationprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricsgenerationprocessor`  
**Purpose:** Generates new metrics from existing ones  
**Usage in Odigos:** Creates calculated metrics (e.g., ratios, percentages)

#### `metricstransformprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/metricstransformprocessor`  
**Purpose:** Transforms metric names and labels  
**Usage in Odigos:** Renames metrics and modifies labels for backend compatibility

### Sampling Processors

#### `probabilisticsamplerprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/probabilisticsamplerprocessor`  
**Purpose:** Probabilistic trace sampling  
**Usage in Odigos:** Reduces trace volume by sampling a percentage

#### `tailsamplingprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor`  
**Purpose:** Makes sampling decisions after seeing complete traces  
**Usage in Odigos:** Intelligent sampling based on trace characteristics (errors, latency)

### Span Processors

#### `spanprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/spanprocessor`  
**Purpose:** Modifies span names and attributes  
**Usage in Odigos:** Customizes span data for better backend representation

### Routing Processors

#### `routingprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/routingprocessor`  
**Purpose:** Routes telemetry to different pipelines based on conditions  
**Usage in Odigos:** Enables conditional routing (e.g., send errors to different backends)

### Transformation Processors

#### `transformprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor`  
**Purpose:** Applies OTTL (OpenTelemetry Transformation Language) transformations  
**Usage in Odigos:** Powerful transformation engine for complex data manipulation

#### `sumologicprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/sumologicprocessor`  
**Purpose:** Formats data for Sumo Logic  
**Usage in Odigos:** Applies Sumo Logic-specific transformations

### Specialized Processors

#### `remotetapprocessor`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/processor/remotetapprocessor`  
**Purpose:** Samples telemetry for remote debugging  
**Usage in Odigos:** Enables remote debugging and sampling of live data

---

## Receivers

Receivers ingest telemetry data from various sources.

### OTLP Receivers

#### `otlpreceiver`
**Package:** `go.opentelemetry.io/collector/receiver/otlpreceiver`  
**Purpose:** Receives OTLP data via gRPC and HTTP  
**Usage in Odigos:** Primary receiver for auto-instrumented applications sending OTLP data

### Trace Receivers

#### `zipkinreceiver`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/zipkinreceiver`  
**Purpose:** Receives traces in Zipkin format  
**Usage in Odigos:** Supports applications using Zipkin instrumentation

### Log Receivers

#### `filelogreceiver`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver`  
**Purpose:** Reads logs from files  
**Usage in Odigos:** Collects application logs from log files

### Metrics Receivers

#### `kubeletstatsreceiver`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/kubeletstatsreceiver`  
**Purpose:** Collects pod and node metrics from Kubelet  
**Usage in Odigos:** Gathers Kubernetes resource metrics (CPU, memory, disk, network)

#### `hostmetricsreceiver`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver`  
**Purpose:** Collects host system metrics  
**Usage in Odigos:** Gathers system-level metrics (CPU, memory, disk, network)

#### `prometheusreceiver`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver`  
**Purpose:** Scrapes Prometheus metrics endpoints  
**Usage in Odigos:** Collects metrics from Prometheus exporters

### Odigos Custom Receivers

#### `odigosebpfreceiver`
**Package:** `github.com/odigos-io/odigos/collector/receivers/odigosebpfreceiver`  
**Purpose:** Receives telemetry data from eBPF instrumentation  
**Usage in Odigos:** Critical for Odigos's eBPF-based auto-instrumentation, especially for Go applications

---

## Connectors

Connectors act as both exporter and receiver, connecting different pipelines.

### Core Connectors

#### `forwardconnector`
**Package:** `go.opentelemetry.io/collector/connector/forwardconnector`  
**Purpose:** Forwards data from one pipeline to another  
**Usage in Odigos:** Enables pipeline chaining and fan-out patterns

### Metric Generation Connectors

#### `countconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector`  
**Purpose:** Generates count metrics from telemetry  
**Usage in Odigos:** Creates metrics counting spans, logs, etc.

#### `exceptionsconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/exceptionsconnector`  
**Purpose:** Generates metrics from exception spans  
**Usage in Odigos:** Creates error/exception metrics from traces

#### `servicegraphconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/servicegraphconnector`  
**Purpose:** Generates service dependency graph metrics  
**Usage in Odigos:** Creates service topology and dependency metrics

#### `spanmetricsconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector`  
**Purpose:** Generates metrics from span data  
**Usage in Odigos:** Creates RED (Rate, Error, Duration) metrics from traces

### Backend-Specific Connectors

#### `datadogconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/datadogconnector`  
**Purpose:** Generates Datadog-specific metrics from traces  
**Usage in Odigos:** Creates APM stats for Datadog backend

### Routing Connectors

#### `routingconnector`
**Package:** `github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector`  
**Purpose:** Routes telemetry between pipelines based on conditions  
**Usage in Odigos:** Enables dynamic routing across pipeline boundaries

### Odigos Custom Connectors

#### `odigosrouterconnector`
**Package:** `github.com/odigos-io/odigos/collector/connectors/odigosrouterconnector`  
**Purpose:** Custom routing logic for Odigos  
**Usage in Odigos:** Implements Odigos-specific routing between pipelines based on destinations

---

## Providers

Providers supply configuration data to the collector.

#### `envprovider`
**Package:** `go.opentelemetry.io/collector/confmap/provider/envprovider`  
**Purpose:** Loads configuration from environment variables  
**Usage in Odigos:** Allows configuration via env vars (Kubernetes-native pattern)

#### `odigosk8scmprovider`
**Package:** `go.opentelemetry.io/collector/confmap/provider/odigosk8scmprovider`  
**Purpose:** Custom Kubernetes ConfigMap provider  
**Usage in Odigos:** Loads collector configuration from Kubernetes ConfigMaps (Odigos-specific)

#### `fileprovider`
**Package:** `go.opentelemetry.io/collector/confmap/provider/fileprovider`  
**Purpose:** Loads configuration from files  
**Usage in Odigos:** Standard file-based configuration

---

## Excludes

### `github.com/knadh/koanf v1.5.0`
**Reason:** Excluded due to OpenTelemetry issue #8127 (dependency conflict or bug)

---

## Component Versioning

All components are pinned to version `v0.130.0` to ensure:
- Compatibility across all components
- Reproducible builds
- Controlled upgrades

---

## Custom Component Details

Odigos maintains several custom components (in `replaces` section):
- **Processors:** odigossamplingprocessor, odigosconditionalattributes, odigossqldboperationprocessor, odigostrafficmetrics, odigosurltemplateprocessor
- **Exporters:** azureblobstorageexporter, googlecloudstorageexporter, mockdestinationexporter
- **Receivers:** odigosebpfreceiver
- **Connectors:** odigosrouterconnector
- **Providers:** odigosk8scmprovider

These are built locally and use `replace` directives to override upstream versions.

---

## Maintenance Notes

When updating components:
1. Update the version number in `dist.version`
2. Update all component versions consistently
3. Test custom components for compatibility
4. Review changelog for breaking changes
5. Update this documentation if component usage changes

---

## Future Considerations

### Potential Additions
- Additional database exporters as user demand grows
- New processors for specialized transformations
- Additional authentication extensions for new backends

### Potential Removals
- Exporters with low usage could be removed to reduce binary size
- Deprecated components should be removed when upstream support ends

---

*Last updated: 2026-01-20*
*Odigos Collector Version: v0.130.0*
