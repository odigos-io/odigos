package config

import (
	"strings"
	"unicode"
)

// DestSecretEnvPrefix returns the envFrom prefix used when mounting a Destination's
// Secret into the gateway. Secret keys stay as field names (e.g. DATADOG_API_KEY);
// the process env var becomes prefix + key so two destinations of the same type
// cannot collide.
//
// Dynamic destinations keep unprefixed envFrom so user-authored ${ENV} names in
// raw exporter YAML continue to work.
func DestSecretEnvPrefix(destID string) string {
	return "ODIGOS_DEST_" + SanitizeEnvIdent(destID) + "_"
}

// SecretEnvVarName is the process environment variable name for a secret field
// belonging to a specific destination.
func SecretEnvVarName(fieldName, destID string) string {
	return DestSecretEnvPrefix(destID) + fieldName
}

// SecretEnvPlaceholder returns a collector config placeholder that expands from
// the destination-scoped env var, e.g. ${ODIGOS_DEST_odigos_io_dest_otlp_abc_DATADOG_API_KEY}.
func SecretEnvPlaceholder(fieldName string, dest ExporterConfigurer) string {
	return "${" + SecretEnvVarName(fieldName, dest.GetID()) + "}"
}

// SanitizeEnvIdent maps an arbitrary destination ID to a Kubernetes C_IDENTIFIER
// fragment (letters, digits, underscore).
func SanitizeEnvIdent(s string) string {
	if s == "" {
		return "unknown"
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	return b.String()
}
