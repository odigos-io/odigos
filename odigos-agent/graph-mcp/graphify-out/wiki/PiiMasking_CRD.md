# PiiMasking CRD

> 13 nodes · cohesion 0.28

## Key Concepts

- **otlp_test_connection.go** (10 connections) — `services/test_connection/otlp_test_connection.go`
- **TestOTLPConnectionSuccess()** (5 connections) — `services/test_connection/otlp_test_connection_test.go`
- **TestOTLPConnectionRefused()** (4 connections) — `services/test_connection/otlp_test_connection_test.go`
- **NewOTLPHTTPTester()** (4 connections) — `services/test_connection/otlphttp_test_connection.go`
- **NewOTLPTester()** (3 connections) — `services/test_connection/otlp_test_connection.go`
- **freePort()** (3 connections) — `services/test_connection/otlp_test_connection_test.go`
- **otlpExporterConnectionTester** (3 connections) — `services/test_connection/otlp_test_connection.go`
- **otlphttpExporterConnectionTester** (3 connections) — `services/test_connection/otlphttp_test_connection.go`
- **startNoOpOTLPReceiver()** (2 connections) — `services/test_connection/otlp_test_connection_test.go`
- **.Factory()** (2 connections) — `services/test_connection/otlp_test_connection.go`
- **.ModifyConfigForConnectionTest()** (2 connections) — `services/test_connection/otlp_test_connection.go`
- **TestHTTPModifyConfigForConnectionTest_WrongType()** (2 connections) — `services/test_connection/otlphttp_test_connection_test.go`
- **dummyHTTPConfig** (1 connections) — `services/test_connection/otlphttp_test_connection_test.go`

## Relationships

- [[CLI Diagnose Port-Forward]] (44 shared connections)

## Source Files

- `services/test_connection/otlp_test_connection.go`
- `services/test_connection/otlp_test_connection_test.go`
- `services/test_connection/otlphttp_test_connection.go`
- `services/test_connection/otlphttp_test_connection_test.go`

## Audit Trail

- EXTRACTED: 30 (68%)
- INFERRED: 14 (32%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*