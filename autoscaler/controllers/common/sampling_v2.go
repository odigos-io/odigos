package common

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// IsSamplingV2Enabled reports whether the sampling v2 feature is active.
// Both conditions must hold:
//   - tail sampling is not globally disabled in the OdigosConfiguration
//   - at least one non-disabled Sampling CR exists in the Odigos namespace
func IsSamplingV2Enabled(ctx context.Context, gateway *odigosv1.CollectorsGroup, c client.Client) bool {
	logger := log.FromContext(ctx)

	if gateway.Spec.TailSampling != nil &&
		gateway.Spec.TailSampling.Disabled != nil &&
		*gateway.Spec.TailSampling.Disabled {
		return false
	}

	samplingList := &odigosv1.SamplingList{}
	if err := c.List(ctx, samplingList, &client.ListOptions{Namespace: env.GetCurrentNamespace()}); err != nil {
		logger.Error(err, "Failed to list Sampling CRs")
		return false
	}

	for _, s := range samplingList.Items {
		if !s.Spec.Disabled {
			return true
		}
	}

	return false
}
