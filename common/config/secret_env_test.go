package config

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeDestinationID(t *testing.T) {
	assert.Equal(t, "odigos-io-dest-otlphttp-abc123", SanitizeDestinationID("odigos.io.dest.otlphttp-abc123"))
	assert.Equal(t, "simple", SanitizeDestinationID("simple"))
	assert.Equal(t, "unknown", SanitizeDestinationID(""))
}

func TestDestSecretEnvPrefix(t *testing.T) {
	assert.Equal(t, "ODIGOS_DEST_odigos_io_dest_otlp_x_", DestSecretEnvPrefix("odigos.io.dest.otlp-x"))
}

// The collector's confmap env provider fails config resolution outright for a ${VAR}
// whose name does not match this pattern, and envFrom.prefix must be a C_IDENTIFIER on
// clusters without KEP-4369. Destination names always carry dots and dashes, so the
// generated names must be checked against the real pattern rather than against
// themselves.
var envVarNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func TestSecretEnvVarNameIsAValidEnvVarName(t *testing.T) {
	destIDs := []string{
		"odigos.io.dest.datadog-abc12",
		"odigos.io.dest.otlphttp-aaaa",
		"odigos.io.dest.simple-trace-db",
		"my.dest-1",
		"",
	}
	for _, destID := range destIDs {
		prefix := DestSecretEnvPrefix(destID)
		assert.Regexp(t, envVarNamePattern, prefix, "envFrom prefix for %q must be a C_IDENTIFIER", destID)
		for _, field := range []string{"DATADOG_API_KEY", "OTLP_HTTP_CLIENT_KEY_PEM", "DSN"} {
			assert.Regexp(t, envVarNamePattern, SecretEnvVarName(field, destID),
				"env var name for field %q of destination %q must be a C_IDENTIFIER", field, destID)
		}
	}
}

func TestDestSecretEnvPrefixIsUniquePerDestination(t *testing.T) {
	assert.NotEqual(t,
		DestSecretEnvPrefix("odigos.io.dest.datadog-aaaa"),
		DestSecretEnvPrefix("odigos.io.dest.datadog-bbbb"))
}

func TestSecretEnvPlaceholder_DistinctPerDestination(t *testing.T) {
	a := &mockGrpcDestination{id: "odigos.io.dest.otlphttp-aaaa", config: map[string]string{}}
	b := &mockGrpcDestination{id: "odigos.io.dest.otlphttp-bbbb", config: map[string]string{}}

	pa := SecretEnvPlaceholder("OTLP_HTTP_CLIENT_KEY_PEM", a)
	pb := SecretEnvPlaceholder("OTLP_HTTP_CLIENT_KEY_PEM", b)

	assert.Equal(t, "${ODIGOS_DEST_odigos_io_dest_otlphttp_aaaa_OTLP_HTTP_CLIENT_KEY_PEM}", pa)
	assert.Equal(t, "${ODIGOS_DEST_odigos_io_dest_otlphttp_bbbb_OTLP_HTTP_CLIENT_KEY_PEM}", pb)
	assert.NotEqual(t, pa, pb)
}

func TestOTLPHttpModifyConfig_NoSecretCollisionAcrossDestinations(t *testing.T) {
	otlp := &OTLPHttp{}
	currentConfig := &Config{
		Exporters:  GenericMap{},
		Extensions: GenericMap{},
		Service:    Service{Pipelines: map[string]Pipeline{}, Extensions: []string{}},
	}

	destA := &mockDestination{
		id: "odigos.io.dest.otlphttp-aaaa",
		config: map[string]string{
			"OTLP_HTTP_ENDPOINT":     "https://vendor-a.example.com",
			"OTLP_HTTP_TLS_ENABLED":  "true",
			"OTLP_HTTP_MTLS_ENABLED": "true",
		},
	}
	destB := &mockDestination{
		id: "odigos.io.dest.otlphttp-bbbb",
		config: map[string]string{
			"OTLP_HTTP_ENDPOINT":     "https://vendor-b.example.com",
			"OTLP_HTTP_TLS_ENABLED":  "true",
			"OTLP_HTTP_MTLS_ENABLED": "true",
		},
	}

	_, err := otlp.ModifyConfig(destA, currentConfig)
	require.NoError(t, err)
	_, err = otlp.ModifyConfig(destB, currentConfig)
	require.NoError(t, err)

	expA := currentConfig.Exporters["otlp_http/generic-odigos.io.dest.otlphttp-aaaa"].(GenericMap)
	expB := currentConfig.Exporters["otlp_http/generic-odigos.io.dest.otlphttp-bbbb"].(GenericMap)
	tlsA := expA["tls"].(GenericMap)
	tlsB := expB["tls"].(GenericMap)

	assert.Equal(t, "https://vendor-a.example.com", expA["endpoint"])
	assert.Equal(t, "https://vendor-b.example.com", expB["endpoint"])
	assert.Equal(t, SecretEnvPlaceholder("OTLP_HTTP_CLIENT_CERT_PEM", destA), tlsA["cert_pem"])
	assert.Equal(t, SecretEnvPlaceholder("OTLP_HTTP_CLIENT_KEY_PEM", destA), tlsA["key_pem"])
	assert.Equal(t, SecretEnvPlaceholder("OTLP_HTTP_CLIENT_CERT_PEM", destB), tlsB["cert_pem"])
	assert.Equal(t, SecretEnvPlaceholder("OTLP_HTTP_CLIENT_KEY_PEM", destB), tlsB["key_pem"])
	assert.NotEqual(t, tlsA["cert_pem"], tlsB["cert_pem"])
	assert.NotEqual(t, tlsA["key_pem"], tlsB["key_pem"])
}
