package config

import "strings"

// DestSecretEnvPrefix returns the envFrom prefix used when mounting a Destination's
// Secret into the gateway. Secret keys stay as field names (e.g. DATADOG_API_KEY);
// the process env var becomes prefix + key so two destinations of the same type
// cannot collide.
//
// Dynamic destinations keep unprefixed envFrom so user-authored ${ENV} names in
// raw exporter YAML continue to work.
func DestSecretEnvPrefix(destID string) string {
	return "ODIGOS_DEST_" + envSafeDestinationID(destID) + "_"
}

// envSafeDestinationID maps a Destination metadata.name to a C_IDENTIFIER fragment.
// Destination names are DNS subdomains (odigos.io.dest.<type>-<suffix>) so they carry
// dots and dashes, and neither is legal in an environment variable name: the collector's
// confmap env provider fails config resolution for any ${VAR} whose name does not match
// ^[a-zA-Z_][a-zA-Z0-9_]*$, and envFrom.prefix must be a C_IDENTIFIER on clusters without
// KEP-4369 (relaxed env var names). Callers always prepend a literal prefix, so a
// destination name starting with a digit still yields a valid identifier.
func envSafeDestinationID(destID string) string {
	if destID == "" {
		return "unknown"
	}
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, destID)
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

// SanitizeDestinationID maps a Destination metadata.name to a fragment safe for
// DNS-1123 k8s object names, such as the ClickHouse CA volume name built in
// autoscaler/k8sconfig/clickhouse.go. Destination names use dots
// (odigos.io.dest.<type>-<suffix>) and dots are not allowed in a volume name, so
// they become dashes. This is not usable as an env var name — see envSafeDestinationID.
func SanitizeDestinationID(destID string) string {
	if destID == "" {
		return "unknown"
	}
	return strings.ReplaceAll(destID, ".", "-")
}
