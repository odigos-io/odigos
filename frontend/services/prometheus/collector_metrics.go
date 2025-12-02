package prometheus

import (
	"context"
	"fmt"
	"time"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type PodRates struct {
	MetricsAcceptedRps float64
	MetricsDroppedRps  float64
	ExporterSuccessRps float64
	ExporterFailedRps  float64
	Window             string
	LastScrape         time.Time
}

func GetOdigletCollectorMetrics(ctx context.Context, api v1.API, namespace string, podNames []string, window string) (map[string]PodRates, error) {
	if api == nil {
		return nil, fmt.Errorf("prometheus API is nil")
	}
	if window == "" {
		window = DefaultMetricsWindow()
	}
	if namespace == "" {
		namespace = env.GetCurrentNamespace()
	}
	if len(podNames) == 0 {
		return map[string]PodRates{}, nil
	}

	podRegex := buildPodRegex(podNames)
	now := time.Now()

	qAccepted := rateSumByPod("otelcol_receiver_accepted_metric_points_total", namespace, podRegex, window)
	qRefused := rateSumByPod("otelcol_receiver_refused_metric_points_total", namespace, podRegex, window)
	qDropped := rateSumByPod("otelcol_receiver_dropped_metric_points_total", namespace, podRegex, window)
	qExpSent := rateSumByPod("otelcol_exporter_sent_metric_points_total", namespace, podRegex, window)
	qExpFailed := rateSumByPod("otelcol_exporter_send_failed_metric_points_total", namespace, podRegex, window)

	accepted, tsAcc, err := queryVector(ctx, api, qAccepted, now)
	if err != nil {
		return nil, err
	}
	refused, tsRef, err := queryVector(ctx, api, qRefused, now)
	if err != nil {
		return nil, err
	}
	dropped, tsDrop, err := queryVector(ctx, api, qDropped, now)
	if err != nil {
		return nil, err
	}
	expSent, tsSent, err := queryVector(ctx, api, qExpSent, now)
	if err != nil {
		return nil, err
	}
	expFailed, tsFail, err := queryVector(ctx, api, qExpFailed, now)
	if err != nil {
		return nil, err
	}

	receiverDropped := refused
	if len(receiverDropped) == 0 {
		receiverDropped = dropped
	}

	result := make(map[string]PodRates, len(podNames))
	for _, pod := range podNames {
		result[pod] = PodRates{
			Window: window,
		}
	}

	lastScrape := maxTime(tsAcc, tsRef, tsDrop, tsSent, tsFail)

	for pod := range result {
		r := result[pod]
		if v, ok := accepted[pod]; ok {
			r.MetricsAcceptedRps = v
		}
		if v, ok := receiverDropped[pod]; ok {
			r.MetricsDroppedRps = v
		}
		if v, ok := expSent[pod]; ok {
			r.ExporterSuccessRps = v
		}
		if v, ok := expFailed[pod]; ok {
			r.ExporterFailedRps = v
		}
		r.LastScrape = lastScrape
		result[pod] = r
	}

	return result, nil
}
