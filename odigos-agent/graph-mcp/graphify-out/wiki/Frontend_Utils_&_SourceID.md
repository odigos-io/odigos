# Frontend Utils & SourceID

> 21 nodes · cohesion 0.13

## Key Concepts

- **DiagnoseResponse** (10 connections) — `graph/model/models_gen.go`
- **DiagnoseStats** (7 connections) — `graph/model/models_gen.go`
- **.fieldContext_DiagnoseResponse_stats()** (6 connections) — `graph/generated.go`
- **.fieldContext_Query_diagnose()** (6 connections) — `graph/generated.go`
- **.field_Query_diagnose_args()** (5 connections) — `graph/generated.go`
- **._DiagnoseResponse_includeMetrics()** (4 connections) — `graph/generated.go`
- **._DiagnoseResponse_includeSourceWorkloads()** (4 connections) — `graph/generated.go`
- **._DiagnoseResponse_sourceWorkloadNamespaces()** (4 connections) — `graph/generated.go`
- **._DiagnoseStats_fileCount()** (4 connections) — `graph/generated.go`
- **._DiagnoseStats_totalSizeBytes()** (4 connections) — `graph/generated.go`
- **._DiagnoseStats_totalSizeHuman()** (4 connections) — `graph/generated.go`
- **.fieldContext_DiagnoseResponse_includeSourceWorkloads()** (4 connections) — `graph/generated.go`
- **._Query_diagnose()** (4 connections) — `graph/generated.go`
- **.field_Query_diagnose_argsDryRun()** (3 connections) — `graph/generated.go`
- **.fieldContext_DiagnoseResponse_includeMetrics()** (3 connections) — `graph/generated.go`
- **.fieldContext_DiagnoseStats_fileCount()** (3 connections) — `graph/generated.go`
- **.fieldContext_DiagnoseStats_totalSizeBytes()** (3 connections) — `graph/generated.go`
- **.fieldContext_DiagnoseStats_totalSizeHuman()** (3 connections) — `graph/generated.go`
- **.marshalNDiagnoseResponse2githubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐDiagnoseResponse()** (3 connections) — `graph/generated.go`
- **.marshalNDiagnoseStats2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐDiagnoseStats()** (3 connections) — `graph/generated.go`
- **.unmarshalODiagnoseInput2ᚖgithubᚗcomᚋodigosᚑioᚋodigosᚋfrontendᚋgraphᚋmodelᚐDiagnoseInput()** (3 connections) — `graph/generated.go`

## Relationships

- [[Odiglet Instrumentation Reconciler]] (49 shared connections)
- [[GraphQL Marshalers (Frontend)]] (27 shared connections)
- [[Effective Collector Config Schema]] (3 shared connections)
- [[Odigos Collector Processor Catalog]] (3 shared connections)
- [[GraphQL Query Resolvers]] (2 shared connections)
- [[Frontend Sampling Rules]] (2 shared connections)
- [[CLI Centralized Install]] (2 shared connections)
- [[Odigos CRD Informers (api)]] (1 shared connections)
- [[Profile Store & Buffer]] (1 shared connections)

## Source Files

- `graph/generated.go`
- `graph/model/models_gen.go`

## Audit Trail

- EXTRACTED: 90 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*