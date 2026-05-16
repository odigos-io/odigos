# Config YAML Field Schema

> 24 nodes · cohesion 0.11

## Key Concepts

- **utils.go** (12 connections) — `config/utils.go`
- **AWSCloudWatch** (5 connections) — `config/awscloudwatch.go`
- **sharedConfig()** (5 connections) — `config/awscloudwatch.go`
- **LogsConfig** (5 connections) — `config/config.go`
- **MetricsConfig** (5 connections) — `config/config.go`
- **parseBool()** (5 connections) — `config/utils.go`
- **parseOtlpGrpcUrl()** (4 connections) — `config/utils.go`
- **Elasticsearch** (3 connections) — `config/elasticsearch.go`
- **.SanitizeURL()** (3 connections) — `config/elasticsearch.go`
- **otlphttp_test.go** (3 connections) — `config/otlphttp_test.go`
- **errorMissingKey()** (3 connections) — `config/utils.go`
- **getBooleanConfig()** (3 connections) — `config/utils.go`
- **parseInt()** (3 connections) — `config/utils.go`
- **parseOtlpHttpEndpoint()** (3 connections) — `config/utils.go`
- **urlHostContainsPort()** (3 connections) — `config/utils.go`
- **TestParseOtlpHttpEndpoint()** (2 connections) — `config/otlphttp_test.go`
- **addHeader()** (2 connections) — `config/utils.go`
- **TestParseUnencryptedOtlpGrpcUrl()** (2 connections) — `config/utils_test.go`
- **TestOAuth2Configuration()** (1 connections) — `config/otlphttp_test.go`
- **SpanMetricNames** (1 connections) — `config/utils.go`
- **utils_test.go** (1 connections) — `config/utils_test.go`
- **MergeOptionalBools()** (1 connections) — `mergeconfig/utils.go`
- **MergeOptionalIntChooseLower()** (1 connections) — `mergeconfig/utils.go`
- **MergeStringArrays()** (1 connections) — `mergeconfig/utils.go`

## Relationships

- [[Odiglet Main & Instrumentation]] (56 shared connections)
- [[Destination Configurations (common)]] (17 shared connections)
- [[Self-Hosted Backend Docs]] (2 shared connections)
- [[Autoscaler Profiling OTLP Config]] (2 shared connections)

## Source Files

- `config/awscloudwatch.go`
- `config/config.go`
- `config/elasticsearch.go`
- `config/otlphttp_test.go`
- `config/utils.go`
- `config/utils_test.go`
- `mergeconfig/utils.go`

## Audit Trail

- EXTRACTED: 54 (70%)
- INFERRED: 23 (30%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*