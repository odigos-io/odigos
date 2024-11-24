# **Odigos Conditional Attributes Processor**

## Overview

The **Odigos Conditional Attributes Processor** is a custom OpenTelemetry Collector processor that adds new attributes to spans based on the values of existing span attributes. It allows you to define rules to conditionally add static values, copy values from other attributes, or set a global default when no rules match.

This processor is ideal for enriching spans with categorized or contextual information, improving observability.

## Features

- **Conditional Attribute Processing**: Add attributes based on specific span attribute values.
- **Static Values**: Assign predefined values to new attributes.
- **Dynamic Values**: Copy values from other span attributes.
- **Global Default**: Set a fallback value when no rules apply or an attribute is missing.

# Key Configuration Fields

## Global Options

| Field           | Description                                     | Required | Default   |
|------------------|-------------------------------------------------|----------|-----------|
| `global_default` | The value assigned when no rules match to any value specified in some of the rules.         | Yes       | - |

## Rule Options

| Field               | Description                                                               | Required |
|----------------------|---------------------------------------------------------------------------|----------|
| `attribute_to_check` | The attribute in the span to evaluate.                                   | Yes      |
| `new_attribute_value_configurations`             | A map of potential values to a list of actions to execute when `attribute_to_check` equals value.                | Yes      |

## Value Map Options

Each key in the `new_attribute_value_configurations` map corresponds to a potential value of `attribute_to_check`. The associated value is a list of `Value` objects, each specifying how to process the span.

| Field            | Description                                                                | Required |
|-------------------|----------------------------------------------------------------------------|----------|
| `new_attribute`  | The name of the new attribute to add to the span.                          | Yes      |
| `value`          | A static string value to assign to the `new_attribute`.                   | No       |
| `from_attribute` | The attribute whose value will be copied to the `new_attribute`.          | No       |

# How It Works

1. The processor checks the value of `attribute_to_check` for each span.
2. It matches the value against the keys in `new_attribute_value_configurations`.
3. For each match, it iterates through the list of `new_attribute_value_configurations` objects:
   - Assigns a static value (`value`) to the `new_attribute`.
   - Copies a value from another attribute (`from_attribute`) to the `new_attribute`.
4. If no match is found, assigns the `global_default` to all `new_attribute`s specified by some of the rule.

# Example Configuration
```yaml
processors:
  odigosconditionalattributes:
    global_default: "Unknown"
    rules:
      - attribute_to_check: "instrumentation_scope.name"
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
      - attribute_to_check: "net.host.name"
        new_attribute_value_configurations:
          "coupon":
            - new_attribute: "odigos.sub_category"
              from_attribute: "http.scheme"
```

# Outputs

## Input Span Example 1 (Matching Rules)

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

## Input Span Example 2 (Partial Match)

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

## Input Span Example 3 (No Match)

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
