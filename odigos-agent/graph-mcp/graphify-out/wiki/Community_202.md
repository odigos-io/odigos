# Community 202

> 10 nodes · cohesion 0.31

## Key Concepts

- **mockAuthServer** (10 connections) — `config/configgrpc/configgrpc_test.go`
- **authStreamServerInterceptor()** (7 connections) — `config/configgrpc/configgrpc.go`
- **authUnaryServerInterceptor()** (6 connections) — `config/configgrpc/configgrpc.go`
- **TestDefaultStreamInterceptorAuthFailure()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultStreamInterceptorAuthFailureWithStatusErr()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultStreamInterceptorMissingMetadata()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultUnaryInterceptorAuthFailure()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultUnaryInterceptorAuthFailureWithStatusErr()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultUnaryInterceptorAuthSucceeded()** (3 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDefaultUnaryInterceptorMissingMetadata()** (3 connections) — `config/configgrpc/configgrpc_test.go`

## Relationships

- [[Community 206]] (28 shared connections)
- [[gRPC Config Tests (Collector)]] (9 shared connections)
- [[Odigos Configuration Common]] (4 shared connections)
- [[TestConnection GraphQL]] (3 shared connections)

## Source Files

- `config/configgrpc/configgrpc.go`
- `config/configgrpc/configgrpc_test.go`

## Audit Trail

- EXTRACTED: 28 (64%)
- INFERRED: 16 (36%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*