# Pod Details GraphQL

> 56 nodes · cohesion 0.06

## Key Concepts

- **pyroscope_convert.go** (13 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **service.go** (10 connections) — `services/profiles/service.go`
- **MergedGoogleProfileForPyroscopeSymdb()** (9 connections) — `services/profiles/flamegraph/pyroscope_symdb.go`
- **buildPyroscopeProfileFromChunks()** (8 connections) — `services/profiles/builder.go`
- **GetProfilingForSource()** (6 connections) — `services/profiles/service.go`
- **SourceIDFromStrings()** (6 connections) — `services/profiles/service.go`
- **googleProfileToSamples()** (5 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **locationFrameLabels()** (5 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **truncateFrameName()** (5 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **SourceKeyFromSourceID()** (5 connections) — `services/profiles/utils.go`
- **pprof_merge.go** (5 connections) — `services/profiles/flamegraph/pprof_merge.go`
- **pyroscope_symdb.go** (5 connections) — `services/profiles/flamegraph/pyroscope_symdb.go`
- **googleProfilesFromParsedRequest()** (4 connections) — `services/profiles/flamegraph/pprof_merge.go`
- **mergeGoogleProfilesGrouped()** (4 connections) — `services/profiles/flamegraph/pprof_merge.go`
- **functionLineLabel()** (4 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **locationFallbackLabel()** (4 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **profileTypeFromGoogleProfile()** (4 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- **BuildFlamebearerViaPyroscopeSymdb()** (4 connections) — `services/profiles/flamegraph/pyroscope_symdb.go`
- **collectGoogleProfilesFromChunks()** (4 connections) — `services/profiles/flamegraph/pyroscope_symdb.go`
- **ClearProfilingBufferForSource()** (4 connections) — `services/profiles/service.go`
- **DisableProfilingForSource()** (4 connections) — `services/profiles/service.go`
- **EnableProfilingForSource()** (4 connections) — `services/profiles/service.go`
- **tree.go** (4 connections) — `services/profiles/flamegraph/tree.go`
- **profileCompatibilityKey()** (3 connections) — `services/profiles/flamegraph/pprof_merge.go`
- **DefaultProfileType()** (3 connections) — `services/profiles/flamegraph/pyroscope_convert.go`
- *... and 31 more nodes in this community*

## Relationships

- [[Action GraphQL Schema]] (188 shared connections)
- [[Service Graph Connector]] (3 shared connections)
- [[Config YAML Field Schema]] (3 shared connections)
- [[Collector Generated Telemetry]] (2 shared connections)

## Source Files

- `graph/profiling.resolvers.go`
- `services/profiles/builder.go`
- `services/profiles/flamegraph/pprof_merge.go`
- `services/profiles/flamegraph/pyroscope_adapter.go`
- `services/profiles/flamegraph/pyroscope_convert.go`
- `services/profiles/flamegraph/pyroscope_symdb.go`
- `services/profiles/flamegraph/tree.go`
- `services/profiles/service.go`
- `services/profiles/utils.go`
- `webapp/graphql/mutations/profiling.ts`

## Audit Trail

- EXTRACTED: 139 (71%)
- INFERRED: 57 (29%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*