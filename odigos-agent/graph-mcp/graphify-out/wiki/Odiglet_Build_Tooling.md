# Odiglet Build Tooling

> 13 nodes · cohesion 0.22

## Key Concepts

- **CSRFService** (8 connections) — `services/csrf.go`
- **csrf.go** (5 connections) — `services/csrf.go`
- **GetCSRFService()** (4 connections) — `services/csrf.go`
- **CSRFMiddleware()** (3 connections) — `middlewares/csrf.go`
- **CSRFTokenHandler()** (3 connections) — `middlewares/csrf.go`
- **.ValidateToken()** (3 connections) — `services/csrf.go`
- **.cleanup()** (2 connections) — `services/csrf.go`
- **.GenerateToken()** (2 connections) — `services/csrf.go`
- **.RefreshToken()** (2 connections) — `services/csrf.go`
- **.ValidateRequest()** (2 connections) — `services/csrf.go`
- **.GetCSRFToken()** (1 connections) — `services/csrf.go`
- **.SetCSRFCookie()** (1 connections) — `services/csrf.go`
- **CSRFToken** (1 connections) — `services/csrf.go`

## Relationships

- [[Collector Client Tests]] (34 shared connections)
- [[Component Log Levels Config]] (2 shared connections)
- [[Odigos Collector Processor Catalog]] (1 shared connections)

## Source Files

- `middlewares/csrf.go`
- `services/csrf.go`

## Audit Trail

- EXTRACTED: 30 (81%)
- INFERRED: 7 (19%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*