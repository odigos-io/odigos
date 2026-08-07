# **Odigos Conditional Attributes Processor**

## Overview

The **Odigos Conditional Attributes Processor** is a custom OpenTelemetry Collector processor that adds new attributes to spans and metrics based on the values of existing attributes. It allows you to define rules to conditionally add static values, copy values from other attributes, or set a global default when no rules match.

This processor is ideal for enriching telemetry data with categorized or contextual information, improving observability.

## Features

- **Conditional Attribute Processing**: Add attributes based on specific span or metric attribute values.
- **Static Values**: Assign predefined values to new attributes.
- **Dynamic Values**: Copy values from other attributes.
- **Global Default**: Set a fallback value when no rules apply or an attribute is missing.
- **Dual Signal Support**: Process both traces and metrics with separate field configurations.

# Key Configuration Fields

## Global Options

| Field           | Description                                     | Required | Default   |
|------------------|-------------------------------------------------|----------|-----------|
| `global_default` | The value assigned when no rules match to any value specified in some of the rules.         | Yes       | - |

## Rule Options

| Field               | Description                                                               | Required |
|----------------------|---------------------------------------------------------------------------|----------|
| `field_to_check` | The attribute to evaluate for **traces**. For traces, this supports both span attributes and the special value `instrumentation_scope.name` to check the instrumentation scope name.                                   | Yes      |
| `field_to_check_metrics` | The attribute to evaluate for **metrics**. If not specified, the rule will be skipped for metrics processing. Note: `instrumentation_scope.name` is not supported for metrics.                                   | No      |
| `new_attribute_value_configurations`             | A map of potential values to a list of actions to execute when the checked field equals the map key value.                | Yes      |

## Value Map Options

Each key in the `new_attribute_value_configurations` map corresponds to a potential value of the checked field. The associated value is a list of `NewAttributeValueConfiguration` objects, each specifying how to process the telemetry data.

| Field            | Description                                                                | Required |
|-------------------|----------------------------------------------------------------------------|----------|
| `new_attribute`  | The name of the new attribute to add to the span or metric.                          | Yes      |
| `value`          | A static string value to assign to the `new_attribute`.                   | No       |
| `from_field` | The attribute name whose value will be copied to the `new_attribute`. For traces, also supports `instrumentation_scope.name`.          | No       |

**Note**: Either `value` or `from_field` must be specified for each configuration.

# How It Works

## For Traces

1. The processor checks the value of `field_to_check` for each span.
   - If `field_to_check` is `instrumentation_scope.name`, it uses the span's instrumentation scope name.
   - Otherwise, it looks for the attribute in span attributes, scope attributes, or resource attributes (in that order).
2. It matches the value against the keys in `new_attribute_value_configurations`.
3. For each match, it iterates through the list of configurations:
   - Assigns a static value (`value`) to the `new_attribute`.
   - Copies a value from another field (`from_field`) to the `new_attribute`.
4. If no match is found, the `global_default` value will be assigned to all new_attributes defined across any of the rules.

## For Metrics

1. The processor checks if `field_to_check_metrics` is defined for each rule. If not, the rule is skipped for metrics.
2. The processor checks the value of `field_to_check_metrics` for each metric data point.
   - It looks for the attribute in data point attributes or resource attributes (in that order).
3. It matches the value against the keys in `new_attribute_value_configurations`.
4. For each match, it iterates through the list of configurations:
   - Assigns a static value (`value`) to the `new_attribute`.
   - Copies a value from another data point attribute (`from_field`) to the `new_attribute`.
5. If no match is found, the `global_default` value will be assigned to all new_attributes defined across any of the rules.

**Note**: The special `instrumentation_scope.name` field is only supported for traces, not for metrics.

# Example Configuration

## Traces Only
```yaml
processors:
  odigosconditionalattributes:
    global_default: "Unknown"
    rules:
      - field_to_check: "instrumentation_scope.name"
        new_attribute_value_configurations:
          "opentelemetry.instrumentation.flask":
            - new_attribute: "odigos.category"
              value: "flask"
            - new_attribute: "odigos.sub_category"
              value: "biz"
          "io.opentelemetry.tomcat-10.0":
            - new_attribute: "odigos.category"
              value: "tomcat"
            - new_attribute: "odigos.sub_category"
              value: "baz"
      - field_to_check: "net.host.name"
        new_attribute_value_configurations:
          "coupon":
            - new_attribute: "odigos.sub_category"
              from_field: "http.scheme"
```

## Traces and Metrics
```yaml
processors:
  odigosconditionalattributes:
    global_default: "Unknown"
    rules:
      - field_to_check: "instrumentation_scope.name"
        field_to_check_metrics: "span.instrumentation.scope.name"  # For metrics generated from spans
        new_attribute_value_configurations:
          "opentelemetry.instrumentation.flask":
            - new_attribute: "odigos.category"
              value: "flask"
          "io.opentelemetry.tomcat-10.0":
            - new_attribute: "odigos.category"
              value: "tomcat"
      - field_to_check: "http.target"
        field_to_check_metrics: "http.target"  # Apply same rule to both traces and metrics
        new_attribute_value_configurations:
          "/api/users":
            - new_attribute: "odigos.endpoint_type"
              value: "user_management"
          "/api/products":
            - new_attribute: "odigos.endpoint_type"
              value: "product_catalog"
```

# Examples

## Trace Processing

### Input Span Example 1 (Matching Rules)

### Input Span

{
    "instrumentation_scope.name": "opentelemetry.instrumentation.flask",
    "http.scheme": "https"
}

### Output Span

{
    "instrumentation_scope.name": "opentelemetry.instrumentation.flask",
    "http.scheme": "https",
    "odigos.category": "flask",
    "odigos.sub_category": "biz"
}

### Input Span Example 2 (Partial Match)

### Input Span

{
    "net.host.name": "coupon",
    "http.scheme": "https"
}

### Output Span

{
    "net.host.name": "coupon",
    "http.scheme": "https",
    "odigos.category": "Unknown",
    "odigos.sub_category": "https"
}

### Input Span Example 3 (No Match)

### Input Span

{
    "instrumentation_scope.name": "unknown.library",
    "net.host.name": "unknown"
}

### Output Span

{
    "instrumentation_scope.name": "unknown.library",
    "net.host.name": "unknown",
    "odigos.category": "Unknown",
    "odigos.sub_category": "Unknown"
}

## Metric Processing

Given this configuration:

```yaml
processors:
  odigosconditionalattributes:
    global_default: "other"
    rules:
      - field_to_check: "instrumentation_scope.name"
        field_to_check_metrics: "span.instrumentation.scope.name"
        new_attribute_value_configurations:
          "opentelemetry.instrumentation.flask":
            - new_attribute: "odigos.category"
              value: "web_framework"
      - field_to_check: "http.method"
        field_to_check_metrics: "http.method"
        new_attribute_value_configurations:
          "POST":
            - new_attribute: "odigos.operation_type"
              value: "mutation"
          "GET":
            - new_attribute: "odigos.operation_type"
              value: "query"
```

### Input Metric Data Point Example 1 (Matching Rules)

#### Input Data Point Attributes

{
    "span.instrumentation.scope.name": "opentelemetry.instrumentation.flask",
    "http.method": "POST"
}

#### Output Data Point Attributes

{
    "span.instrumentation.scope.name": "opentelemetry.instrumentation.flask",
    "http.method": "POST",
    "odigos.category": "web_framework",
    "odigos.operation_type": "mutation"
}

### Input Metric Data Point Example 2 (No Match)

#### Input Data Point Attributes

{
    "span.instrumentation.scope.name": "unknown.library",
    "http.method": "DELETE"
}

#### Output Data Point Attributes

{
    "span.instrumentation.scope.name": "unknown.library",
    "http.method": "DELETE",
    "odigos.category": "other",
    "odigos.operation_type": "other"
}
