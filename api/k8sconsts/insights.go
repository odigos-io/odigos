package k8sconsts

import "fmt"

const (
	OdigosInsightsServiceName = "odigos-insights"
	OdigosInsightsOTLPPort    = 4317
	OdigosInsightsHTTPPort    = 8080

	// OdigosInsightsHeadlessServiceName is a headless (ClusterIP: None)
	// companion Service that fronts the insights pods on the OTLP gRPC port.
	// The cluster gateway targets this name via the dns:/// resolver so it can
	// client-side load balance (round_robin) across all insights replicas,
	// instead of pinning to a single pod behind the regular ClusterIP VIP.
	OdigosInsightsHeadlessServiceName = "odigos-insights-headless"
)

// InsightsHTTPEndpoint returns the base URL of the insights HTTP API in the
// given namespace (Kubernetes cluster DNS: odigos-insights.<namespace>).
func InsightsHTTPEndpoint(namespace string) string {
	return fmt.Sprintf("http://%s.%s:%d", OdigosInsightsServiceName, namespace, OdigosInsightsHTTPPort)
}
