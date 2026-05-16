# Enterprise Installation Docs

> 17 nodes · cohesion 0.21

## Key Concepts

- **deployment.go** (9 connections) — `controllers/common/deployment.go`
- **getDesiredDeployment()** (8 connections) — `controllers/clustercollector/deployment.go`
- **syncHPA()** (8 connections) — `controllers/clustercollector/hpa.go`
- **syncDeployment()** (6 connections) — `controllers/clustercollector/deployment.go`
- **intPtr()** (4 connections) — `controllers/clustercollector/deployment.go`
- **hpa.go** (4 connections) — `controllers/clustercollector/hpa.go`
- **GetDeploymentName()** (3 connections) — `controllers/common/deployment.go`
- **boolPtr()** (2 connections) — `controllers/clustercollector/deployment.go`
- **deleteOldDeployments()** (2 connections) — `controllers/clustercollector/deployment.go`
- **getSecretsFromDests()** (2 connections) — `controllers/clustercollector/deployment.go`
- **int64Ptr()** (2 connections) — `controllers/clustercollector/deployment.go`
- **patchDeployment()** (2 connections) — `controllers/clustercollector/deployment.go`
- **buildHPACommonFields()** (2 connections) — `controllers/clustercollector/hpa.go`
- **buildv2beta2Metrics()** (2 connections) — `controllers/clustercollector/hpa.go`
- **buildv2Metrics()** (2 connections) — `controllers/clustercollector/hpa.go`
- **Sha256Hash()** (2 connections) — `controllers/common/hash.go`
- **hash.go** (1 connections) — `controllers/common/hash.go`

## Relationships

- [[Autoscaler Sampler Handlers]] (58 shared connections)
- [[Community 223]] (1 shared connections)
- [[Community 200]] (1 shared connections)

## Source Files

- `controllers/clustercollector/deployment.go`
- `controllers/clustercollector/hpa.go`
- `controllers/common/deployment.go`
- `controllers/common/hash.go`

## Audit Trail

- EXTRACTED: 48 (79%)
- INFERRED: 13 (21%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*