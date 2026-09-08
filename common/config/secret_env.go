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
	return "ODIGOS_DEST_" + SanitizeDestinationID(destID) + "_"
}

// SecretEnvVarName is the process environment variable name for a secret field
// belonging to a specific destination.
func SecretEnvVarName(fieldName, destID string) string {
	return DestSecretEnvPrefix(destID) + fieldName
}

// SecretEnvPlaceholder returns a collector config placeholder that expands from
// the destination-scoped env var, e.g. ${ODIGOS_DEST_odigos-io-dest-otlp-abc_DATADOG_API_KEY}.
func SecretEnvPlaceholder(fieldName string, dest ExporterConfigurer) string {
	return "${" + SecretEnvVarName(fieldName, dest.GetID()) + "}"
}

// SanitizeDestinationID maps a Destination metadata.name to a fragment safe for
// envFrom.prefix and other k8s identifiers. Destination names use dots
// (odigos.io.dest.<type>-<suffix>); we replace dots with dashes, matching the
// transform already used when mounting destination secrets (e.g. ClickHouse CA
// volumes in autoscaler/k8sconfig/clickhouse.go).
//
// envFrom.prefix must be a C_IDENTIFIER on all supported clusters regardless of
// KEP-4369 (relaxed env var names); this keeps older clusters working without
// relying on secret-key normalization from that KEP.
func SanitizeDestinationID(destID string) string {
	if destID == "" {
		return "unknown"
	}
	return strings.ReplaceAll(destID, ".", "-")
}
