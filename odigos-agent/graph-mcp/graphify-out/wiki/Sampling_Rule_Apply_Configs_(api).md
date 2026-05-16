# Sampling Rule Apply Configs (api)

> 36 nodes · cohesion 0.08

## Key Concepts

- **deployment_config.go** (12 connections) — `services/deployment_config.go`
- **TestConnectionResult** (9 connections) — `services/test_connection/test_connection.go`
- **buildConfigResponse()** (7 connections) — `services/deployment_config.go`
- **.TestConnectionForDestination()** (6 connections) — `graph/schema.resolvers.go`
- **getInstallationStatus()** (6 connections) — `services/deployment_config.go`
- **SyncWorkloadsInNamespace()** (5 connections) — `services/namespaces.go`
- **getOdigosConfiguration()** (5 connections) — `services/url_validation.go`
- **DestinationConfigurer** (5 connections) — `services/test_connection/conversion.go`
- **GET_CONFIG** (4 connections) — `webapp/graphql/queries/config.ts`
- **url_validation.go** (4 connections) — `services/url_validation.go`
- **validateURLForTestConnection()** (4 connections) — `services/url_validation.go`
- **.PersistK8sNamespaces()** (3 connections) — `graph/schema.resolvers.go`
- **.PersistK8sSources()** (3 connections) — `graph/schema.resolvers.go`
- **MarkInstallationFinished()** (3 connections) — `services/deployment_config.go`
- **persistInstallationStatus()** (3 connections) — `services/deployment_config.go`
- **conversion.go** (3 connections) — `services/describe/utils/conversion.go`
- **ValidateDestinationURLs()** (3 connections) — `services/url_validation.go`
- **TestConnectionHoneycomb()** (3 connections) — `services/test_connection/test_connection.go`
- **replacePlaceholders()** (3 connections) — `services/test_connection/utils.go`
- **init()** (2 connections) — `kube/client.go`
- **isCentralProxyRunning()** (2 connections) — `services/deployment_config.go`
- **isConfiguredForCentralBackend()** (2 connections) — `services/deployment_config.go`
- **isDestinationConnected()** (2 connections) — `services/deployment_config.go`
- **isSourceCreated()** (2 connections) — `services/deployment_config.go`
- **utils_test.go** (2 connections) — `services/test_connection/utils_test.go`
- *... and 11 more nodes in this community*

## Relationships

- [[K8s Workload GraphQL Resolver]] (100 shared connections)
- [[CLI Centralized Install]] (4 shared connections)
- [[Service Graph Connector]] (3 shared connections)
- [[Community 207]] (2 shared connections)
- [[Config YAML Field Schema]] (2 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)
- [[OTel API Enrichment Docs]] (1 shared connections)
- [[Collector Factories]] (1 shared connections)
- [[Autoscaler K8sAttributes Resolver]] (1 shared connections)
- [[Frontend Destination Connection Test]] (1 shared connections)

## Source Files

- `graph/schema.resolvers.go`
- `kube/client.go`
- `services/deployment_config.go`
- `services/describe/utils/conversion.go`
- `services/namespaces.go`
- `services/test_connection/conversion.go`
- `services/test_connection/test_connection.go`
- `services/test_connection/utils.go`
- `services/test_connection/utils_test.go`
- `services/url_validation.go`
- `webapp/graphql/queries/config.ts`

## Audit Trail

- EXTRACTED: 94 (79%)
- INFERRED: 25 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*