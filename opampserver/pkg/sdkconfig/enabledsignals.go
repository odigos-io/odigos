package sdkconfig

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func calcEnabledSignals(ctx context.Context, kubeClient client.Client) (tracesEnabled bool, metricsEnabled bool, err error) {

	// TODO: consider storing this info in collectors group, so it accurately reflects the current state of the node collector
	destinations := &v1alpha1.DestinationList{}
	err = kubeClient.List(ctx, destinations)
	if err != nil {
		return
	}

	tracesEnabled = false
	metricsEnabled = false

	for _, destination := range destinations.Items {
		for _, signal := range destination.Spec.Signals {
			if signal == common.TracesObservabilitySignal {
				tracesEnabled = true
			} else if signal == common.MetricsObservabilitySignal {
				metricsEnabled = true
			}
		}
	}

	return
}
