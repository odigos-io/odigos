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

// AbnormalOtlpGrpcEndpoint returns the in-cluster OTLP gRPC host:port for
// the sidecar service consumed by the side-channel trace pipeline on the
// cluster gateway. Resolves via Kubernetes service DNS.
func AbnormalOtlpGrpcEndpoint(namespace string) string {
	return fmt.Sprintf("%s.%s:%d", OdigosAbnormalServiceName, namespace, OdigosAbnormalOTLPPort)
}

// OtlpGrpcDNSEndpoint returns a gRPC client endpoint in dns:/// form for a Kubernetes Service
// (cluster DNS: <service>.<namespace> with port).
func OtlpGrpcDNSEndpoint(serviceName, namespace string, port int) string {
	return fmt.Sprintf("dns:///%s.%s:%d", serviceName, namespace, port)
}
