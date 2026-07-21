# PII Masking Processor

The `odigospiimasking` processor masks personally identifiable information (PII) in span attributes.

It is similar to the [redact processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/redactprocessor) from OpenTelemetry Collector Contrib, but supports additional PII cases and omits features that are not needed for Odigos.

## Configuration

```yaml
processors:
  odigospiimasking:
    pii_categories:
      - CREDIT_CARD
      - EMAIL
      - JWT
      - UUID
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `pii_categories` | []string | `[]` | Categories of PII to mask. Supported: `CREDIT_CARD`, `EMAIL`, `JWT`, `UUID`. |

## Status

| Status    |        |
|-----------|--------|
| Stability | alpha  |
| Signals   | traces |
