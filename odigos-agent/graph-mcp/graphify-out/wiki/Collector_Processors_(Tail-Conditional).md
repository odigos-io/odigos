# Collector Processors (Tail/Conditional)

> 48 nodes · cohesion 0.07

## Key Concepts

- **spanWithAttrs()** (10 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- **matchHttpRoute()** (8 connections) — `processors/odigostailsamplingprocessor/matchers/match.go`
- **HeadSamplingOperationMatcher()** (7 connections) — `processors/odigostailsamplingprocessor/matchers/headsampling.go`
- **TailSamplingOperationMatcher()** (7 connections) — `processors/odigostailsamplingprocessor/matchers/tailsampling.go`
- **matchHighlyRelevantRulesForSingleSpan()** (5 connections) — `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- **spanWithAttrsAndKind()** (5 connections) — `processors/odigostailsamplingprocessor/matchers/headsampling_test.go`
- **operationHttpServerMatcher()** (5 connections) — `processors/odigostailsamplingprocessor/matchers/tailsampling.go`
- **attrgetter.go** (5 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- **attrgetter_test.go** (5 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- **compareHttpMethod()** (4 connections) — `processors/odigostailsamplingprocessor/matchers/attrcompare.go`
- **comparePathToTemplate()** (4 connections) — `processors/odigostailsamplingprocessor/matchers/attrcompare.go`
- **TestSpanDurationMatcher()** (4 connections) — `processors/odigostailsamplingprocessor/matchers/highlyrelevance_test.go`
- **matchServerAddress()** (4 connections) — `processors/odigostailsamplingprocessor/matchers/match.go`
- **matchTemplatedPath()** (4 connections) — `processors/odigostailsamplingprocessor/matchers/match.go`
- **tailsampling.go** (4 connections) — `processors/odigostailsamplingprocessor/matchers/tailsampling.go`
- **compareHttpRoute()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrcompare.go`
- **getHttpMethod()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- **getHttpRoute()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- **getHttpServerPath()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- **getServerAddress()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- **TestGetHttpRoute()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- **TestGetHttpServerPath()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- **TestGetServerAddress()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- **TestHeadSamplingOperationMatcher()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/headsampling_test.go`
- **SpanDurationMatcher()** (3 connections) — `processors/odigostailsamplingprocessor/matchers/highlyrelevance.go`
- *... and 23 more nodes in this community*

## Relationships

- [[Collector Workload Info GraphQL]] (160 shared connections)
- [[Cypress E2E Tests]] (4 shared connections)
- [[Instrumentor Rollout Mocks]] (2 shared connections)

## Source Files

- `processors/odigostailsamplingprocessor/category/highlyrelevant/highlyrelevanceoperations.go`
- `processors/odigostailsamplingprocessor/matchers/attrcompare.go`
- `processors/odigostailsamplingprocessor/matchers/attrcompare_test.go`
- `processors/odigostailsamplingprocessor/matchers/attrgetter.go`
- `processors/odigostailsamplingprocessor/matchers/attrgetter_test.go`
- `processors/odigostailsamplingprocessor/matchers/headsampling.go`
- `processors/odigostailsamplingprocessor/matchers/headsampling_test.go`
- `processors/odigostailsamplingprocessor/matchers/highlyrelevance.go`
- `processors/odigostailsamplingprocessor/matchers/highlyrelevance_test.go`
- `processors/odigostailsamplingprocessor/matchers/match.go`
- `processors/odigostailsamplingprocessor/matchers/match_test.go`
- `processors/odigostailsamplingprocessor/matchers/tailsampling.go`
- `processors/odigostailsamplingprocessor/matchers/tailsampling_test.go`

## Audit Trail

- EXTRACTED: 92 (55%)
- INFERRED: 74 (45%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*