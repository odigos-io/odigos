package collectormetrics

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	exporterQueueCapacityMetricName = "otelcol_exporter_queue_capacity"
	exporterQueueSizeMetricName     = "otelcol_exporter_queue_size"
)

// AutoscalerDecision represents the decision made by the autoscaler algorithm
// Positive values indicate that the autoscaler should scale up, negative values
// indicate that the autoscaler should scale down, and zero indicates that the
// autoscaler should not scale.
type AutoscalerDecision int

type AutoscalerAlgorithm interface {
	Decide(ctx context.Context, metrics []MetricFetchResult) AutoscalerDecision
}

type exporterQueueAndBatchQueue struct{}

var ScaleBasedOnExporterQueueAndBatchQueue = &exporterQueueAndBatchQueue{}

// Decide scales based on the exporter queue and batch queue sizes.
// If more than 50% of the pods
func (e *exporterQueueAndBatchQueue) Decide(ctx context.Context, metrics []MetricFetchResult) AutoscalerDecision {
	logger := log.FromContext(ctx)
	currentReplicas := len(metrics)
	for _, podMetrics := range metrics {
		if podMetrics.Error != nil {
			continue
		}

		for metricName, metricFamily := range podMetrics.Metrics {
			if metricName == exporterQueueCapacityMetricName {
				for _, exporterMetric := range metricFamily.Metric {
					if exporterMetric.Gauge != nil {
						logger.V(0).Info("Exporter queue capacity", "value", exporterMetric.Gauge.GetValue())
					}
				}
			} else if metricName == exporterQueueSizeMetricName {
				for _, exporterMetric := range metricFamily.Metric {
					if exporterMetric.Gauge != nil {
						logger.V(0).Info("Exporter queue size", "value", exporterMetric.Gauge.GetValue())
					}
				}
			}
		}
	}

	currentReplicas++
	logger.V(0).Info("Scaling up", "currentReplicas", currentReplicas)
	return AutoscalerDecision(currentReplicas)
}
