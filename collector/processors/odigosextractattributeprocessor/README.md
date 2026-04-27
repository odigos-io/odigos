# Odigos Extract Attribute Processor

## Overview

The **Odigos Extract Attribute Processor** extracts values embedded inside existing string-valued span
attributes and surfaces them as new, normalized attributes.

Useful when identifiers (study IDs, tenant IDs, request IDs, etc.) are buried inside SQL statements,
JSON blobs, or URL paths, and you want them as first-class span attributes.

Supports **traces** only.

## Status

| Status    |        |
|-----------|--------|
| Stability | alpha  |
| Signals   | traces |

## How it works

The processor holds a list of **extractions**. For every span it walks each extraction independently,
scans the span's string-valued attributes, and on the first regex match writes the captured value to
that extraction's `target`. Misses leave the target untouched. Multiple targets can be populated from
one span.

## Configuration

Each entry in `extractions` is self-contained. It uses **either** a preset pattern (`source` +
`data_format`) **or** a custom `regex`, and always writes to its own `target`. Preset and regex
entries can be mixed freely in the same list.

### Extraction fields

| field         | required | description |
|---------------|----------|-------------|
| `target`      | yes      | Destination span attribute name. Must be unique across entries. |
| `source`      | XOR with `regex` | Literal key to search for, using the pattern selected by `data_format`. |
| `data_format` | with `source` | One of `json` or `url`. |
| `regex`       | XOR with `source` | A Go (RE2) regex with one capture group. |

### Preset formats

- **`json`** -- JSON / SQL key-value pairs: `"key": "value"`, `key = 'value'`, `key:value`, etc.
- **`url`** -- URL path segments: `/key/<value>`.

### Examples

Mixed preset and regex entries:

```yaml
processors:
  odigosextractattribute:
    extractions:
      - source: study_id
        data_format: json
        target: study.id
      - source: studies
        data_format: url
        target: study.id.url
      - regex: 'request_id=([0-9a-f-]+)'
        target: request.id
```

Multiple URL segments from a DICOM path:

```yaml
processors:
  odigosextractattribute:
    extractions:
      - source: studies
        data_format: url
        target: study.id
      - source: series
        data_format: url
        target: series.id
      - source: instances
        data_format: url
        target: instance.id
```

### Validation rules

- `extractions` must not be empty.
- Each entry must have a non-empty `target`, unique across all entries.
- Each entry must set exactly one of `source` or `regex`.
- When `source` is set, `data_format` must be `json` or `url`.
- When `regex` is set, `data_format` and `source` must be empty.

## Development

Regenerate component metadata after editing `metadata.yaml`:

```bash
make generate
```
