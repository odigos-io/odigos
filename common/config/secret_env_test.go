package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSanitizeEnvIdent(t *testing.T) {
	assert.Equal(t, "odigos_io_dest_otlphttp_abc123", SanitizeEnvIdent("odigos.io.dest.otlphttp-abc123"))
	assert.Equal(t, "simple", SanitizeEnvIdent("simple"))
	assert.Equal(t, "unknown", SanitizeEnvIdent(""))
}

func TestDestSecretEnvPrefix(t *testing.T) {
	assert.Equal(t, "ODIGOS_DEST_odigos_io_dest_otlp_x_", DestSecretEnvPrefix("odigos.io.dest.otlp-x"))
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
