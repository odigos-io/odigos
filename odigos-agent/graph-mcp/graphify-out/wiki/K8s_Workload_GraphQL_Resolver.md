# K8s Workload GraphQL Resolver

> 38 nodes · cohesion 0.07

## Key Concepts

- **config.go** (19 connections) — `receivers/odigosebpfreceiver/config.go`
- **.Validate()** (13 connections) — `processors/odigossamplingprocessor/internal/sampling/error.go`
- **ClientConfig** (7 connections) — `config/configgrpc/configgrpc.go`
- **.getGrpcDialOptions()** (6 connections) — `config/configgrpc/configgrpc.go`
- **.sanitizedEndpoint()** (4 connections) — `config/configgrpc/configgrpc.go`
- **Config** (4 connections) — `receivers/odigosebpfreceiver/config.go`
- **.addHeadersIfAbsent()** (3 connections) — `config/configgrpc/configgrpc.go`
- **.isSchemeHTTP()** (3 connections) — `config/configgrpc/configgrpc.go`
- **.ToClientConn()** (3 connections) — `config/configgrpc/configgrpc.go`
- **config_test.go** (3 connections) — `connectors/servicegraphconnector/config_test.go`
- **validatePropertiesConfig()** (3 connections) — `processors/odigosurltemplateprocessor/config.go`
- **ErrorRule** (3 connections) — `processors/odigossamplingprocessor/internal/sampling/error.go`
- **SpanAttributeRule** (3 connections) — `processors/odigossamplingprocessor/internal/sampling/spanattribute.go`
- **getGRPCCompressionName()** (2 connections) — `config/configgrpc/configgrpc.go`
- **TestConfig_Validate()** (2 connections) — `processors/odigosextractattributeprocessor/config_test.go`
- **DataFormat** (2 connections) — `processors/odigosextractattributeprocessor/config.go`
- **decodeAndValidate()** (2 connections) — `processors/odigossamplingprocessor/config.go`
- **Rule** (2 connections) — `processors/odigossamplingprocessor/config.go`
- **validateK8sWorkload()** (2 connections) — `processors/odigosurltemplateprocessor/config.go`
- **spanattribute.go** (2 connections) — `processors/odigossamplingprocessor/internal/sampling/spanattribute.go`
- **AzureBlobStorageUploadConfig** (1 connections) — `exporters/azureblobstorageexporter/config.go`
- **GCSUploadConfig** (1 connections) — `exporters/googlecloudstorageexporter/config.go`
- **TestUnmarshalDefaultConfig()** (1 connections) — `exporters/mockdestinationexporter/config_test.go`
- **ConditionalRule** (1 connections) — `processors/odigosconditionalattributes/config.go`
- **NewAttributeValueConfiguration** (1 connections) — `processors/odigosconditionalattributes/config.go`
- *... and 13 more nodes in this community*

## Relationships

- [[RenameAttribute CRD]] (94 shared connections)
- [[Cypress E2E Tests]] (7 shared connections)
- [[Odigos Configuration Common]] (3 shared connections)
- [[Sampling Matchers (Collector)]] (1 shared connections)
- [[Pyroscope Profiling Conversion]] (1 shared connections)

## Source Files

- `config/configgrpc/configgrpc.go`
- `connectors/servicegraphconnector/config.go`
- `connectors/servicegraphconnector/config_test.go`
- `exporters/azureblobstorageexporter/config.go`
- `exporters/googlecloudstorageexporter/config.go`
- `exporters/mockdestinationexporter/config_test.go`
- `processors/odigosconditionalattributes/config.go`
- `processors/odigosextractattributeprocessor/config.go`
- `processors/odigosextractattributeprocessor/config_test.go`
- `processors/odigossamplingprocessor/config.go`
- `processors/odigossamplingprocessor/internal/sampling/error.go`
- `processors/odigossamplingprocessor/internal/sampling/spanattribute.go`
- `processors/odigossqldboperationprocessor/config.go`
- `processors/odigostracestateprocessor/config.go`
- `processors/odigosurltemplateprocessor/config.go`
- `receivers/odigosebpfreceiver/config.go`

## Audit Trail

- EXTRACTED: 100 (94%)
- INFERRED: 6 (6%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*