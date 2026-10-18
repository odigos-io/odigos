package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestFirehoseConfig(t *testing.T) {
	dest := newAWSCloudWatchDestination(map[string]string{
		AWS_CLOUDWATCH_REGION:  "us-east-1",
		firehoseRoleARNKey:     "arn:aws:iam::123456789012:role/producer",
		firehoseExternalIDKey:  "tenant",
		AWS_CLOUDWATCH_RAW_LOG: "true",
	})
	cfg := newAWSCloudWatchConfig()
	pipelines, err := (&AWSFirehose{}).ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"logs/firehose-test-id"}, pipelines)
	require.Len(t, cfg.Exporters, 1)
	exporter := cfg.Exporters["awscloudwatchlogs/firehose-test-id"].(GenericMap)
	require.Equal(t, "us-east-1", exporter["region"])
	require.Equal(t, false, exporter["raw_log"])
	require.Equal(t, "tenant", exporter["external_id"])
	require.Equal(t, "arn:aws:iam::123456789012:role/producer", exporter["role_arn"])
	require.Equal(t, []string{"awscloudwatchlogs/firehose-test-id"}, cfg.Service.Pipelines[pipelines[0]].Exporters)
}

func TestFirehoseRejectsInvalidConfiguration(t *testing.T) {
	cases := []struct {
		name    string
		fields  map[string]string
		signals []common.ObservabilitySignal
	}{
		{name: "no signals", signals: []common.ObservabilitySignal{}},
		{name: "traces", signals: []common.ObservabilitySignal{common.TracesObservabilitySignal}},
		{name: "mixed signals", signals: []common.ObservabilitySignal{common.LogsObservabilitySignal, common.MetricsObservabilitySignal}},
		{name: "profiles", signals: []common.ObservabilitySignal{common.ProfilesObservabilitySignal}},
		{name: "missing region", fields: map[string]string{AWS_CLOUDWATCH_REGION: ""}},
		{name: "empty group", fields: map[string]string{AWS_CLOUDWATCH_LOG_GROUP_NAME: " "}},
		{name: "empty stream", fields: map[string]string{AWS_CLOUDWATCH_LOG_STREAM_NAME: ""}},
		{name: "dynamic group", fields: map[string]string{AWS_CLOUDWATCH_LOG_GROUP_NAME: "/odigos/{ServiceName}"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dest := newAWSCloudWatchDestination(map[string]string{AWS_CLOUDWATCH_REGION: "us-east-1"})
			for k, v := range tc.fields {
				dest.config[k] = v
			}
			if tc.signals != nil {
				dest.signals = tc.signals
			}
			cfg := newAWSCloudWatchConfig()
			_, err := (&AWSFirehose{}).ModifyConfig(dest, cfg)
			require.Error(t, err)
			require.Empty(t, cfg.Exporters)
			require.Empty(t, cfg.Service.Pipelines)
		})
	}
}
