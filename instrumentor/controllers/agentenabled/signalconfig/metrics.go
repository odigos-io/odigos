package signalconfig

import (
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
)

func CalculateMetricsConfig(metricsEnabled bool, effectiveConfig *common.OdigosConfiguration, distro *distro.OtelDistro, containerName string) (*odigosv1.AgentMetricsConfig, *odigosv1.ContainerAgentConfig) {
	if !metricsEnabled {
		return nil, nil
	}

	metricsConfig := &odigosv1.AgentMetricsConfig{}

	distroSupportsAgentSpanMetrics := distro.AgentMetrics != nil &&
		distro.AgentMetrics.SpanMetrics != nil &&
		distro.AgentMetrics.SpanMetrics.Supported

	agentSpanMetricsEnabled := effectiveConfig.MetricsSources != nil &&
		effectiveConfig.MetricsSources.AgentMetrics != nil &&
		effectiveConfig.MetricsSources.AgentMetrics.SpanMetrics != nil &&
		effectiveConfig.MetricsSources.AgentMetrics.SpanMetrics.Enabled

	if distro.Name == "java-enterprise" {
		fmt.Println("distroSupportsAgentSpanMetrics", distroSupportsAgentSpanMetrics)
		fmt.Println("agentSpanMetricsEnabled", agentSpanMetricsEnabled)
	}

	if distroSupportsAgentSpanMetrics && agentSpanMetricsEnabled {
		metricsConfig.SpanMetrics = &odigosv1.AgentSpanMetricsConfig{
			Interval:   "123s",
			Dimensions: []string{"foo.bar"},
		}
	}

	return metricsConfig, nil
}
