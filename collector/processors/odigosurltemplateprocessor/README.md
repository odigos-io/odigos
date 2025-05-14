# UrlTemplate Processor

> ⚠️ **Warning**: Improper configuration of this processor can result in high cardinality values in span names and attributes.
> This can lead to performance degradation and excessive costs in telemetry backends.
> It is highly recommended to test and monitor templatization results in staging before deploying to production. Use `include`/`exclude` filters and custom regex rules judiciously.

This processor fills a gap between semantic conventions and real users needs.

According to http semantic conventions for span names:

> HTTP span names SHOULD be {method} {target} if there is a (low-cardinality) target available. If there is no (low-cardinality) {target} available, HTTP span names SHOULD be {method}.

The target should be a templated string (e.g. `/user/{id}`, not `/user/1234`).
Templated value is **sometimes** available in server spans where the framework and instrumentation library supports such feature, but it is almost **never** available in client spans.

When the templated path is not collected at instrumentation level, this processor will attempt to heuristically "guess" a templated value, and update span names and relevant attribute accordingly, enhancing the usability of the data for humans and machines.

## Mechanism

### Relevant Spans

The following conditions must be met for a span to be considered relevant for this processor:

0. the span matches any processor "include" or "exclude" filters which can limit the spans to be processed (more info and examples below).
1. an http span - contains `http.request.method` or `http.method` attribute.
2. the attribute is not already set by instrumentation. e.g. no `http.route` for server spans and no `url.template` for client spans.
3. the url path is recorded on a relevant attribute in the span (`url.path` / `url.full`) or the deprecated attributes (`http.target` / `http.url`).
4. path can be parsed from the relevant attributes.

### Templated Route Attribute

For spans that match the above constraints, the processor will calculate the templated url and set it in the relevant semconv attributes:

- `url.template` - for client spans.
- `http.route` - for server spans.

### Span Name

If the span name equals the method (e.g. "GET"), and the processor is able to calculate a templated route, the span name will be set to `{method} {target}`. Otherwise, the span name will not be modified.

## Configuration

Example configuration: (see more details for each option below)

```yaml
processors:
  odigosurltemplateprocessor:

    # when include is set, the span must match at least one of the properties to be processed.
    include:
      k8s_workloads:
        - namespace: "default"
          kind: "deployment" ## or "daemonset" or "statefulset"
          name: "myapp1"
        - namespace: "default"
          kind: "deployment"
          name: "myapp2"

    # when exclude is set, a span that matches the filter properties will be excluded from processing.
    # if a span matches both include and exclude, it will be excluded (exclude takes precedence).exclude:
    exclude:
      k8s_workloads:
        - namespace: "default"
          kind: "deployment"
          name: "noisyapp"

    # This option allows fine-tuning for specific paths to customize what to templatize and what not.
    # The rule looks like this: "/v1/{foo:regex}/bar/{baz}".
    # Each segment part in "{}" denote templatization, and all other segments should match the text exactly.
    # Inside the "{}" you can optionally set the template name and matching regex.
    # The template name is the name that will be used in the span name and attributes (e.g. "/users/{userId}").
    # The regex is optional, and if provided, it will be used to match the segment.
    # If the regex does not match, the rule will be skipped and other rules and templatization will be evaluated.
    # Example: "/v1/{foo:\d+}" will match "/v1/123" producing "/v1/{foo}", but not with "/v1/abc".
    # compatible with golang regexp module https://pkg.go.dev/regexp
    # for performance reasons, avoid using compute-intensive expressions or adding too many values here.
    templatization_rules:
      - "/user/{user-name}/friends/{friend-id:\d+}"

    # list of additional regex patterns that will be used to match and templated matching path segment.
    # It allows users to define their own regex patterns for custom id formats used/observed in their applications.
    # Note that this regexp should catch ids, but avoid catching other static unrelated strings.
    # For example, if you have ids in the system like "ap123" then a regexp that matches "^ap\d+" would be good,
    # but regexp like "^ap" is too permissive and will also catch "/api".
    # compatible with golang regexp module https://pkg.go.dev/regexp
    # for performance reasons, avoid using compute-intensive expressions or adding too many values here.
    custom_ids:
      - regexp: "^inc_\d+$"
        template_name: "incidentId"
```

## Include/Exclude Filters

This processor is powerful and well polished based on real world usage. However, it is not hermetic, and the consequences of a false positive can be high cardinality values in span names and attributes which can lead to performance issues in some backends and is generally not recommended.

To work around this, the processor supports include/exclude configuration options that limit which spans will be enriched with url templatization values.

### Default Mode

When no include/exclude filters are set, the processor will attempt to process all spans that match the relevant span conditions. This is the default mode and is recommended for most users.

### Opt-In (Include Filter)

Safer, more manual mode where users review each source for low cardinality values before being added to the processor.

To use this mode, set the `include` option in the configuration. A span will be processed only if it matches with at least one of the include filters.

If exclude filters are also set, and the span matches with any of the exclude filters, it will be excluded from processing even if it matches with the include filters.

### Opt-Out (Exclude Filter)

Less manual mode where all spans are processed by default, and users can exclude specific sources if they found that the processor is causing high cardinality values later on.

This reduce the chore in reviewing each and every source upfront, and allows a more "reactive" approach where issues are addressed as the are found. It will also automatically include future sources and opt them in without reviewing them, potentially introducing high cardinality values.

## Templatization

The processor applies a heuristic approach to determine the templated value. It might not always be correct and might leak high cardinality values into span names and low-cardinality attributes.

The templatization process should be monitored and adjusted according to the values observed in the cluster.

### Default Templatization

By default, the processor will split the path to segment (e.g. "/user/1234" -> ["user", "1234"]) and replace the segments with the following rules:

- only digits or special characters - ```^[\d_\-!@#$%^&*()=+{}\[\]:;"'<>,.?/\\|`~]+$``` -> `{id}` (`1234`, `123_456`, `0`)
- uuids - `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}` -> `{id}` (`123e4567-e89b-12d3-a456-426614174000`). They can appear as either prefix or suffix of the segment (for example `/process/PROCESS_123e4567-e89b-12d3-a456-42661bd74000`)
- hex-encoded strings - `^(?:[0-9a-fA-F]{2}){8,}$` -> `{id}` (`6f2a9cdeab34f01e`, `6F2A9CDEAB34F01E`)
- long numbers anywhere - `\d{7,}` -> `{id}` (`1234567`, `INC328962358623904`, `sb_12345678901234567890_us`)
- common [ISO-8601](https://en.wikipedia.org/wiki/ISO_8601) date-time formats - `^\d{4}-\d{2}-\d{2}(?:T\d{2}:\d{2}(?::\d{2})?)?(?:Z|[+-]\d{4})?$` -> `{date}` (`2023-10-01T12:00:00+0000`)
- emails - `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$` like `foo@bar.io` -> `{email}`

These default rules will not templatize paths like `/user/john`, `/user/s111222`, `/users/123456_789` which will be copied as is into the span name and attribute with potentially high cardinality.

### Custom Templatization

To address cases not covered by the default templatization, the processor supports custom templatization rules to be set in the configuration.

Example for templatization rules:

```
/user/{user-name}/friends/{friend-id}
```

This rule, when applied to the path `/user/john/friends/1234`, will result in the templated value `/user/{user-name}/friends/{friend-id}`.

To denote a template path segment, use `{}` brackets with name and optional regexp: `{name:regexp}`. name will be used to generate the templated path (e.g `/user/{foo})` will result in this template value when matched against `/user/john`).

### Custom Ids

The default rule will match various common ids as described above. Systems can and do use a variety of ids conventions and formats. if you system is using a custom id that is not matched by the default rules, you can set a custom regexp to match these ids.

For example, if your system uses `id`s in format `id-1234`, you can set the regexp `^id-\d+$` to match this format, so that `/user/id-1234` will be templatized to `/user/{id}`.

Few more examples for ids that will not be catched by default but can be configured with custom regexp:

- `SA_8856_BH` - `^SA_\d{4}_\w{2}$` ("SA_" then 4 digits then "_" then 2 "word characters" ([a-zA-Z0-9_]))
- `prod-api-001` - `^(dev|staging|prod)-[a-z]+-\d{3}$` (limit the first part to dev/staging/prod)
- `backup_20250416_073045` - `^backup_\d{8}_\d{6}$` (Timestamped IDs)
- `v2.3.4-beta` - `^v\d+\.\d+\.\d+(-[a-z]+)?$` (Application Release Tags)
- `svc_auth_xyz123TOKEN` - `^svc_[a-z]+_[a-zA-Z0-9]+$` (Keys)
- `svc-us-west-2-db12` - `^svc-[a-z]{2}-[a-z]+-\d-[a-z0-9]+$` (Multi-region Services)

Few considerations for the custom ids:

- The regexp must match the entire segment value, not just part of it.
- Keep regexp precise and correct so they don't match unrelated values from other endpoints in the same cluster. The value will be evaluated against all un-templated http spans in the pipeline.
- The regexp must be valid and will be evaluated at runtime. If the regexp is invalid, the processor will fail to start.
- Regexp syntax should be compatible with the Go regexp syntax. For more information, see [Go regexp syntax](https://pkg.go.dev/regexp/syntax).
- Avoid using too complex expressions or adding too many custom regexp values, as these will be evaluated very often and can impact performance.
