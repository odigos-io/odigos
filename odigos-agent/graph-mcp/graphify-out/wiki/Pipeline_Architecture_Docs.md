# Pipeline Architecture Docs

> 12 nodes · cohesion 0.23

## Key Concepts

- **.Context()** (8 connections) — `config/configgrpc/configgrpc_test.go`
- **wrapServerStream()** (5 connections) — `config/configgrpc/wrappedstream.go`
- **wrappedstream_test.go** (4 connections) — `config/configgrpc/wrappedstream_test.go`
- **TestDefaultStreamInterceptorAuthSucceeded()** (4 connections) — `config/configgrpc/configgrpc_test.go`
- **TestDoubleWrapping()** (3 connections) — `config/configgrpc/wrappedstream_test.go`
- **TestWrapServerStream()** (3 connections) — `config/configgrpc/wrappedstream_test.go`
- **wrappedstream.go** (2 connections) — `config/configgrpc/wrappedstream.go`
- **fakeServerStream** (2 connections) — `config/configgrpc/wrappedstream_test.go`
- **mockedStream** (2 connections) — `config/configgrpc/configgrpc_test.go`
- **mockServerStream** (2 connections) — `config/configgrpc/configgrpc_test.go`
- **wrappedServerStream** (2 connections) — `config/configgrpc/wrappedstream.go`
- **ctxKey** (1 connections) — `config/configgrpc/wrappedstream_test.go`

## Relationships

- [[TestConnection GraphQL]] (30 shared connections)
- [[gRPC Config Tests (Collector)]] (3 shared connections)
- [[Community 206]] (3 shared connections)
- [[Odigos Configuration Common]] (2 shared connections)

## Source Files

- `config/configgrpc/configgrpc_test.go`
- `config/configgrpc/wrappedstream.go`
- `config/configgrpc/wrappedstream_test.go`

## Audit Trail

- EXTRACTED: 31 (82%)
- INFERRED: 7 (18%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*