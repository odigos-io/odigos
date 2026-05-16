# Cypress E2E Tests

> 32 nodes · cohesion 0.11

## Key Concepts

- **.Len()** (77 connections) — `connectors/servicegraphconnector/internal/store/store.go`
- **.Evaluate()** (21 connections) — `processors/odigossamplingprocessor/internal/sampling/error.go`
- **costreductionoperations.go** (7 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **highlyrelevanceoperations.go** (7 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **templatize.go** (7 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **GetPercentageOrDefault()** (6 connections) — `processors/odigostailsamplingprocessor/category/utils.go`
- **parseUserInputRuleString()** (6 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **calculateDecidingRule()** (5 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **.applyTemplatizationOnPathWithRules()** (5 connections) — `processors/odigosurltemplateprocessor/processor.go`
- **calculateCostReductionDecidingRule()** (4 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **selectCostReductionRuleFromMatches()** (4 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **selectHighlyRelevantRuleFromMatches()** (4 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **defaultTemplatizeURLPath()** (4 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **HttpRouteLatencyRule** (4 connections) — `processors/odigossamplingprocessor/internal/sampling/latency.go`
- **getCostReductionRulesConfig()** (3 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **matchCostReductionRulesForSingleSpan()** (3 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **getHighlyRelevantOperationsConfig()** (3 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **setHighlyRelevantRuleAttributesOnSpan()** (3 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **attemptTemplateWithRule()** (3 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **setCostReductionRuleAttributesOnSpan()** (2 connections) — `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- **recordEvalResultForSingleSpan()** (2 connections) — `processors/odigostailsamplingprocessor/category/costreduction/metrics.go`
- **getSegmentTemplatizationString()** (2 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **parseRuleTemplateString()** (2 connections) — `processors/odigosurltemplateprocessor/templatize.go`
- **noisyoperations.go** (2 connections) — `processors/odigostailsamplingprocessor/category/noisy/noisyoperations.go`
- **.matchEndpoint()** (2 connections) — `processors/odigossamplingprocessor/internal/sampling/latency.go`
- *... and 7 more nodes in this community*

## Relationships

- [[Instrumentor Rollout Mocks]] (15 shared connections)
- [[Sampling Rule Types (GraphQL)]] (11 shared connections)
- [[Sampling Matchers (Collector)]] (10 shared connections)
- [[RenameAttribute CRD]] (7 shared connections)
- [[Collector Client gRPC Config]] (6 shared connections)
- [[gRPC Config Tests (Collector)]] (5 shared connections)
- [[Collector Workload Info GraphQL]] (4 shared connections)
- [[Frontend Destination CRUD]] (4 shared connections)
- [[CLI Uninstall & Logging]] (4 shared connections)
- [[Workload Resource Attrs (Collector)]] (3 shared connections)
- [[Odigos Configuration Common]] (3 shared connections)
- [[Collector Settings CRD]] (2 shared connections)

## Source Files

- `connectors/servicegraphconnector/internal/store/store.go`
- `processors/odigossamplingprocessor/internal/sampling/error.go`
- `processors/odigossamplingprocessor/internal/sampling/latency.go`
- `processors/odigostailsamplingprocessor/category/costreduction/costreductionoperations.go`
- `processors/odigostailsamplingprocessor/category/costreduction/metrics.go`
- `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- `processors/odigostailsamplingprocessor/category/noisy/noisyoperations.go`
- `processors/odigostailsamplingprocessor/category/utils.go`
- `processors/odigosurltemplateprocessor/processor.go`
- `processors/odigosurltemplateprocessor/templatize.go`
- `receivers/odigosebpfreceiver/metrics.go`

## Audit Trail

- EXTRACTED: 91 (47%)
- INFERRED: 104 (53%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*