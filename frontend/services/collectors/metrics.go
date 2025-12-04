package collectors

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	metrics "github.com/odigos-io/odigos/frontend/services/metrics"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// GetOdigletPodsWithMetrics returns odiglet pods enriched with per-pod collector metrics from Prometheus.
// If api is nil or queries fail, the pods are returned without metrics.
func GetOdigletPodsWithMetrics(ctx context.Context, api v1.API) ([]*model.PodInfo, error) {
	selector := fmt.Sprintf("%s=%s", k8sconsts.OdigosCollectorRoleLabel, string(k8sconsts.CollectorsRoleNodeCollector))
	pods, err := GetPodsBySelector(ctx, selector)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(pods))
	for _, p := range pods {
		names = append(names, p.Name)
	}
	if len(names) == 0 || api == nil {
		return pods, nil
	}

	ns := env.GetCurrentNamespace()
	ratesByPod, err := metrics.GetDataCollectorContainerMetrics(ctx, api, ns, names, metrics.DefaultMetricsWindow)
	if err != nil {
		return pods, nil
	}

	for _, p := range pods {
		if rates, ok := ratesByPod[p.Name]; ok {
			last := rates.LastScrape.Format(time.RFC3339)
			p.CollectorMetrics = &model.CollectorPodMetrics{
				MetricsAcceptedRps: rates.MetricsAcceptedRps,
				MetricsDroppedRps:  rates.MetricsDroppedRps,
				ExporterSuccessRps: rates.ExporterSuccessRps,
				ExporterFailedRps:  rates.ExporterFailedRps,
				Window:             rates.Window,
				LastScrape:         &last,
			}
		}
	}
	return pods, nil
}
