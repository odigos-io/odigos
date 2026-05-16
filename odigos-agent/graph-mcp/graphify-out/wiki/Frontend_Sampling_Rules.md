# Frontend Sampling Rules

> 55 nodes · cohesion 0.07

## Key Concepts

- **index.ts** (69 connections) — `webapp/types/index.ts`
- **Page()** (22 connections) — `webapp/app/(v2)/settings/page.tsx`
- **profiling.go** (15 connections) — `services/profiling.go`
- **useConfig()** (14 connections) — `webapp/hooks/config/useConfig.ts`
- **overview-modals-and-drawers.tsx** (13 connections) — `webapp/components/lib-imports/overview-modals-and-drawers.tsx`
- **._Query_potentialDestinations()** (11 connections) — `graph/generated.go`
- **UseSourceCrud** (8 connections) — `webapp/hooks/sources/useSourceCRUD.ts`
- **useNamespace()** (7 connections) — `webapp/hooks/namespaces/useNamespace.ts`
- **UseActionCrud** (6 connections) — `webapp/hooks/actions/useActionCRUD.ts`
- **OverviewHeader()** (6 connections) — `webapp/components/lib-imports/overview-header.tsx`
- **OverviewLayout()** (6 connections) — `webapp/app/(v2)/layout.tsx`
- **useDataStreamsCRUD.ts** (5 connections) — `webapp/hooks/data-streams/useDataStreamsCRUD.ts`
- **useDescribe()** (4 connections) — `webapp/hooks/describe/useDescribe.ts`
- **useTokenCRUD()** (4 connections) — `webapp/hooks/tokens/useTokenCRUD.ts`
- **useEffectiveConfig.ts** (4 connections) — `webapp/hooks/config/useEffectiveConfig.ts`
- **useDestinationCategories.ts** (4 connections) — `webapp/hooks/destinations/useDestinationCategories.ts`
- **useInstrumentationRuleCRUD.ts** (4 connections) — `webapp/hooks/instrumentation-rules/useInstrumentationRuleCRUD.ts`
- **useWorkloadUtils.ts** (4 connections) — `webapp/hooks/sources/useWorkloadUtils.ts`
- **useTokenTracker.ts** (4 connections) — `webapp/hooks/tokens/useTokenTracker.ts`
- **navigation.ts** (4 connections) — `webapp/utils/functions/navigation.ts`
- **getNavbarIcons()** (3 connections) — `webapp/utils/functions/navigation.ts`
- **useServiceMap()** (3 connections) — `webapp/hooks/metrics/useServiceMap.ts`
- **page.tsx** (3 connections) — `webapp/app/page.tsx`
- **useDiagnose.ts** (3 connections) — `webapp/hooks/diagnose/useDiagnose.ts`
- **useSamplingRuleCRUD.ts** (3 connections) — `webapp/hooks/sampling/useSamplingRuleCRUD.ts`
- *... and 30 more nodes in this community*

## Relationships

- [[Collector Generated Telemetry]] (159 shared connections)
- [[Profile Store & Buffer]] (56 shared connections)
- [[Scheduler Resource Settings]] (24 shared connections)
- [[Autoscaler Signal Pipelines]] (5 shared connections)
- [[GraphQL Query Resolvers]] (4 shared connections)
- [[Odigos Collector Processor Catalog]] (3 shared connections)
- [[Workload Describe (Frontend)]] (3 shared connections)
- [[Autoscaler K8sAttributes Resolver]] (2 shared connections)
- [[Action GraphQL Schema]] (2 shared connections)
- [[JVM Metrics Handler]] (2 shared connections)
- [[Community 244]] (2 shared connections)
- [[Settings Cypress Tests]] (2 shared connections)

## Source Files

- `graph/generated.go`
- `services/profiling.go`
- `webapp/app/(v2)/layout.tsx`
- `webapp/app/(v2)/settings/page.tsx`
- `webapp/app/page.tsx`
- `webapp/components/lib-imports/overview-header.tsx`
- `webapp/components/lib-imports/overview-modals-and-drawers.tsx`
- `webapp/cypress/constants/index.ts`
- `webapp/cypress/functions/index.ts`
- `webapp/graphql/queries/destination.ts`
- `webapp/graphql/queries/profiling.ts`
- `webapp/hooks/actions/useActionCRUD.ts`
- `webapp/hooks/common/useSetupHelpers.ts`
- `webapp/hooks/config/useConfig.ts`
- `webapp/hooks/config/useEffectiveConfig.ts`
- `webapp/hooks/config/useUpdateLocalUiConfig.ts`
- `webapp/hooks/data-streams/useDataStreamsCRUD.ts`
- `webapp/hooks/describe/useDescribe.ts`
- `webapp/hooks/destinations/useDestinationCategories.ts`
- `webapp/hooks/destinations/usePotentialDestinationsLegacy.ts`

## Audit Trail

- EXTRACTED: 162 (60%)
- INFERRED: 108 (40%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*