# Odigos tracestate processor (`odigostracestate`)

OpenTelemetry Collector processor that reads the [W3C `tracestate`](https://www.w3.org/TR/trace-context/#tracestate-header) carried on each span and maps Odigos-specific sampling metadata onto span attributes (for exporters and downstream analysis).

## Why `tracestate` exists

Distributed traces cross many services. The trace context (`traceparent` + `tracestate`) is designed so **vendor-specific information can ride along with the trace** without breaking interoperability. Odigos uses a single vendor entry keyed by `odigos` so every hop can keep the same sampling semantics attached to the trace.

## Layers of encoding

### 1. W3C `tracestate` string (outer)

Per the Trace Context spec, `tracestate` is a comma-separated list of `key=value` pairs (each key is a vendor identifier).

Odigos contributes **one** pair:

```text
odigos=<vendor-value>
```

`<vendor-value>` is **opaque to the spec**; Odigos defines its own structure below.

### 2. Odigos vendor value (inner)

The value for the `odigos` key is a **semicolon-separated** list of inner fields:

```text
<field>;<field>;...
```

Each field is `key:value`. The gateway processor understands these keys (see `processor.go`):

| Inner key | Meaning | Example value | Mapped span attribute (when enabled) |
|-----------|---------|-----------------|----------------------------------------|
| `c` | Sampling **category** shorthand | `n` → noisy operation category | `odigos.sampling.category` = `"noise"` |
| `dr.p` | **Trace deciding rule** — configured keep percentage | floating-point string | `odigos.sampling.trace.deciding_rule.keep_percentage` |
| `dr.id` | **Trace deciding rule** — stable rule id | opaque id string | `odigos.sampling.trace.deciding_rule.id` |

Attribute names match `github.com/odigos-io/odigos/common/odigosattributes` (`SamplingCategory`, `SamplingTraceDecidingRuleKeepPercentage`, `SamplingTraceDecidingRuleId`, etc.).

**Example** (illustrative):

```text
tracestate: odigos=c:n;dr.p:12.5;dr.id=abc123,othervendor=1
```

Only the `odigos=...` segment is parsed here; other vendors’ entries are ignored by this processor.

### 3. Processor configuration

Optional `span_sampling_attributes` mirrors the tail-sampling processor: you can disable recording the category or trace-deciding-rule pieces per deployment. See `config.go` and `common/api/sampling` for `SpanSamplingAttributesConfiguration`.

## Where the `odigos` entry is populated

### Head sampling in agents

Odigos evaluates **noisy-operation** rules as **head sampling** inside language agents when the distro supports it:

1. The **instrumentor** builds `AgentTracesConfig.headSampling` from Odigos configuration and `Sampling` CRDs (including optional kubelet health-probe paths). See `instrumentor/controllers/agentenabled/sampling/headsampling.go` and `instrumentor/controllers/agentenabled/signalconfig/traces.go`.
2. That configuration is exposed to workloads through the **`InstrumentationConfig`** CR (`AgentTracesConfig`, field `headSampling`). The API documents that **head sampling applies to root spans** (`instrumentationconfig_types.go`).
3. Supported agents apply those rules **at trace entry** (in-process): when a rule matches the root span, the agent records the decision by encoding it into the **`odigos`** `tracestate` entry so it stays attached to the trace.

If head sampling is **not** supported for a language/distro, the same noisy-operation rules can fall back to **tail sampling** at the collector (`odigostailsamplingprocessor`), which enriches spans with attributes directly rather than relying on agent-side `tracestate`.

### Propagation across services

Once the root span carries `tracestate`, **OpenTelemetry context propagation** forwards it on outbound calls:

- HTTP: `tracestate` header (alongside `traceparent`).
- gRPC and other stacks: propagators serialize the same trace context.

Downstream services continue the trace with the same trace id; **child spans inherit the propagated `tracestate`** until it reaches the Odigos collectors.

## Role in the gateway pipeline

On the **cluster gateway** collector, `odigostracestate` runs in the traces pipeline so telemetry **arriving from node collectors** can still expose Odigos sampling metadata as **regular span attributes**, consistent with tail sampling enrichment and backend queries—without requiring each backend to parse W3C `tracestate` itself.

Typical placement: after resource attributes and user processors, before routing to destinations (see `common/pipelinegen/config_builder.go`).
