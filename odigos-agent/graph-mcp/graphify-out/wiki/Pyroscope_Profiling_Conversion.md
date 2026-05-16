# Pyroscope Profiling Conversion

> 57 nodes · cohesion 0.07

## Key Concepts

- **processor.go** (29 connections) — `processors/odigostrafficmetrics/processor.go`
- **.processTraces()** (24 connections) — `processors/odigossamplingprocessor/processor.go`
- **urlTemplateProcessor** (13 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **conditionalAttributesProcessor** (9 connections) — `processors/odigosconditionalattributes/processor.go`
- **tailSamplingProcessor** (7 connections) — `processors/odigostailsamplingprocessor/processor.go`
- **.processSpan()** (6 connections) — `processors/odigosextractattributeprocessor/processor.go`
- **utils.go** (6 connections) — `processors/odigostailsamplingprocessor/utils.go`
- **partialK8sAttrsProcessor** (5 connections) — `processors/odigoslogsresourceattrsprocessor/processor.go`
- **dataSizesMetricsProcessor** (5 connections) — `processors/odigostrafficmetrics/processor.go`
- **.processLogs()** (5 connections) — `processors/odigostrafficmetrics/processor.go`
- **.calculateTemplatedUrlFromAttrWithRules()** (5 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **.processSpanWithRules()** (5 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **.processMetricDataPoint()** (4 connections) — `processors/odigosconditionalattributes/processor.go`
- **compileRegexExtractors()** (4 connections) — `processors/odigosextractattributeprocessor/processor.go`
- **extractPodUIDFromFilePath()** (4 connections) — `processors/odigoslogsresourceattrsprocessor/processor.go`
- **DetectSQLOperationName()** (4 connections) — `processors/odigossqldboperationprocessor/processor.go`
- **.evaluateNoisyOperations()** (4 connections) — `processors/odigostailsamplingprocessor/processor.go`
- **extractOdigosTraceStateValue()** (4 connections) — `processors/odigostailsamplingprocessor/utils.go`
- **.attributeSetFromResource()** (4 connections) — `processors/odigostrafficmetrics/processor.go`
- **.processMetrics()** (4 connections) — `processors/odigostrafficmetrics/processor.go`
- **.enhanceSpanWithRules()** (4 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **.parseRuleStrings()** (4 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **.addAttributes()** (3 connections) — `processors/odigosconditionalattributes/processor.go`
- **.setDefaultValueAttributes()** (3 connections) — `processors/odigosconditionalattributes/processor.go`
- **extractAttributeProcessor** (3 connections) — `processors/odigosextractattributeprocessor/processor.go`
- *... and 32 more nodes in this community*

## Relationships

- [[Instrumentor Rollout Mocks]] (217 shared connections)
- [[Cypress E2E Tests]] (14 shared connections)
- [[Sampling Rule Types (GraphQL)]] (5 shared connections)
- [[Frontend Destination CRUD]] (2 shared connections)
- [[Collector Workload Info GraphQL]] (2 shared connections)
- [[Community 216]] (1 shared connections)
- [[GraphQL Introspection]] (1 shared connections)

## Source Files

- `processors/odigosconditionalattributes/processor.go`
- `processors/odigosextractattributeprocessor/processor.go`
- `processors/odigoslogsresourceattrsprocessor/processor.go`
- `processors/odigoslogsresourceattrsprocessor/processor_test.go`
- `processors/odigossamplingprocessor/processor.go`
- `processors/odigossqldboperationprocessor/processor.go`
- `processors/odigostailsamplingprocessor/category/utils.go`
- `processors/odigostailsamplingprocessor/processor.go`
- `processors/odigostailsamplingprocessor/utils.go`
- `processors/odigostracefilterprocessor/processor.go`
- `processors/odigostracestateprocessor/processor.go`
- `processors/odigostrafficmetrics/processor.go`
- `processors/odigosurltemplateprocessor/processor.go`

## Audit Trail

- EXTRACTED: 219 (90%)
- INFERRED: 23 (10%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*