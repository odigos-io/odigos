# Odigos Extract Attribute Processor

## Overview

The **Odigos Extract Attribute Processor** extracts values embedded inside existing string-valued span
attributes and surfaces them as new, normalized attributes.

Useful when identifiers (study IDs, tenant IDs, request IDs, etc.) are buried inside SQL statements,
JSON blobs, or URL paths, and you want them as first-class span attributes.

Searches the various payload span attributes generated in Odigos.

Supports **traces** only.

## Status

| Status    |        |
|-----------|--------|
| Stability | alpha  |
| Signals   | traces |

## How it works

The processor holds a list of **extractions**. For every span it walks each extraction independently,
scans the span's string-valued attributes, and on the first regex match writes the captured value to
a new span attribute named by that extraction's `target_attribute_name`. Misses leave the span untouched.
Multiple new attributes can be populated from one span.

## Configuration

Each entry in `extractions` is self-contained. It uses **either** a preset pattern (`lookup_key` +
`data_format`) **or** a custom `regex`, and always writes to its own `target_attribute_name`. Preset
and regex entries can be mixed freely in the same list.

### Extraction fields

| field                | required | description |
|----------------------|----------|-------------|
| `target_attribute_name` | yes      | Name of the new span attribute the extracted value will be written to. Must be unique across entries. |
| `lookup_key`         | use this or `regex` | Literal key to search for, using the pattern selected by `data_format`. |
| `data_format`        | use with `lookup_key` | One of `json`, `sql`, or `resource_path`. Points to a pre-set regex which is combined with `lookup_key`. |
| `regex`              | use this or `lookup_key` | A Go (RE2) regex with one capture group, which is the key searched for. |

### Preset formats

- **`json`** -- JSON key-value pairs (colon separator): `"key": "value"`, `key:value`, `{"key": 42}`.
- **`sql`** -- SQL key-value pairs (equals separator): `key = 'value'`, `key=value`, `WHERE key=42`.
- **`resource_path`** -- URL path segments: `/key/<value>`.

### Examples

Mixed preset and regex entries:

```yaml
processors:
  odigosextractattribute:
    extractions:
      - lookup_key: study_id
        data_format: json
        target_attribute_name: study.id
      - lookup_key: studies
        data_format: resource_path
        target_attribute_name: study.id.url
      - regex: 'request_id=([0-9a-f-]+)'
        target_attribute_name: request.id
```

Multiple URL segments from a DICOM path:

```yaml
processors:
  odigosextractattribute:
    extractions:
      - lookup_key: studies
        data_format: resource_path
        target_attribute_name: study.id
      - lookup_key: series
        data_format: resource_path
        target_attribute_name: series.id
      - lookup_key: instances
        data_format: resource_path
        target_attribute_name: instance.id
```

### Validation rules

- `extractions` must not be empty.
- Each entry must have a non-empty `target_attribute_name`, unique across all entries.
- Each entry must set exactly one of `lookup_key` or `regex`.
- When `lookup_key` is set, `data_format` must be `json`, `sql`, or `resource_path`.
- When `regex` is set, `data_format` and `lookup_key` must be empty.

## Development

Regenerate component metadata after editing `metadata.yaml`:

```bash
make generate
```
