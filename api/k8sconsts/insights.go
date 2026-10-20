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

	// Bundled insights ClickHouse (shared with interrogation profile storage).
	OdigosInsightsClickHouseServiceName = "odigos-insights-clickhouse"
	OdigosInsightsClickHouseNativePort  = 9000
	OdigosInsightsClickHouseUser        = "odigos_insights"
	OdigosInsightsClickHouseDatabase    = "odigos_insights"
	OdigosInsightsClickHouseSecretName  = "odigos-insights-clickhouse"
	OdigosInsightsClickHouseSecretKey   = "password"
	// OdigosInsightsClickHousePasswordEnv is the gateway env var that holds the
	// bundled ClickHouse password (distinct from destination CLICKHOUSE_PASSWORD).
	OdigosInsightsClickHousePasswordEnv = "ODIGOS_INSIGHTS_CLICKHOUSE_PASSWORD"
)

// InsightsHTTPEndpoint returns the base URL of the insights HTTP API in the
// given namespace (Kubernetes cluster DNS: odigos-insights.<namespace>).
func InsightsHTTPEndpoint(namespace string) string {
	return fmt.Sprintf("http://%s.%s:%d", OdigosInsightsServiceName, namespace, OdigosInsightsHTTPPort)
}

// InsightsClickHouseEndpoint returns the native-protocol tcp:// endpoint for the
// bundled insights ClickHouse Service (cluster DNS).
func InsightsClickHouseEndpoint(namespace string) string {
	return fmt.Sprintf("tcp://%s.%s:%d", OdigosInsightsClickHouseServiceName, namespace, OdigosInsightsClickHouseNativePort)
}
