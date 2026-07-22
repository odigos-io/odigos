# PII Masking Processor

The `odigospiimasking` processor masks personally identifiable information (PII) in span attributes.

It is similar to the [redact processor](https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/redactprocessor) from OpenTelemetry Collector Contrib, but supports additional PII cases and omits features that are not needed for Odigos.

## Configuration

```yaml
processors:
  odigospiimasking:
    piiCategories:
      - CREDIT_CARD
      - EMAIL
      - JWT
      - UUID
    customFormatMaskings:
      - lookupKey: ssn
        dataFormat: json
    customRegexMaskings:
      - regex: 'secret=([^\s&]+)'
```

| Option | Type | Default | Description |
| --- | --- | --- | --- |
| `piiCategories` | []string | `[]` | Categories of PII to mask. Supported: `CREDIT_CARD`, `EMAIL`, `JWT`, `UUID`. |
| `customFormatMaskings` | []object | `[]` | Format-based masking rules (`lookupKey` + `dataFormat`). |
| `customRegexMaskings` | []object | `[]` | Regex-based masking rules (`regex` with a single capture group). |

## Status

| Status    |        |
|-----------|--------|
| Stability | alpha  |
| Signals   | traces |
