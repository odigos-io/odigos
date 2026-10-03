package common

import (
	odigoscommon "github.com/odigos-io/odigos/common"
)

// EffectiveInsightsConfig returns the insights configuration the collectors should be
// wired for. odigos-insights is an enterprise-only component - helm renders its
// Deployment, Services and ClickHouse only when an on-prem token is available - so on
// community tier the flag is ignored. Otherwise the gateway would force the traces
// signal on cluster-wide, install groupbytrace in front of it and export every span to
// a Service that was never deployed. Same reasoning as the tier gate around
// addProfilingGatewayPipeline in the cluster collector's syncConfigMap.
func EffectiveInsightsConfig(insights *odigoscommon.InsightsConfiguration, tier odigoscommon.OdigosTier) *odigoscommon.InsightsConfiguration {
	if !tier.IsEnterprise() {
		return nil
	}
	return insights
}
