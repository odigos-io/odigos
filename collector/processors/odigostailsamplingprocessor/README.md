# Odigos Tail Sampling Processor

The **odigostailsampling** processor applies Odigos tail-sampling rules to complete traces. It evaluates traces in three categories (in order): **noise**, **highly relevant**, and **cost reduction**. The first category with a deciding rule wins; later categories are skipped for that trace.

The processor requires the Odigos config extension (`odigos_config_extension`) for highly relevant and cost reduction rules. Noisy-operation rules are loaded per source from that extension as well.

When `dry_run` is enabled, traces are never dropped, but metrics and optional span attributes still reflect the decisions that would apply.

## Configuration

| Field | Description |
| ----- | ----------- |
| `odigos_config_extension` | **Required.** Collector extension that provides per-source tail-sampling configuration. |
| `dry_run` | If `true`, log and measure sampling decisions without dropping traces. |
| `span_sampling_attributes` | Optional span attributes written when a category matches (see `common/api/sampling`). |
| `tail_sampling` | Processor-level tail sampling settings. |

## Internal metrics

The processor emits OpenTelemetry metrics via the collector’s internal telemetry pipeline. In Prometheus and similar backends, metric names are prefixed with `otelcol_` (for example `otelcol_odigos.sampling.trace.check_count`).

All metrics are monotonic counters with **development** stability.

Metrics are recorded at three granularities: **general** (no category or rule labels), **per rule**, and **per category**. Per-rule and per-category data points include `odigos.sampling.category` (`noise`, `highly relevant`, or `cost reduction`). When `dry_run` is enabled in config, `odigos.sampling.dry_run=true` is added to per-rule and per-category data points only.

### General metrics

Emitted once per trace that enters tail sampling (after prerequisites pass), before any category is evaluated. **No metric attributes** are set — use these counters for overall tail-sampling volume.

| Metric | Unit | When incremented |
| ------ | ---- | ---------------- |
| `odigos.sampling.trace.check_count` | `{traces}` | `+1` per trace when tail sampling evaluation starts. |
| `odigos.sampling.span.check_count` | `{spans}` | By span count in the trace when tail sampling evaluation starts. |

**Not emitted at general scope:** `span.match_count`, `span.drop_count`, `span.keep_count`, `trace.match_count`, `trace.drop_count`, `trace.keep_count`.

### Per-rule metrics

Emitted during rule evaluation for **each rule** in a category. Labels identify the rule.

**Attributes:** `odigos.sampling.category`, `odigos.sampling.rule.id`, `odigos.sampling.rule.name`, and optionally `odigos.sampling.rule.disabled` (when the rule is disabled) and `odigos.sampling.dry_run`.

| Metric | Unit | When incremented |
| ------ | ---- | ---------------- |
| `odigos.sampling.span.check_count` | `{spans}` | By `SpanCheckedCount` — once per span evaluated against this rule. |
| `odigos.sampling.trace.check_count` | `{traces}` | `+1` per trace for each rule in the category evaluation results (including rules that did not match). |
| `odigos.sampling.trace.match_count` | `{traces}` | `+1` per trace when `SpanMatchedCount > 0` for this rule. |
| `odigos.sampling.span.match_count` | `{spans}` | By `SpanMatchedCount` when `SpanMatchedCount > 0`. |
| `odigos.sampling.trace.drop_count` | `{traces}` | `+1` per trace when the rule matched and `tracePercentage > RulePercentage` (would drop). Recorded even in dry-run mode. |
| `odigos.sampling.trace.keep_count` | `{traces}` | `+1` per trace when the rule matched and `tracePercentage <= RulePercentage` (would keep). Recorded even in dry-run mode. |

**Not emitted per rule:** `span.drop_count`, `span.keep_count`.

### Per-category metrics

Emitted at category scope — either after evaluating all rules in a category, or when that category’s **deciding rule** applies the final sampling decision. Labels do **not** include `rule.id` or `rule.name`.

**Attributes:** `odigos.sampling.category`, and optionally `odigos.sampling.dry_run`.

| Metric | Unit | When incremented |
| ------ | ---- | ---------------- |
| `odigos.sampling.trace.check_count` | `{traces}` | `+1` per trace after all rules in the category are evaluated (even if no rule matched). |
| `odigos.sampling.trace.match_count` | `{traces}` | `+1` per trace when the category’s deciding rule matches and this category makes the final decision. |
| `odigos.sampling.span.match_count` | `{spans}` | By span count in the trace when the category’s deciding rule matches. |
| `odigos.sampling.trace.keep_count` | `{traces}` | `+1` per trace when the category decision **keeps** the trace. |
| `odigos.sampling.span.keep_count` | `{spans}` | By span count in the trace when the category decision **keeps** the trace. |
| `odigos.sampling.trace.drop_count` | `{traces}` | `+1` per trace when the category decision **drops** the trace. |
| `odigos.sampling.span.drop_count` | `{spans}` | By span count in the trace when the category decision **drops** the trace. |

**Not emitted per category:** `span.check_count`.

Per-category keep/drop counters are mutually exclusive for a given trace (one of keep or drop is recorded, not both).

### Quick reference

| Metric | General | Per rule | Per category |
| ------ | :-----: | :------: | :------------: |
| `odigos.sampling.span.check_count` | ✓ | ✓ | |
| `odigos.sampling.span.match_count` | | ✓ | ✓ |
| `odigos.sampling.span.drop_count` | | | ✓ |
| `odigos.sampling.span.keep_count` | | | ✓ |
| `odigos.sampling.trace.check_count` | ✓ | ✓ | ✓ |
| `odigos.sampling.trace.match_count` | | ✓ | ✓ |
| `odigos.sampling.trace.drop_count` | | ✓ | ✓ |
| `odigos.sampling.trace.keep_count` | | ✓ | ✓ |

## Evaluation flow and metrics

For each trace that passes prerequisites:

1. **General metrics** — `recordTraceCheckMetrics` records `trace.check_count` and `span.check_count` with no labels.
2. **Category evaluation** — categories are tried in order until one produces a deciding rule:
   - **Noise** — rules on the root span; deciding rule is the matching rule with the lowest `percentageAtMost`.
   - **Highly relevant** — rules across spans; deciding rule uses `percentageAtLeast`.
   - **Cost reduction** — rules across spans; deciding rule uses `percentageAtMost`.
3. **Per-rule / per-category metrics** — for every category reached, `recordMetrics` emits per-rule data points and a category-level `trace.check_count`. If a deciding rule is found, `recordCategoryMatchMetrics` emits the category decision metrics and later categories are skipped.

Per-rule `trace.drop_count` / `trace.keep_count` reflect what each matching rule would decide. Per-category keep/drop reflect the **final** decision for that trace in the winning category.

Trace keep/drop uses a deterministic value derived from the trace ID (`tracePercentage` in `[0, 100)`), compared to the rule’s configured keep percentage.

## Related documentation

- Auto-generated metric reference (no attribute detail): [documentation.md](./documentation.md)
- Metric definitions for code generation: [metadata.yaml](./metadata.yaml)
- Span attribute keys used on traces (separate from metric labels): `common/odigosattributes/sampling.go`
