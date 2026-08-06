package metrics

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
)

// CalculateNetworkMetricsConfig returns the network flow and TCP stats metrics config for a container
// based on its InstrumentationRules. Enablement is per-workload and presence-based: if any matching
// rule sets networkMetrics, metrics are collected (OR semantics). A nil result means network metrics
// are not collected for the container.
func CalculateNetworkMetricsConfig(irls *[]odigosv1.InstrumentationRule) *instrumentationrules.NetworkMetricsConfig {
	if irls == nil {
		return nil
	}

	var result *instrumentationrules.NetworkMetricsConfig
	for _, irl := range *irls {
		result = mergeNetworkMetrics(result, irl.Spec.NetworkMetrics)
	}
	return result
}

func mergeNetworkMetrics(existing, incoming *instrumentationrules.NetworkMetricsConfig) *instrumentationrules.NetworkMetricsConfig {
	if incoming == nil {
		return existing
	}
	return incoming
}
