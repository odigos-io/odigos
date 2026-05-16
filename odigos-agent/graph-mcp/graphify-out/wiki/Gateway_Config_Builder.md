# Gateway Config Builder

> 19 nodes · cohesion 0.24

## Key Concepts

- **ErrorSamplerHandler** (8 connections) — `controllers/actions/sampling/errorsampler.go`
- **LatencySamplerHandler** (8 connections) — `controllers/actions/sampling/latencysampler.go`
- **ServiceNameSamplerHandler** (8 connections) — `controllers/actions/sampling/servicenamesampler.go`
- **SpanAttributeSamplerHandler** (8 connections) — `controllers/actions/sampling/spanattributesampler.go`
- **.Validate()** (5 connections) — `controllers/actions/sampling/errorsampler.go`
- **.ConvertLegacyToAction()** (5 connections) — `controllers/actions/sampling/errorsampler.go`
- **.GetRuleConfig()** (5 connections) — `controllers/actions/sampling/errorsampler.go`
- **.ValidateRuleConfig()** (5 connections) — `controllers/actions/sampling/errorsampler.go`
- **.GetActionReference()** (4 connections) — `controllers/actions/sampling/errorsampler.go`
- **.GetActionScope()** (4 connections) — `controllers/actions/sampling/errorsampler.go`
- **.IsActionDisabled()** (4 connections) — `controllers/actions/sampling/errorsampler.go`
- **.List()** (4 connections) — `controllers/actions/sampling/errorsampler.go`
- **errorsampler.go** (2 connections) — `controllers/actions/sampling/errorsampler.go`
- **latencysampler.go** (2 connections) — `controllers/actions/sampling/latencysampler.go`
- **servicenamesampler.go** (2 connections) — `controllers/actions/sampling/servicenamesampler.go`
- **ErrorConfig** (2 connections) — `controllers/actions/sampling/errorsampler.go`
- **LatencyConfig** (2 connections) — `controllers/actions/sampling/latencysampler.go`
- **ServiceNameConfig** (2 connections) — `controllers/actions/sampling/servicenamesampler.go`
- **SpanAttributeConfig** (2 connections) — `controllers/actions/sampling/spanattributesampler.go`

## Relationships

- [[Common Logger Service Pipelines]] (82 shared connections)

## Source Files

- `controllers/actions/sampling/errorsampler.go`
- `controllers/actions/sampling/latencysampler.go`
- `controllers/actions/sampling/servicenamesampler.go`
- `controllers/actions/sampling/spanattributesampler.go`

## Audit Trail

- EXTRACTED: 82 (100%)
- INFERRED: 0 (0%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*