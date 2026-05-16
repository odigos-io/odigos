# Action Resolver Logic

> 32 nodes · cohesion 0.17

## Key Concepts

- **03-sources.cy.ts** (16 connections) — `webapp/cypress/e2e/03-sources.cy.ts`
- **04-destinations.cy.ts** (16 connections) — `webapp/cypress/e2e/04-destinations.cy.ts`
- **namespaces.go** (15 connections) — `services/namespaces.go`
- **05-actions.cy.ts** (15 connections) — `webapp/cypress/e2e/05-actions.cy.ts`
- **06-rules.cy.ts** (15 connections) — `webapp/cypress/e2e/06-rules.cy.ts`
- **07-sampling.cy.ts** (15 connections) — `webapp/cypress/e2e/07-sampling.cy.ts`
- **ROUTES** (11 connections) — `webapp/cypress/constants/index.ts`
- **02-onboarding.cy.ts** (11 connections) — `webapp/cypress/e2e/02-onboarding.cy.ts`
- **visitPage()** (9 connections) — `webapp/cypress/functions/index.ts`
- **waitForGraphqlOperation()** (9 connections) — `webapp/cypress/functions/index.ts`
- **DATA_IDS** (8 connections) — `webapp/cypress/constants/index.ts`
- **handleExceptions()** (8 connections) — `webapp/cypress/functions/index.ts`
- **CRD_NAMES** (7 connections) — `webapp/cypress/constants/index.ts`
- **TEXTS** (7 connections) — `webapp/cypress/constants/index.ts`
- **awaitToast()** (7 connections) — `webapp/cypress/functions/index.ts`
- **SELECTED_ENTITIES** (6 connections) — `webapp/cypress/constants/index.ts`
- **getCrdIds()** (6 connections) — `webapp/cypress/functions/index.ts`
- **getCrdById()** (5 connections) — `webapp/cypress/functions/index.ts`
- **deleteV2Entity()** (4 connections) — `webapp/cypress/functions/index.ts`
- **updateV2Entity()** (4 connections) — `webapp/cypress/functions/index.ts`
- **01-connection.cy.ts** (4 connections) — `webapp/cypress/e2e/01-connection.cy.ts`
- **dismissSamplingOnboardingModal()** (2 connections) — `webapp/cypress/functions/index.ts`
- **routes.tsx** (2 connections) — `webapp/utils/constants/routes.tsx`
- **API** (1 connections) — `webapp/utils/constants/routes.tsx`
- **crdIds** (1 connections) — `webapp/cypress/e2e/03-sources.cy.ts`
- *... and 7 more nodes in this community*

## Relationships

- [[Scheduler Resource Settings]] (162 shared connections)
- [[Collector Generated Telemetry]] (24 shared connections)
- [[Autoscaler K8sAttributes Resolver]] (12 shared connections)
- [[Settings Cypress Tests]] (8 shared connections)
- [[Profile Store & Buffer]] (1 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)
- [[K8s Workload GraphQL Resolver]] (1 shared connections)
- [[Sampling Rule Apply Configs (api)]] (1 shared connections)

## Source Files

- `services/namespaces.go`
- `webapp/cypress/constants/index.ts`
- `webapp/cypress/e2e/01-connection.cy.ts`
- `webapp/cypress/e2e/02-onboarding.cy.ts`
- `webapp/cypress/e2e/03-sources.cy.ts`
- `webapp/cypress/e2e/04-destinations.cy.ts`
- `webapp/cypress/e2e/05-actions.cy.ts`
- `webapp/cypress/e2e/06-rules.cy.ts`
- `webapp/cypress/e2e/07-sampling.cy.ts`
- `webapp/cypress/functions/index.ts`
- `webapp/types/namespaces.ts`
- `webapp/utils/constants/routes.tsx`

## Audit Trail

- EXTRACTED: 210 (100%)
- INFERRED: 1 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*