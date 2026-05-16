# gRPC Server Config (Collector)

> 22 nodes · cohesion 0.12

## Key Concepts

- **configgrpc.go** (26 connections) — `config/configgrpc/configgrpc.go`
- **.getGrpcServerOptions()** (7 connections) — `config/configgrpc/configgrpc.go`
- **contextWithClient()** (5 connections) — `config/configgrpc/configgrpc.go`
- **enhanceStreamWithClientInformation()** (5 connections) — `config/configgrpc/configgrpc.go`
- **ServerConfig** (4 connections) — `config/configgrpc/configgrpc.go`
- **enhanceWithClientInformation()** (3 connections) — `config/configgrpc/configgrpc.go`
- **TestGrpcClientExtraOption()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestGrpcServerExtraOption()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestStreamInterceptorEnhancesClient()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestContextWithClient()** (2 connections) — `config/configgrpc/configgrpc_test.go`
- **WithGrpcDialOption()** (2 connections) — `config/configgrpc/configgrpc.go`
- **WithGrpcServerOption()** (2 connections) — `config/configgrpc/configgrpc.go`
- **grpcDialOptionWrapper** (2 connections) — `config/configgrpc/configgrpc.go`
- **grpcServerOptionWrapper** (2 connections) — `config/configgrpc/configgrpc.go`
- **KeepaliveEnforcementPolicy** (2 connections) — `config/configgrpc/configgrpc.go`
- **.ToServer()** (2 connections) — `config/configgrpc/configgrpc.go`
- **ToServerOption** (2 connections) — `config/configgrpc/configgrpc.go`
- **.isToClientConnOption()** (1 connections) — `config/configgrpc/configgrpc.go`
- **KeepaliveClientConfig** (1 connections) — `config/configgrpc/configgrpc.go`
- **KeepaliveServerConfig** (1 connections) — `config/configgrpc/configgrpc.go`
- **KeepaliveServerParameters** (1 connections) — `config/configgrpc/configgrpc.go`
- **ToClientConnOption** (1 connections) — `config/configgrpc/configgrpc.go`

## Relationships

- [[Odigos Configuration Common]] (56 shared connections)
- [[Community 227]] (4 shared connections)
- [[Community 206]] (4 shared connections)
- [[gRPC Config Tests (Collector)]] (4 shared connections)
- [[Community 243]] (3 shared connections)
- [[RenameAttribute CRD]] (3 shared connections)
- [[Cypress E2E Tests]] (3 shared connections)
- [[TestConnection GraphQL]] (2 shared connections)
- [[GraphQL Introspection]] (1 shared connections)

## Source Files

- `config/configgrpc/configgrpc.go`
- `config/configgrpc/configgrpc_test.go`

## Audit Trail

- EXTRACTED: 67 (84%)
- INFERRED: 13 (16%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*