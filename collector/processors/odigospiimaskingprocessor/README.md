# PII Masking Processor

The `odigospiimasking` processor masks personally identifiable information (PII) in span attributes.

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
