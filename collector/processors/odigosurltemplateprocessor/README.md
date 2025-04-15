# UrlTemplate Processor

This processor fills a gap between semantic conventions and real users needs.

According to http semantic conventions for span names: 

```
HTTP span names SHOULD be {method} {target} if there is a (low-cardinality) target available. If there is no (low-cardinality) {target} available, HTTP span names SHOULD be {method}.
```

The target should be a templated string (e.g. not `/user/1234` but `/user/{id}`).
The templated value is sometimes available to instrumentations in server spans where the framework and instrumentation supports such feature, but it is never available in client spans.

To work around this, this processor will attempt to heuristically "guess" a templated value, and fill it in the span name and relevant attribute.

## Mechanism

## Relevant Spans

The following conditions must be met for a span to be considered relevant for this processor:

1. an http span - contains `http.request.method` or `http.method` attribute.
2. the attribute is not already set by instrumentation. e.g. no `http.route` for server spans and no `url.template` for client spans.
3. the url path is recorded on a relevant attribute in the span (`url.path` / `url.full`) or the deprecated attributes (`http.target` / `http.url`).
4. path can be parsed from the relevant attributes.

## Templated Route Attribute

For spans that match the above constraints, the processor will calculate the templated url and set it in the relevant attributes:

- `url.template` - for client spans.
- `http.route` - for server spans.

## Span Name

If the span name equals the method (e.g. "GET"), and the processor is able to calculate a templated route, the span name will be set to `{method} {target}`. Otherwise, the span name will not be modified.

## Templatization

The processor applies a heuristic approach to determine the templated value. It might not always be correct and might leak high cardinality values into span names and low-cardinality attributes.

The templatization process should be monitored and adjusted according to the values observed in the cluster.

### Default Templatization

By default, the processor will split the path to segment (e.g. "/user/1234" -> ["user", "1234"]) and replace the segments with the following rules:

- numbers - `\d+` -> `{id}` (`1234`, `328962358623904`, `0`)
- uuids - `[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}` -> `{id}` (`123e4567-e89b-12d3-a456-426614174000`). They can appear as either prefix or suffix of the segment (for example `/process/PROCESS_123e4567-e89b-12d3-a456-426614174000`)
- hex-encoded strings - `[0-9a-f]{2}([0-9a-f]{2})*` -> `{id}` (`6f2a9cdeab34f01e`)

These default rules will not templatize paths like `/user/john` or `/user/s111222333` which will be copied as is into the span name and attribute with potentially high cardinality.

## Custom Templatization

To address cases not covered by the default templatization, the processor supports custom templatization rules to be set in the configuration.

Example for templatization rules:

```
/user/{user-name}/friends/{friend-id}
```

This rule, when applied to the path `/user/john/friends/1234`, will result in the templated value `/user/{user-name}/friends/{friend-id}`.

To denote a template path segment, use `{}` brackets with name and optional regexp: `{name:regexp}`. name will be used to generate the templated path (e.g `/user/{foo})` will result in this template value when matched against `/user/john`).
