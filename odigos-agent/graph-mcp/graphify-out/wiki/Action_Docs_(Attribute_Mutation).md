# Action Docs (Attribute Mutation)

> 12 nodes · cohesion 0.24

## Key Concepts

- **server_middleware_test.go** (6 connections) — `config/configgrpc/server_middleware_test.go`
- **TestGrpcServerUnaryInterceptor()** (5 connections) — `config/configgrpc/server_middleware_test.go`
- **testClientMiddleware** (5 connections) — `config/configgrpc/client_middleware_test.go`
- **client_middleware_test.go** (4 connections) — `config/configgrpc/client_middleware_test.go`
- **newTestClientMiddleware()** (3 connections) — `config/configgrpc/client_middleware_test.go`
- **newTestMiddlewareConfig()** (3 connections) — `config/configgrpc/client_middleware_test.go`
- **getMiddlewareCalls()** (3 connections) — `config/configgrpc/server_middleware_test.go`
- **newTestServerMiddleware()** (3 connections) — `config/configgrpc/server_middleware_test.go`
- **TestClientMiddlewareToClientErrors()** (1 connections) — `config/configgrpc/client_middleware_test.go`
- **contextKey** (1 connections) — `config/configgrpc/server_middleware_test.go`
- **TestServerMiddlewareToServerErrors()** (1 connections) — `config/configgrpc/server_middleware_test.go`
- **testServerMiddleware** (1 connections) — `config/configgrpc/server_middleware_test.go`

## Relationships

- [[Wrapped Stream Auth Tests]] (32 shared connections)
- [[Cypress E2E Tests]] (2 shared connections)
- [[gRPC Config Tests (Collector)]] (2 shared connections)

## Source Files

- `config/configgrpc/client_middleware_test.go`
- `config/configgrpc/server_middleware_test.go`

## Audit Trail

- EXTRACTED: 30 (83%)
- INFERRED: 6 (17%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*