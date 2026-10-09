package metricshandler

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// IsOwnedByOdigos reports whether the given APIService was registered by Odigos.
// v1beta1.custom.metrics.k8s.io is a cluster-scoped singleton that may be created
// by another tool (e.g. Prometheus Adapter, KEDA). We identify Odigos ownership
// by checking that the service reference points at an Odigos webhook service
// (autoscaler historically, instrumentor after the merge).
func IsOwnedByOdigos(apiSvc *apiregv1.APIService) bool {
	if apiSvc.Spec.Service == nil {
		return false
	}
	switch apiSvc.Spec.Service.Name {
	case k8sconsts.AutoScalerWebhookServiceName, k8sconsts.InstrumentorServiceName:
		return true
	default:
		return false
	}
}
