package consts

type DbQueryCollectionPolicy string

const (
	// DbQueryCollectionPolicySanitized: Only collect sanitized (redacted) DB query payloads, which avoids collecting PII.
	// If the sanitization is not supported, the payload will not be collected.
	DbQueryCollectionPolicySanitized DbQueryCollectionPolicy = "sanitized"

	// DbQueryCollectionPolicyFull: Collect the full (unsanitized) DB query payloads, regardless of redaction.
	// Maximum visibility, with a risk of collecting PII.
	DbQueryCollectionPolicyFull DbQueryCollectionPolicy = "full"

	// DbQueryCollectionPolicySanitizedOrFull: Collect sanitized payloads if possible, and fall back to full if sanitization isn't supported.
	// Good balance between visibility and privacy. prefer collecting sanitized payloads if available,
	// and fall-back to full collection otherwise, so we won't miss any payloads.
	DbQueryCollectionPolicySanitizedOrFull DbQueryCollectionPolicy = "sanitized-or-full"
)

// Higher value = higher priority (more restrictive).
var DbQueryCollectionPolicyPriority = map[DbQueryCollectionPolicy]int{
	DbQueryCollectionPolicySanitized:       3,
	DbQueryCollectionPolicySanitizedOrFull: 2,
	DbQueryCollectionPolicyFull:            1,
}
