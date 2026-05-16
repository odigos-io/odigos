# Instrumentor CRUD Predicates

> 14 nodes · cohesion 0.22

## Key Concepts

- **.CreateInstrumentation()** (6 connections) — `pkg/ebpf/sdks/go.go`
- **obiInstrumentation** (5 connections) — `pkg/ebpf/sdks/obi/obi.go`
- **NewGoInstrumentationFactory()** (5 connections) — `pkg/ebpf/sdks/go.go`
- **GoOtelEbpfSdk** (5 connections) — `pkg/ebpf/sdks/go.go`
- **go.go** (4 connections) — `pkg/ebpf/sdks/go.go`
- **obi.go** (4 connections) — `pkg/ebpf/sdks/obi/obi.go`
- **convertToGoInstrumentationConfig()** (3 connections) — `pkg/ebpf/sdks/go.go`
- **.ApplyConfig()** (3 connections) — `pkg/ebpf/sdks/go.go`
- **NewConfigProvider()** (2 connections) — `pkg/ebpf/configprovider.go`
- **obiConfigForOdigos()** (2 connections) — `pkg/ebpf/sdks/obi/obi.go`
- **OBIInstrumentationFactory** (2 connections) — `pkg/ebpf/sdks/obi/obi.go`
- **GoInstrumentationFactory** (2 connections) — `pkg/ebpf/sdks/go.go`
- **.Close()** (2 connections) — `pkg/ebpf/sdks/go.go`
- **.Load()** (2 connections) — `pkg/ebpf/sdks/go.go`

## Relationships

- [[Odiglet Pod Manager]] (40 shared connections)
- [[Odiglet File Copy]] (4 shared connections)
- [[Source Object Docs]] (2 shared connections)
- [[Odiglet Agent Filesystem Setup]] (1 shared connections)

## Source Files

- `pkg/ebpf/configprovider.go`
- `pkg/ebpf/sdks/go.go`
- `pkg/ebpf/sdks/obi/obi.go`

## Audit Trail

- EXTRACTED: 42 (89%)
- INFERRED: 5 (11%)
- AMBIGUOUS: 0 (0%)

---

*Part of the graphify knowledge wiki. See [[index]] to navigate.*