# Central Backend Architecture Docs

> 17 nodes · cohesion 0.16

## Key Concepts

- **Central Backend (port 8081)** (6 connections) — `docs/central/architecture.mdx`
- **Odigos Central Architecture** (5 connections) — `docs/central/architecture.mdx`
- **Odigos Central Authentication** (4 connections) — `docs/central/authentication.mdx`
- **Connecting Remote Clusters** (4 connections) — `docs/central/adding-connections/remote-clusters.mdx`
- **Connecting VM Agent** (4 connections) — `docs/central/adding-connections/vmagent.mdx`
- **Central Proxy** (3 connections) — `docs/central/architecture.mdx`
- **Bundled Keycloak Identity Provider** (3 connections) — `docs/central/authentication.mdx`
- **Central Redis (state store)** (2 connections) — `docs/central/architecture.mdx`
- **Central UI (port 3000)** (2 connections) — `docs/central/architecture.mdx`
- **Odigos VM Agent (eBPF on Linux)** (2 connections) — `docs/central/adding-connections/vmagent.mdx`
- **OIDC SSO Provider** (2 connections) — `docs/central/authentication.mdx`
- **SAML SSO Provider** (2 connections) — `docs/central/authentication.mdx`
- **auth.externalUrl Helm value** (1 connections) — `docs/central/authentication.mdx`
- **centralProxy.centralBackendURL value** (1 connections) — `docs/central/adding-connections/remote-clusters.mdx`
- **Central Proxy TLS / mTLS config** (1 connections) — `docs/central/adding-connections/remote-clusters.mdx`
- **odictl interactive CLI** (1 connections) — `docs/central/adding-connections/vmagent.mdx`
- **tower (Central controller) config** (1 connections) — `docs/central/adding-connections/vmagent.mdx`

## Relationships

- No strong cross-community connections detected

## Source Files

- `docs/central/adding-connections/remote-clusters.mdx`
- `docs/central/adding-connections/vmagent.mdx`
- `docs/central/architecture.mdx`
- `docs/central/authentication.mdx`

## Audit Trail

- EXTRACTED: 38 (86%)
- INFERRED: 6 (14%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*