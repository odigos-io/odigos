package consts

type DbQuerySanitizationPolicy string

const (
	// DbQuerySanitizationPolicySanitized: Only collect sanitized (redacted) DB query payloads, which avoids collecting PII.
	// If the sanitization is not supported, the payload will not be collected.
	DbQuerySanitizationPolicySanitized DbQuerySanitizationPolicy = "sanitized"

	// DbQuerySanitizationPolicyFull: Collect the full (unsanitized) DB query payloads, regardless of redaction.
	// Maximum visibility, with a risk of collecting PII.
	DbQuerySanitizationPolicyFull DbQuerySanitizationPolicy = "full"

	// DbQuerySanitizationPolicySanitizedOrFull: Collect sanitized payloads if possible, and fall back to full if sanitization isn't supported.
	// Good balance between visibility and privacy. prefer collecting sanitized payloads if available,
	// and fall-back to full collection otherwise, so we won't miss any payloads.
	DbQuerySanitizationPolicySanitizedOrFull DbQuerySanitizationPolicy = "sanitized-or-full"
)

func DbQuerySanitizationPolicyPriority(policy DbQuerySanitizationPolicy) uint8 {
	switch policy {
	case DbQuerySanitizationPolicySanitized:
		return 3
	case DbQuerySanitizationPolicySanitizedOrFull:
		return 2
	case DbQuerySanitizationPolicyFull:
		return 1
	}
	return 0
}
