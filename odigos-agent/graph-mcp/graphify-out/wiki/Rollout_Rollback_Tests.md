# Rollout Rollback Tests

> 12 nodes · cohesion 0.24

## Key Concepts

- **TestConnectionResponse** (11 connections) — `graph/model/models_gen.go`
- **.fieldContext_Mutation_testConnectionForDestination()** (7 connections) — `graph/generated.go`
- **.fieldContext_TestConnectionResponse_message()** (4 connections) — `graph/generated.go`
- **._Mutation_testConnectionForDestination()** (4 connections) — `graph/generated.go`
- **._TestConnectionResponse_succeeded()** (4 connections) — `graph/generated.go`
- **useTestConnection.ts** (4 connections) — `webapp/hooks/destinations/useTestConnection.ts`
- **.fieldContext_TestConnectionResponse_destinationType()** (3 connections) — `graph/generated.go`
- **.fieldContext_TestConnectionResponse_reason()** (3 connections) — `graph/generated.go`
- **.fieldContext_TestConnectionResponse_succeeded()** (3 connections) — `graph/generated.go`
- **.marshalNTestConnectionResponse2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐTestConnectionResponse()** (3 connections) — `graph/generated.go`
- **._TestConnectionResponse_destinationType()** (3 connections) — `graph/generated.go`
- **._TestConnectionResponse_message()** (3 connections) — `graph/generated.go`

## Relationships

- [[Workload Describe (Frontend)]] (32 shared connections)
- [[GraphQL Marshalers (Frontend)]] (11 shared connections)
- [[Effective Collector Config Schema]] (3 shared connections)
- [[Collector Generated Telemetry]] (3 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)
- [[Frontend Sampling Rules]] (1 shared connections)
- [[CLI Centralized Install]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`
- `webapp/hooks/destinations/useTestConnection.ts`

## Audit Trail

- EXTRACTED: 50 (96%)
- INFERRED: 2 (4%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*