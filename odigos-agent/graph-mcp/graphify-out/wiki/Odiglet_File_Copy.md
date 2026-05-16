# Odiglet File Copy

> 15 nodes · cohesion 0.20

## Key Concepts

- **RetryOnFailureConfig** (9 connections) — `graph/model/models_gen.go`
- **OtlpExporterConfig** (7 connections) — `graph/model/models_gen.go`
- **.fieldContext_CollectorNodeConfig_otlpExporterConfiguration()** (6 connections) — `graph/generated.go`
- **.fieldContext_OtlpExporterConfig_retryOnFailure()** (5 connections) — `graph/generated.go`
- **.fieldContext_RetryOnFailureConfig_maxInterval()** (5 connections) — `graph/generated.go`
- **._CollectorNodeConfig_otlpExporterConfiguration()** (4 connections) — `graph/generated.go`
- **._OtlpExporterConfig_enableDataCompression()** (4 connections) — `graph/generated.go`
- **._OtlpExporterConfig_retryOnFailure()** (4 connections) — `graph/generated.go`
- **.fieldContext_OtlpExporterConfig_enableDataCompression()** (3 connections) — `graph/generated.go`
- **.fieldContext_OtlpExporterConfig_timeout()** (3 connections) — `graph/generated.go`
- **.fieldContext_RetryOnFailureConfig_enabled()** (3 connections) — `graph/generated.go`
- **.marshalOOtlpExporterConfig2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐOtlpExporterConfig()** (3 connections) — `graph/generated.go`
- **.marshalORetryOnFailureConfig2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐRetryOnFailureConfig()** (3 connections) — `graph/generated.go`
- **._RetryOnFailureConfig_initialInterval()** (3 connections) — `graph/generated.go`
- **._RetryOnFailureConfig_maxElapsedTime()** (3 connections) — `graph/generated.go`

## Relationships

- [[Odiglet Health & Config Provider]] (42 shared connections)
- [[GraphQL Marshalers (Frontend)]] (15 shared connections)
- [[GraphQL Mutation Schema]] (2 shared connections)
- [[Odigos CRD Informers (api)]] (2 shared connections)
- [[Odigos Collector Processor Catalog]] (2 shared connections)
- [[CLI Centralized Install]] (2 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 65 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*