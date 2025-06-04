package service

import (
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
)

func LocalTrafficOTLPGrpcDataCollectionEndpoint(nodeIP string) string {
	if feature.ServiceInternalTrafficPolicy(feature.GA) {
		return fmt.Sprintf("%s.%s:%d", k8sconsts.OdigosNodeCollectorLocalTrafficServiceName, env.GetCurrentNamespace(), consts.OTLPPort)
	}
	return fmt.Sprintf("%s:%d", nodeIP, consts.OTLPPort)
}

// LocalTrafficOTLPHttpDataCollectionEndpoint returns the endpoint for the OTLP HTTP data collection pod on the same node.
// If the internal traffic policy is enabled, the endpoint will use the service name.
// Otherwise, it will use the node IP.
// The node IP might be passed as explicit IP or as a pattern like "$(NODE_IP)"
// which also requires having the `NODE_IP` environment variable set in the manifest.
// Using a pattern is useful when the target node is not known once calling this function.
func LocalTrafficOTLPHttpDataCollectionEndpoint(nodeIP string) string {
	if feature.ServiceInternalTrafficPolicy(feature.GA) {
		return fmt.Sprintf("http://%s.%s:%d", k8sconsts.OdigosNodeCollectorLocalTrafficServiceName, env.GetCurrentNamespace(), consts.OTLPHttpPort)
	}
	return fmt.Sprintf("http://%s:%d", nodeIP, consts.OTLPHttpPort)
}

// LocalTrafficOTLPHttpDataCollectionEndpoint returns the endpoint for OpAmp server on the same node (odiglet).
// If the internal traffic policy is enabled, the endpoint will use the service name.
// Otherwise, it will use the node IP.
// The node IP might be passed as explicit IP or as a pattern like "$(NODE_IP)"
// which also requires having the `NODE_IP` environment variable set in the manifest.
// Using a pattern is useful when the target node is not known once calling this function.
func LocalTrafficOpAmpOdigletEndpoint(nodeIP string) string {
	if feature.ServiceInternalTrafficPolicy(feature.GA) {
		return fmt.Sprintf("%s.%s:%d", k8sconsts.OdigletLocalTrafficServiceName, env.GetCurrentNamespace(), consts.OpAMPPort)
	}
	return fmt.Sprintf("%s:%d", nodeIP, consts.OpAMPPort)
}
