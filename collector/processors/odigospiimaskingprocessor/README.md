# PII Masking Processor

The `odigospiimasking` processor masks personally identifiable information (PII) in span attributes.

It is similar to the [redact processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/redactprocessor) from OpenTelemetry Collector Contrib, but supports additional PII cases and omits features that are not needed for Odigos.

## Configuration

Per-source rules come from InstrumentationConfig (`workloadCollectorConfig[].piiMasking`) via `odigos_config_extension`:

```yaml
processors:
  odigospiimasking:
    odigos_config_extension: odigosconfigk8s
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `odigos_config_extension` | component ID | required | Extension implementing `OdigosConfigExtension` that supplies per-source PII masking config. |

Supported PII categories: `CREDIT_CARD`, `EMAIL`, `JWT`, `UUID`.

Custom format (`lookup_key` + `data_format`) and regex rules replace only the matched capture group with `****`.

## Status

| Status    |        |
|-----------|--------|
| Stability | alpha  |
| Signals   | traces |
