package signalconfig

import (
	"fmt"
	"time"

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
		fmt.Println("=============== distroSupportsAgentSpanMetrics", distroSupportsAgentSpanMetrics)
		fmt.Println("=============== agentSpanMetricsEnabled", agentSpanMetricsEnabled)
	}

	if distroSupportsAgentSpanMetrics && agentSpanMetricsEnabled {
		// TODO: these defaults are duplication of the value written to the
		// collector config in autoscaler.
		// it would be better to consolidate them going forward.
		intervalMs := 60 * 1000 // 60 seconds
		dimensions := []string{
			"http.method",
			"http.request.method",
			"http.status_code",
			"http.response.status_code",
			"http.route",
		}
		if effectiveConfig.MetricsSources.SpanMetrics != nil {
			if effectiveConfig.MetricsSources.SpanMetrics.Interval != "" {
				interval, err := time.ParseDuration(effectiveConfig.MetricsSources.SpanMetrics.Interval)
				if err != nil {
					return nil, &odigosv1.ContainerAgentConfig{
						ContainerName:       containerName,
						AgentEnabled:        false,
						AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
						AgentEnabledMessage: fmt.Sprintf("failed to parse span metrics interval: %s", err),
					}
				}
				intervalMs = int(interval.Milliseconds())
			}
			if effectiveConfig.MetricsSources.SpanMetrics.AdditionalDimensions != nil {
				dimensions = append(dimensions, effectiveConfig.MetricsSources.SpanMetrics.AdditionalDimensions...)
			}
		}
		metricsConfig.SpanMetrics = &odigosv1.AgentSpanMetricsConfig{
			IntervalMs: intervalMs,
			Dimensions: dimensions,
		}
	}

	return metricsConfig, nil
}
