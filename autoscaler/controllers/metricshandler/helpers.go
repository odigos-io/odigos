package metricshandler

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
)

// IsOwnedByOdigos reports whether the given APIService was registered by Odigos.
// v1beta1.custom.metrics.k8s.io is a cluster-scoped singleton that may be created
// by another tool (e.g. Prometheus Adapter, KEDA). We identify Odigos ownership
// by checking that the service reference points to the Odigos autoscaler webhook service.
func IsOwnedByOdigos(apiSvc *apiregv1.APIService) bool {
	return apiSvc.Spec.Service != nil &&
		apiSvc.Spec.Service.Name == k8sconsts.AutoScalerWebhookServiceName
}
