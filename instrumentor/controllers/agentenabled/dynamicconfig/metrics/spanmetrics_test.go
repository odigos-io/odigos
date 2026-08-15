package metrics

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

// the defaults documented in CalculateAgentSpanMetricsConfig. duplicated here on purpose,
// so a change to the production values shows up as a failing test.
var (
	defaultSpanMetricsIntervalMs = 60000
	defaultSpanMetricsDimensions = []string{
		"http.method",
		"http.request.method",
		"http.status_code",
		"http.response.status_code",
		"http.route",
	}
	defaultSpanMetricsHistogramBucketsMs = []int{2, 4, 6, 8, 10, 50, 100, 200, 400, 800, 1000, 1400, 2000, 5000, 10000, 15000}
)

func spanMetricsDistro(supported bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		AgentMetrics: &distro.AgentMetrics{
			SpanMetrics: &distro.SpanMetrics{Supported: supported},
		},
	}
}

func TestDistroSupportsAgentSpanMetrics(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		distro   *distro.OtelDistro
		expected bool
	}{
		{
			name:     "distro without agent metrics",
			distro:   &distro.OtelDistro{},
			expected: false,
		},
		{
			name:     "agent metrics without span metrics",
			distro:   &distro.OtelDistro{AgentMetrics: &distro.AgentMetrics{}},
			expected: false,
		},
		{
			name:     "span metrics not supported",
			distro:   spanMetricsDistro(false),
			expected: false,
		},
		{
			name:     "span metrics supported",
			distro:   spanMetricsDistro(true),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, DistroSupportsAgentSpanMetrics(tt.distro))
		})
	}
}

func TestAgentSpanMetricsEnabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		config   *common.OdigosConfiguration
		expected bool
	}{
		{
			name:     "no metrics sources",
			config:   &common.OdigosConfiguration{},
			expected: false,
		},
		{
			name:     "no agent metrics",
			config:   &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{}},
			expected: false,
		},
		{
			name: "no agent span metrics",
			config: &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
				AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{},
			}},
			expected: false,
		},
		{
			name: "agent span metrics explicitly disabled",
			config: &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
				AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{
					SpanMetrics: &common.MetricsSourceAgentSpanMetricsConfiguration{Enabled: false},
				},
			}},
			expected: false,
		},
		{
			name: "agent span metrics enabled",
			config: &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
				AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{
					SpanMetrics: &common.MetricsSourceAgentSpanMetricsConfiguration{Enabled: true},
				},
			}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, AgentSpanMetricsEnabled(tt.config))
		})
	}
}

// the agent span metrics config is pushed to the agent over opamp, so the defaults must be
// stable and user supplied durations must be converted to the milliseconds the agent expects.
func TestCalculateAgentSpanMetricsConfig(t *testing.T) {
	t.Parallel()

	t.Run("defaults when span metrics config is not set", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.NotNil(t, spanMetricsConfig)
		require.Equal(t, defaultSpanMetricsIntervalMs, spanMetricsConfig.IntervalMs)
		require.Equal(t, defaultSpanMetricsDimensions, spanMetricsConfig.Dimensions)
		require.Equal(t, defaultSpanMetricsHistogramBucketsMs, spanMetricsConfig.HistogramBucketsMs)
	})

	t.Run("empty interval keeps the default", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: ""},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.Equal(t, defaultSpanMetricsIntervalMs, spanMetricsConfig.IntervalMs)
	})

	t.Run("interval is converted to milliseconds", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: "15s"},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.Equal(t, 15000, spanMetricsConfig.IntervalMs)
	})

	t.Run("unparsable interval disables the agent", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{Interval: "15 seconds"},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, spanMetricsConfig)
		require.NotNil(t, disabledInfo)
		require.Equal(t, "InjectionConflict", string(disabledInfo.AgentEnabledReason))
		require.Contains(t, disabledInfo.AgentEnabledMessage, "failed to parse span metrics interval")
	})

	t.Run("additional dimensions are appended to the defaults", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				AdditionalDimensions: []string{"rpc.method", "custom.dimension"},
			},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.Equal(t, append(append([]string{}, defaultSpanMetricsDimensions...), "rpc.method", "custom.dimension"), spanMetricsConfig.Dimensions)
	})

	t.Run("explicit histogram buckets replace the defaults and are converted to milliseconds", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				ExplicitHistogramBuckets: []string{"100us", "1ms", "2s", "1m"},
			},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		// 100us is rounded down to 0ms, which is what the agent receives.
		require.Equal(t, []int{0, 1, 2000, 60000}, spanMetricsConfig.HistogramBucketsMs)
	})

	t.Run("empty explicit histogram buckets keep the defaults", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				ExplicitHistogramBuckets: []string{},
			},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.Equal(t, defaultSpanMetricsHistogramBucketsMs, spanMetricsConfig.HistogramBucketsMs)
	})

	t.Run("unparsable histogram bucket disables the agent", func(t *testing.T) {
		t.Parallel()

		config := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				ExplicitHistogramBuckets: []string{"1ms", "not-a-duration"},
			},
		}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(config, spanMetricsDistro(true))

		require.Nil(t, spanMetricsConfig)
		require.NotNil(t, disabledInfo)
		require.Equal(t, "InjectionConflict", string(disabledInfo.AgentEnabledReason))
		require.Contains(t, disabledInfo.AgentEnabledMessage, "failed to parse span metrics histogram bucket")
	})

	// the default dimensions slice must not be shared between calls, otherwise one workload's
	// additional dimensions would leak into every other workload's config.
	t.Run("additional dimensions do not leak between calls", func(t *testing.T) {
		t.Parallel()

		withDimensions := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
			SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{
				AdditionalDimensions: []string{"leaked.dimension"},
			},
		}}
		_, disabledInfo := CalculateAgentSpanMetricsConfig(withDimensions, spanMetricsDistro(true))
		require.Nil(t, disabledInfo)

		withoutDimensions := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{}}
		spanMetricsConfig, disabledInfo := CalculateAgentSpanMetricsConfig(withoutDimensions, spanMetricsDistro(true))

		require.Nil(t, disabledInfo)
		require.Equal(t, defaultSpanMetricsDimensions, spanMetricsConfig.Dimensions)
	})
}

// the span metrics mode is embedded in the head sampling config of every instrumented
// container. reporting "all-spans" while the collector cannot compute metrics from
// unsampled spans would keep spans that should have been dropped at the agent.
func TestCalculateSpanMetricsMode(t *testing.T) {
	t.Parallel()

	disabled := true
	notDisabled := false
	allSpans := commonapisampling.SpanMetricsModeAllSpans

	metricsEnabledGroup := func(spanMetrics *common.MetricsSourceSpanMetricsConfiguration) *odigosv1.CollectorsGroup {
		return &odigosv1.CollectorsGroup{
			Spec: odigosv1.CollectorsGroupSpec{
				Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{SpanMetrics: spanMetrics},
			},
			Status: odigosv1.CollectorsGroupStatus{
				ReceiverSignals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
			},
		}
	}

	configWithMode := &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
		SpanMetrics: &common.MetricsSourceSpanMetricsConfiguration{SpanMetricsMode: &allSpans},
	}}

	tests := []struct {
		name                string
		effectiveConfig     *common.OdigosConfiguration
		nodeCollectorsGroup *odigosv1.CollectorsGroup
		expected            commonapisampling.SpanMetricsMode
	}{
		{
			name:                "no node collectors group",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: nil,
			expected:            commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name:                "collectors group without metrics settings",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: &odigosv1.CollectorsGroup{},
			expected:            commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name:                "span metrics explicitly disabled on the collectors group",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: metricsEnabledGroup(&common.MetricsSourceSpanMetricsConfiguration{Disabled: &disabled}),
			expected:            commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name:            "metrics signal is not collected",
			effectiveConfig: configWithMode,
			nodeCollectorsGroup: &odigosv1.CollectorsGroup{
				Spec: odigosv1.CollectorsGroupSpec{
					Metrics: &odigosv1.CollectorsGroupMetricsCollectionSettings{},
				},
				Status: odigosv1.CollectorsGroupStatus{
					ReceiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
				},
			},
			expected: commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name:                "mode not configured",
			effectiveConfig:     &common.OdigosConfiguration{},
			nodeCollectorsGroup: metricsEnabledGroup(nil),
			expected:            commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name: "metrics sources without span metrics config",
			effectiveConfig: &common.OdigosConfiguration{
				MetricsSources: &common.MetricsSourceConfiguration{},
			},
			nodeCollectorsGroup: metricsEnabledGroup(nil),
			expected:            commonapisampling.SpanMetricsModeSampledSpansOnly,
		},
		{
			name:                "configured mode used when span metrics are collected, no span metrics settings",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: metricsEnabledGroup(nil),
			expected:            commonapisampling.SpanMetricsModeAllSpans,
		},
		{
			name:                "configured mode used when span metrics are not explicitly disabled",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: metricsEnabledGroup(&common.MetricsSourceSpanMetricsConfiguration{Disabled: &notDisabled}),
			expected:            commonapisampling.SpanMetricsModeAllSpans,
		},
		{
			name:                "configured mode used when disabled is unset",
			effectiveConfig:     configWithMode,
			nodeCollectorsGroup: metricsEnabledGroup(&common.MetricsSourceSpanMetricsConfiguration{}),
			expected:            commonapisampling.SpanMetricsModeAllSpans,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, CalculateSpanMetricsMode(tt.effectiveConfig, tt.nodeCollectorsGroup))
		})
	}
}

func TestCalculateDryRun(t *testing.T) {
	t.Parallel()

	dryRun := true
	notDryRun := false

	tests := []struct {
		name     string
		config   *common.OdigosConfiguration
		expected bool
	}{
		{
			name:     "no sampling config",
			config:   &common.OdigosConfiguration{},
			expected: false,
		},
		{
			name:     "sampling config without dry run",
			config:   &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{}},
			expected: false,
		},
		{
			name:     "dry run explicitly false",
			config:   &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{DryRun: &notDryRun}},
			expected: false,
		},
		{
			name:     "dry run enabled",
			config:   &common.OdigosConfiguration{Sampling: &common.SamplingConfiguration{DryRun: &dryRun}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, CalculateDryRun(tt.config))
		})
	}
}
