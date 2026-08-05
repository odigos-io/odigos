package k8sconsts

import (
	"fmt"

	commonconsts "github.com/odigos-io/odigos/common/consts"
)

// UiOtlpGrpcEndpoint returns the OTLP gRPC host:port for the UI Service in the given namespace
// (Kubernetes cluster DNS: ui.<namespace>).
func UiOtlpGrpcEndpoint(namespace string) string {
	return fmt.Sprintf("ui.%s:%d", namespace, commonconsts.OTLPPort)
}

// InsightsOtlpGrpcDNSEndpoint returns the OTLP gRPC endpoint in dns:/// form
// targeting the headless insights Service. The dns:/// resolver returns every
// insights pod IP (not the single ClusterIP VIP), which lets the gateway's
// gRPC client round_robin across all replicas instead of pinning one pod.
func InsightsOtlpGrpcDNSEndpoint(namespace string) string {
	return OtlpGrpcDNSEndpoint(OdigosInsightsHeadlessServiceName, namespace, OdigosInsightsOTLPPort)
}

// OtlpGrpcDNSEndpoint returns a gRPC client endpoint in dns:/// form for a Kubernetes Service
// (cluster DNS: <service>.<namespace> with port).
func OtlpGrpcDNSEndpoint(serviceName, namespace string, port int) string {
	return fmt.Sprintf("dns:///%s.%s:%d", serviceName, namespace, port)
}
