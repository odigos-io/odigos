# OTLP-based Destination Docs

> 34 nodes · cohesion 0.12

## Key Concepts

- **processor_test.go** (33 connections) — `processors/odigostrafficmetrics/processor_test.go`
- **newUrlTemplateProcessor()** (16 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **runProcessorTests()** (9 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **generateTraceData()** (7 connections) — `processors/odigostrafficmetrics/processor_test.go`
- **TestProcessor_Traces()** (6 connections) — `processors/odigostrafficmetrics/processor_test.go`
- **assertSpanNameAndAttribute()** (6 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **generateTestTrace()** (4 connections) — `processors/odigossqldboperationprocessor/processor_test.go`
- **countSpans()** (4 connections) — `processors/odigostracefilterprocessor/processor_test.go`
- **TestProcessor_CustomIdsRegexp()** (4 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_IncludeExclude()** (4 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_TemplatizationRules()** (4 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_Wildcard()** (4 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **firstCapture()** (3 connections) — `processors/odigosextractattributeprocessor/processor_test.go`
- **TestBuildExtractionRegex_URL()** (3 connections) — `processors/odigosextractattributeprocessor/processor_test.go`
- **createTestTraces()** (3 connections) — `processors/odigostracefilterprocessor/processor_test.go`
- **TestDropUnsampledSpans()** (3 connections) — `processors/odigostracefilterprocessor/processor_test.go`
- **TestNoEvaluatorsPassesThrough()** (3 connections) — `processors/odigostracefilterprocessor/processor_test.go`
- **TestDefaultDateTemplatization()** (3 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_EmailAddresses()** (3 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_EmptyPath()** (3 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_HexEncoded()** (3 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestProcessor_NoLetters()** (3 connections) — `processors/odigosurltemplateprocessor/processor_test.go`
- **TestDBOperationProcessor_ExistingDbOperationName()** (2 connections) — `processors/odigossqldboperationprocessor/processor_test.go`
- **TestDBOperationProcessor_NoDbQueryText()** (2 connections) — `processors/odigossqldboperationprocessor/processor_test.go`
- **TestDBOperationProcessor_SetDbOperationName()** (2 connections) — `processors/odigossqldboperationprocessor/processor_test.go`
- *... and 9 more nodes in this community*

## Relationships

- [[Frontend Destination CRUD]] (138 shared connections)
- [[Cypress E2E Tests]] (4 shared connections)
- [[Instrumentor Rollout Mocks]] (4 shared connections)
- [[Community 226]] (1 shared connections)
- [[Collector Client gRPC Config]] (1 shared connections)
- [[Community 216]] (1 shared connections)

## Source Files

- `processors/odigosextractattributeprocessor/processor_test.go`
- `processors/odigoslogsresourceattrsprocessor/processor_test.go`
- `processors/odigossqldboperationprocessor/processor_test.go`
- `processors/odigostracefilterprocessor/processor_test.go`
- `processors/odigostracestateprocessor/processor_test.go`
- `processors/odigostrafficmetrics/processor_test.go`
- `processors/odigosurltemplateprocessor/processor.go`
- `processors/odigosurltemplateprocessor/processor_test.go`

## Audit Trail

- EXTRACTED: 120 (81%)
- INFERRED: 29 (19%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*