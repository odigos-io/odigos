package metrics

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsAgentRuntimeMetrics(d *distro.OtelDistro) bool {
	return d.AgentMetrics != nil &&
		d.AgentMetrics.RuntimeMetrics != nil &&
		d.AgentMetrics.RuntimeMetrics.Supported
}

func CalculateAgentRuntimeMetricsConfig(distro *distro.OtelDistro, effectiveConfig *common.OdigosConfiguration) *common.MetricsSourceAgentRuntimeMetricsConfiguration {

	if !DistroSupportsAgentRuntimeMetrics(distro) {
		return nil
	}

	if effectiveConfig.MetricsSources == nil ||
		effectiveConfig.MetricsSources.AgentMetrics == nil ||
		effectiveConfig.MetricsSources.AgentMetrics.RuntimeMetrics == nil {
		return nil
	}

	return effectiveConfig.MetricsSources.AgentMetrics.RuntimeMetrics
}
