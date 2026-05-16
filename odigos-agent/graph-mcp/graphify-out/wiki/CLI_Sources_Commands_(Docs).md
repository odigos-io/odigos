# CLI Sources Commands (Docs)

> 37 nodes · cohesion 0.11

## Key Concepts

- **Destination** (41 connections) — `graph/model/models_gen.go`
- **DataStream** (19 connections) — `graph/model/models_gen.go`
- **.CreateNewDestination()** (11 connections) — `graph/schema.resolvers.go`
- **UpdateDestination()** (11 connections) — `services/destinations.go`
- **DeleteDestinationOrRemoveStreamName()** (8 connections) — `services/data_stream.go`
- **K8sDestinationToEndpointFormat()** (7 connections) — `services/destinations.go`
- **.DeleteDataStream()** (6 connections) — `graph/schema.resolvers.go`
- **.UpdateDataStream()** (6 connections) — `graph/schema.resolvers.go`
- **destinationDataStreamsNotNull()** (6 connections) — `services/data_stream.go`
- **UpdateDestinationsCurrentStreamName()** (5 connections) — `services/data_stream.go`
- **CreateResourceWithGenerateName()** (5 connections) — `services/utils.go`
- **CREATE_DESTINATION** (4 connections) — `webapp/graphql/mutations/destination.ts`
- **DeleteSourcesOrRemoveStreamName()** (4 connections) — `services/data_stream.go`
- **removeStreamNameFromDestination()** (4 connections) — `services/data_stream.go`
- **DestinationTypeConfigToCategoryItem()** (4 connections) — `services/destinations.go`
- **GetDestinationTypeConfig()** (4 connections) — `services/destinations.go`
- **ArrayContains()** (4 connections) — `services/utils.go`
- **DELETE_DESTINATION** (3 connections) — `webapp/graphql/mutations/destination.ts`
- **ExtractDataStreamsFromDestination()** (3 connections) — `services/data_stream.go`
- **shouldDeleteDestination()** (3 connections) — `services/data_stream.go`
- **UpdateSourcesCurrentStreamName()** (3 connections) — `services/data_stream.go`
- **AddDestinationOwnerReferenceToSecret()** (3 connections) — `services/destinations.go`
- **ExportedSignalsObjectToSlice()** (3 connections) — `services/destinations.go`
- **GetDestinationSecretFields()** (3 connections) — `services/destinations.go`
- **TransformFieldsToDataAndSecrets()** (3 connections) — `services/destinations.go`
- *... and 12 more nodes in this community*

## Relationships

- [[Sampling Rule Apply Configs (api)]] (146 shared connections)
- [[Collector Factories]] (7 shared connections)
- [[Odigos Collector Processor Catalog]] (6 shared connections)
- [[GraphQL Query Resolvers]] (6 shared connections)
- [[Service Graph Connector]] (5 shared connections)
- [[Config YAML Field Schema]] (5 shared connections)
- [[GraphQL Marshalers (Frontend)]] (3 shared connections)
- [[CLI Centralized Install]] (3 shared connections)
- [[Effective Collector Config Schema]] (2 shared connections)
- [[Instrumentation Rule Schema (GraphQL)]] (2 shared connections)
- [[K8s Workload GraphQL Resolver]] (2 shared connections)
- [[Autoscaler ConfigMap Sync]] (1 shared connections)

## Source Files

- `graph/model/models_gen.go`
- `graph/schema.resolvers.go`
- `services/data_stream.go`
- `services/destinations.go`
- `services/utils.go`
- `webapp/graphql/mutations/destination.ts`
- `webapp/graphql/queries/data-streams.ts`
- `webapp/graphql/queries/destination.ts`
- `webapp/types/destinations.ts`
- `webapp/utils/functions/destinations.ts`

## Audit Trail

- EXTRACTED: 116 (60%)
- INFERRED: 78 (40%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*