# urltemplate

Shared helpers for parsing and matching URL **path rules**.

Shared segment-based path rule parsing and matching, used by URL templatization and HTTP route matching for sampling. It lives under `common` so it has no Kubernetes or collector-specific dependencies.

## Path rule syntax

A rule is a `/`-separated pattern. A leading `/` is optional and ignored when parsing.

| Segment form | Meaning |
|---|---|
| `users` | Static: must equal that path segment exactly |
| `*` | Wildcard: matches any single segment; the segment is **not** rewritten when templatizing (caller must keep cardinality low) |
| `{name}` | Templated: matches any single segment; when templatizing, becomes `{name}` |
| `{}` | Same as templated, with default name `id` |

Examples:

- `/users/{id}` — matches `/users/123`, templatizes to `users/{id}`
- `/v1/*` — matches `/v1/users` or `/v1/admins`, but not `/v1/a/b` (wrong length)
- `/api/{resource}/{resourceId}` — named placeholders for templatization output

Matching is always **one segment per rule segment**. There is no `**` / multi-segment wildcard.

## API

```go
rule, err := urltemplate.ParseUserInputRuleString("/users/{userId}/orders/*", false)
segments, hadLeadingSlash := urltemplate.SplitPath("/users/42/orders/abc") // ["users", "42", "orders", "abc"], true

rule.IsPathSegmentsMatching(segments) // exact, or prefix when PathRule.Prefix is true
```

- `ParseUserInputRuleString` — turn a user-facing rule string into a `PathRule`
- `SplitPath` — split a concrete path into segments and whether it had a leading `/`
- `PathRule.IsPathSegmentsMatching` — exact match, or prefix match when `Prefix` is true

Each `RulePathSegment` is one of: static (`StaticString`), wildcard (`Wildcard`), or template (`TemplateName`).

## Current consumers

- `collector/processors/odigosurltemplateprocessor` — parse custom templatization rules and apply them to paths
- `collector/processors/odigostailsamplingprocessor` — match sampling rules against `http.route` / path / templated path
