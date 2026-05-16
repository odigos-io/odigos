# Frontend Layout & Providers

> 17 nodes · cohesion 0.12

## Key Concepts

- **layout.tsx** (10 connections) — `webapp/app/layout.tsx`
- **useCSRF.ts** (6 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **ApolloProvider** (4 connections) — `webapp/app/layout.tsx`
- **ThemeProvider** (2 connections) — `webapp/app/layout.tsx`
- **Layout()** (2 connections) — `webapp/app/(setup)/layout.tsx`
- **UseCSRF** (2 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **RootLayout()** (1 connections) — `webapp/app/layout.tsx`
- **makeClient()** (1 connections) — `webapp/lib/apollo-provider.tsx`
- **StyledComponentsRegistry()** (1 connections) — `webapp/lib/theme-provider.tsx`
- **ContentUnderActions** (1 connections) — `webapp/app/(overview)/layout.tsx`
- **PageContent** (1 connections) — `webapp/app/(overview)/layout.tsx`
- **createCSRFHeaders()** (1 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **CSRFTokenResponse** (1 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **getCSRFTokenFromCookie()** (1 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **getCSRFTokenFromServer()** (1 connections) — `webapp/hooks/tokens/useCSRF.ts`
- **ContentRow** (1 connections) — `webapp/app/(v2)/layout.tsx`
- **ViewportColumn** (1 connections) — `webapp/app/(v2)/layout.tsx`

## Relationships

- [[Autoscaler Signal Pipelines]] (32 shared connections)
- [[Collector Generated Telemetry]] (3 shared connections)
- [[Profile Store & Buffer]] (2 shared connections)

## Source Files

- `webapp/app/(overview)/layout.tsx`
- `webapp/app/(setup)/layout.tsx`
- `webapp/app/(v2)/layout.tsx`
- `webapp/app/layout.tsx`
- `webapp/hooks/tokens/useCSRF.ts`
- `webapp/lib/apollo-provider.tsx`
- `webapp/lib/theme-provider.tsx`

## Audit Trail

- EXTRACTED: 34 (92%)
- INFERRED: 3 (8%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*