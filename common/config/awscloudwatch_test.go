package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newAWSCloudWatchConfig() *Config {
	return &Config{
		Exporters: make(GenericMap),
		Service: Service{
			Pipelines: make(map[string]Pipeline),
		},
	}
}

func newAWSCloudWatchDestination(extraConfig map[string]string) *mockDestination {
	destConfig := map[string]string{
		AWS_CLOUDWATCH_LOG_GROUP_NAME:  "odigos",
		AWS_CLOUDWATCH_LOG_STREAM_NAME: "odigos-stream",
	}
	for key, value := range extraConfig {
		destConfig[key] = value
	}
	return &mockDestination{
		id:      "test-id",
		config:  destConfig,
		signals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
	}
}

func TestAWSCloudWatchNonNumericLogRetentionIsRejected(t *testing.T) {
	for _, badValue := range []string{"forever", "", "30 days"} {
		t.Run("log retention "+badValue, func(t *testing.T) {
			dest := newAWSCloudWatchDestination(map[string]string{AWS_CLOUDWATCH_LOG_RETENTION: badValue})

			currentConfig := newAWSCloudWatchConfig()
			_, err := (&AWSCloudWatch{}).ModifyConfig(dest, currentConfig)
			require.Error(t, err, "a non numeric log retention must fail the destination instead of panicking")
			assert.Contains(t, err.Error(), AWS_CLOUDWATCH_LOG_RETENTION)
			assert.Empty(t, currentConfig.Exporters)
		})
	}
}

func TestAWSCloudWatchLogRetentionIsRenderedAsANumber(t *testing.T) {
	dest := newAWSCloudWatchDestination(map[string]string{AWS_CLOUDWATCH_LOG_RETENTION: "14"})

	currentConfig := newAWSCloudWatchConfig()
	_, err := (&AWSCloudWatch{}).ModifyConfig(dest, currentConfig)
	require.NoError(t, err)

	exporter, ok := currentConfig.Exporters["awscloudwatchlogs/awscloudwatch-test-id"].(GenericMap)
	require.True(t, ok, "expected a cloudwatch logs exporter, got %v", currentConfig.Exporters)
	assert.Equal(t, 14, exporter["log_retention"])
}
