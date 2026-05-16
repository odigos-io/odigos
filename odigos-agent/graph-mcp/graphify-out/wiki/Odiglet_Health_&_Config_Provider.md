# Odiglet Health & Config Provider

> 15 nodes · cohesion 0.23

## Key Concepts

- **oidc.go** (8 connections) — `services/oidc.go`
- **OidcMiddleware()** (5 connections) — `middlewares/gin.go`
- **OidcAuthCallback()** (5 connections) — `services/oidc.go`
- **GetOidcOauthConfig()** (4 connections) — `services/oidc.go`
- **getOidcProvider()** (4 connections) — `services/oidc.go`
- **GetOidcTokenVerifier()** (4 connections) — `services/oidc.go`
- **RedirectToOidcAuth()** (4 connections) — `services/oidc.go`
- **gin.go** (3 connections) — `middlewares/gin.go`
- **getOidcValuesFromConfig()** (3 connections) — `services/oidc.go`
- **setCallbackCookie()** (3 connections) — `services/utils.go`
- **isStaticFile()** (2 connections) — `middlewares/gin.go`
- **SecurityHeadersMiddleware()** (2 connections) — `middlewares/gin.go`
- **GetOidcSecret()** (2 connections) — `services/oidc.go`
- **randString()** (2 connections) — `services/utils.go`
- **EnsureOidcSecret()** (1 connections) — `services/oidc.go`

## Relationships

- [[Retry & OTLP Exporter Config]] (48 shared connections)
- [[Config YAML Field Schema]] (2 shared connections)
- [[Frontend GraphQL Loaders]] (1 shared connections)
- [[Component Log Levels Config]] (1 shared connections)

## Source Files

- `middlewares/gin.go`
- `services/oidc.go`
- `services/utils.go`

## Audit Trail

- EXTRACTED: 38 (73%)
- INFERRED: 14 (27%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*