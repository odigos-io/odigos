package metrics

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
)

func CalculateRuleMetricsConfig(irls *[]odigosv1.InstrumentationRule) *instrumentationrules.MetricsConfig {
	if irls == nil {
		return nil
	}

	var result *instrumentationrules.MetricsConfig
	for _, irl := range *irls {
		result = mergeMetricsConfig(result, irl.Spec.MetricsConfig)
	}
	return result
}

func mergeMetricsConfig(existing *instrumentationrules.MetricsConfig, incoming *instrumentationrules.MetricsConfig) *instrumentationrules.MetricsConfig {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming.DeepCopy()
	}

	out := existing.DeepCopy()
	if incoming.NetworkMetrics != nil && incoming.NetworkMetrics.Enabled != nil && *incoming.NetworkMetrics.Enabled {
		enabled := true
		out.NetworkMetrics = &instrumentationrules.MetricSignal{Enabled: &enabled}
	}
	if incoming.StatsMetrics != nil && incoming.StatsMetrics.Enabled != nil && *incoming.StatsMetrics.Enabled {
		enabled := true
		out.StatsMetrics = &instrumentationrules.MetricSignal{Enabled: &enabled}
	}
	if !out.AnyEnabled() {
		return nil
	}
	return out
}

func ApplyRuleMetricsConfig(agentMetrics **agentsignalconfig.AgentMetricsConfig, irls *[]odigosv1.InstrumentationRule) {
	ruleMetrics := CalculateRuleMetricsConfig(irls)
	if ruleMetrics == nil {
		return
	}

	if *agentMetrics == nil {
		*agentMetrics = &agentsignalconfig.AgentMetricsConfig{}
	}

	if ruleMetrics.NetworkMetrics != nil {
		(*agentMetrics).NetworkMetrics = ruleMetrics.NetworkMetrics.DeepCopy()
	}
	if ruleMetrics.StatsMetrics != nil {
		(*agentMetrics).StatsMetrics = ruleMetrics.StatsMetrics.DeepCopy()
	}
}
