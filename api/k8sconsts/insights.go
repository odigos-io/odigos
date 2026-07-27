package k8sconsts

const (
	OdigosInsightsServiceName = "odigos-insights"
	OdigosInsightsOTLPPort    = 4317

	// OdigosInsightsHeadlessServiceName is a headless (ClusterIP: None)
	// companion Service that fronts the insights pods on the OTLP gRPC port.
	// The cluster gateway targets this name via the dns:/// resolver so it can
	// client-side load balance (round_robin) across all insights replicas,
	// instead of pinning to a single pod behind the regular ClusterIP VIP.
	OdigosInsightsHeadlessServiceName = "odigos-insights-headless"
)
