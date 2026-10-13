package config

import (
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDigitalOceanSpacesConfiguration(t *testing.T) {
	for _, id := range []string{"spaces-a", "spaces-b"} {
		dest := &mockGrpcDestination{id: id, config: map[string]string{"S3_BUCKET": "telemetry", "S3_REGION": "us-east-1", "S3_ENDPOINT": "https://nyc3.digitaloceanspaces.com", "S3_FORCE_PATH_STYLE": "true", "S3_MARSHALER": "otlp_proto", "S3_COMPRESSION": "gzip"}}
		cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		pipelines, err := (&DigitalOceanSpaces{}).ModifyConfig(dest, cfg)
		require.NoError(t, err)
		require.Len(t, pipelines, 3)
		exporter := cfg.Exporters["awss3/"+id].(GenericMap)
		require.Equal(t, "otlp_proto", exporter["marshaler"])
		uploader := exporter["s3uploader"].(GenericMap)
		require.Equal(t, SecretEnvPlaceholder("S3_ACCESS_KEY_ID", dest), uploader["access_key_id"])
		require.Equal(t, SecretEnvPlaceholder("S3_SECRET_ACCESS_KEY", dest), uploader["secret_access_key"])
		require.Equal(t, true, uploader["s3_force_path_style"])
		require.Equal(t, true, uploader["checksum_when_required"])
		require.Equal(t, "uuidv7", uploader["unique_key_func_name"])
		require.Equal(t, "gzip", uploader["compression"])
	}
	configers, err := LoadConfigers()
	require.NoError(t, err)
	require.Contains(t, configers, common.DigitalOceanSpacesDestinationType)
}

func TestSpacesRejectsNonOriginEndpoints(t *testing.T) {
	for _, endpoint := range []string{"", "http://nyc3.digitaloceanspaces.com", "https://bucket.nyc3.digitaloceanspaces.com", "https://nyc3.cdn.digitaloceanspaces.com", "https://nyc3-static.digitaloceanspaces.com", "https://example.com", "https://user:pass@nyc3.digitaloceanspaces.com", "https://nyc3.digitaloceanspaces.com/bucket", "https://nyc3.digitaloceanspaces.com?x=y"} {
		t.Run(endpoint, func(t *testing.T) {
			dest := &mockGrpcDestination{id: "spaces", config: map[string]string{"S3_BUCKET": "telemetry", "S3_REGION": "us-east-1", "S3_ENDPOINT": endpoint}}
			cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
			_, err := (&DigitalOceanSpaces{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			require.Empty(t, cfg.Exporters)
		})
	}
}

func TestAWSS3PreservesDefaultCredentialChain(t *testing.T) {
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	dest := &mockGrpcDestination{id: "aws", config: map[string]string{"S3_BUCKET": "telemetry", "S3_REGION": "us-east-1"}}
	_, err := (&AWSS3{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	uploader := cfg.Exporters["awss3/aws"].(GenericMap)["s3uploader"].(GenericMap)
	require.NotContains(t, uploader, "access_key_id")
	require.NotContains(t, uploader, "checksum_when_required")
	require.Equal(t, "year=%Y/month=%m/day=%d/hour=%H/minute=%M", uploader["s3_partition_format"])
}
