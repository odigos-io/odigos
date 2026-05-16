# Distro Resolver (Instrumentor)

> 12 nodes · cohesion 0.17

## Key Concepts

- **SetupHeader()** (9 connections) — `webapp/components/lib-imports/setup-header.tsx`
- **useDestinationCRUD.ts** (7 connections) — `webapp/hooks/destinations/useDestinationCRUD.ts`
- **useSSE.ts** (5 connections) — `webapp/hooks/notification/useSSE.ts`
- **useSSE()** (4 connections) — `webapp/hooks/notification/useSSE.ts`
- **mapNoUndefinedFields()** (1 connections) — `webapp/hooks/destinations/useDestinationCRUD.ts`
- **backRoutes** (1 connections) — `webapp/components/lib-imports/setup-header.tsx`
- **getFormDataFromDestination()** (1 connections) — `webapp/components/lib-imports/setup-header.tsx`
- **nextRoutes** (1 connections) — `webapp/components/lib-imports/setup-header.tsx`
- **SetupHeaderProps** (1 connections) — `webapp/components/lib-imports/setup-header.tsx`
- **CrdTypes** (1 connections) — `webapp/hooks/notification/useSSE.ts`
- **DebouncedEvent** (1 connections) — `webapp/hooks/notification/useSSE.ts`
- **EventTypes** (1 connections) — `webapp/hooks/notification/useSSE.ts`

## Relationships

- [[Profile Store & Buffer]] (26 shared connections)
- [[Collector Generated Telemetry]] (7 shared connections)

## Source Files

- `webapp/components/lib-imports/setup-header.tsx`
- `webapp/hooks/destinations/useDestinationCRUD.ts`
- `webapp/hooks/notification/useSSE.ts`

## Audit Trail

- EXTRACTED: 21 (64%)
- INFERRED: 12 (36%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*