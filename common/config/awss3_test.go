package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/common"
)

type s3Destination struct {
	mockDestination
	destType common.DestinationType
}

func (d *s3Destination) GetType() common.DestinationType { return d.destType }

func s3TestConfig() *Config {
	return &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
}

func TestS3AWSDefaultsRemainUnchanged(t *testing.T) {
	dest := &s3Destination{mockDestination: mockDestination{id: "aws", config: map[string]string{s3BucketKey: "archive", s3RegionKey: "us-east-1"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}, destType: common.AWSS3DestinationType}
	cfg := s3TestConfig()
	names, err := (&AWSS3{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	assert.Equal(t, []string{"traces/awss3-aws"}, names)
	assert.Equal(t, GenericMap{"marshaler": "otlp_json", "s3uploader": GenericMap{"region": "us-east-1", "s3_bucket": "archive", "s3_partition_format": "year=%Y/month=%m/day=%d/hour=%H/minute=%M"}}, cfg.Exporters["awss3/aws"])
}

func TestSwiftS3SecretsAreDestinationScoped(t *testing.T) {
	cfg := s3TestConfig()
	for _, id := range []string{"swift-a", "swift-b"} {
		dest := &s3Destination{mockDestination: mockDestination{id: id, config: map[string]string{s3BucketKey: "archive", s3RegionKey: "us-east-1", s3EndpointKey: "https://swift.example", s3CAKey: "private-ca", s3CompressionKey: "gzip", s3Marshaller: "otlp_proto"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal}}, destType: common.OpenStackSwiftDestinationType}
		names, err := (&OpenStackSwift{}).ModifyConfig(dest, cfg)
		require.NoError(t, err)
		assert.Len(t, names, 3)
		uploader := cfg.Exporters["awss3/"+id].(GenericMap)["s3uploader"].(GenericMap)
		assert.Equal(t, SecretEnvPlaceholder(s3AccessKey, dest), uploader["access_key_id"])
		assert.Equal(t, SecretEnvPlaceholder(s3SecretKey, dest), uploader["secret_access_key"])
		assert.Equal(t, true, uploader["s3_force_path_style"])
		assert.Equal(t, false, uploader["disable_ssl"])
		assert.Equal(t, "when_required", uploader["request_checksum_calculation"])
		assert.Equal(t, GenericMap{"ca_pem": "private-ca"}, uploader["tls"])
		assert.Equal(t, "gzip", uploader["compression"])
	}
}

func TestSwiftS3InvalidConfigDoesNotMutateCollector(t *testing.T) {
	for _, tc := range []struct{ key, value string }{{s3EndpointKey, ""}, {s3EndpointKey, "swift.example"}, {s3EndpointKey, "ftp://swift.example"}, {s3EndpointKey, "https://user:password@swift.example"}, {s3PathStyleKey, "maybe"}, {s3CompressionKey, "brotli"}, {s3Marshaller, "body"}, {s3BucketKey, ""}, {s3RegionKey, ""}} {
		t.Run(tc.key+tc.value, func(t *testing.T) {
			dest := &s3Destination{mockDestination: mockDestination{id: "swift", config: map[string]string{s3BucketKey: "archive", s3RegionKey: "us-east-1", s3EndpointKey: "https://swift.example"}, signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}}, destType: common.OpenStackSwiftDestinationType}
			dest.config[tc.key] = tc.value
			cfg := s3TestConfig()
			_, err := (&OpenStackSwift{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			assert.Empty(t, cfg.Exporters)
			assert.Empty(t, cfg.Service.Pipelines)
		})
	}
}
