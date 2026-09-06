package metrics

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func runtimeMetricsDistro(supported bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		AgentMetrics: &distro.AgentMetrics{
			RuntimeMetrics: &distro.RuntimeMetrics{Supported: supported},
		},
	}
}

func TestDistroSupportsAgentRuntimeMetrics(t *testing.T) {
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
			name:     "agent metrics without runtime metrics",
			distro:   &distro.OtelDistro{AgentMetrics: &distro.AgentMetrics{}},
			expected: false,
		},
		{
			name:     "runtime metrics not supported",
			distro:   runtimeMetricsDistro(false),
			expected: false,
		},
		{
			name:     "runtime metrics supported",
			distro:   runtimeMetricsDistro(true),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, DistroSupportsAgentRuntimeMetrics(tt.distro))
		})
	}
}

// runtime metrics are only collected when both the distro supports them and the
// user configured them. the user configuration is passed to the agent as is.
func TestCalculateAgentRuntimeMetricsConfig(t *testing.T) {
	t.Parallel()

	javaDisabled := true
	runtimeMetrics := &common.MetricsSourceAgentRuntimeMetricsConfiguration{
		Java: &common.MetricsSourceAgentJavaRuntimeMetricsConfiguration{
			Disabled: &javaDisabled,
			Metrics: []common.MetricsSourceAgentRuntimeMetricConfiguration{
				{Name: "jvm.class.loaded"},
			},
		},
	}
	configWithRuntimeMetrics := &common.OdigosConfiguration{
		MetricsSources: &common.MetricsSourceConfiguration{
			AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{RuntimeMetrics: runtimeMetrics},
		},
	}

	tests := []struct {
		name     string
		distro   *distro.OtelDistro
		config   *common.OdigosConfiguration
		expected *common.MetricsSourceAgentRuntimeMetricsConfiguration
	}{
		{
			name:     "distro does not support runtime metrics",
			distro:   runtimeMetricsDistro(false),
			config:   configWithRuntimeMetrics,
			expected: nil,
		},
		{
			name:     "no metrics sources configured",
			distro:   runtimeMetricsDistro(true),
			config:   &common.OdigosConfiguration{},
			expected: nil,
		},
		{
			name:     "no agent metrics configured",
			distro:   runtimeMetricsDistro(true),
			config:   &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{}},
			expected: nil,
		},
		{
			name:   "no runtime metrics configured",
			distro: runtimeMetricsDistro(true),
			config: &common.OdigosConfiguration{MetricsSources: &common.MetricsSourceConfiguration{
				AgentMetrics: &common.MetricsSourceAgentMetricsConfiguration{},
			}},
			expected: nil,
		},
		{
			name:     "supported distro with configured runtime metrics",
			distro:   runtimeMetricsDistro(true),
			config:   configWithRuntimeMetrics,
			expected: runtimeMetrics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, CalculateAgentRuntimeMetricsConfig(tt.distro, tt.config))
		})
	}
}
