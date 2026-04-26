# Odigos Extract Attribute Processor

## Overview

The **Odigos Extract Attribute Processor** is a custom OpenTelemetry Collector processor that extracts values from existing span attributes and surfaces them as new, normalized attributes.

This processor currently supports **traces** only.

## Status

| Status   |          |
|----------|----------|
| Stability| alpha    |
| Signals  | traces   |

## Configuration

The processor scans existing string-valued span attributes for an embedded
`source_attribute` and writes the matched value onto a new attribute named
`target_attribute`.

| field              | required | description                                                                                                                |
|--------------------|----------|----------------------------------------------------------------------------------------------------------------------------|
| `source_attribute` | yes      | Literal name to search for inside string attributes. Matched in JSON/SQL pairs (`"key": "value"`, `key = 'value'`) and URL path segments (`/key/<value>`). |
| `target_attribute` | yes      | Span attribute name the extracted value is written to.                                                                     |

```yaml
processors:
  odigosextractattribute:
    source_attribute: study_id
    target_attribute: study.id
```

## Development

Regenerate component metadata after editing `metadata.yaml`:

```bash
make generate
```
