package collectormetrics

import (
	"context"
	"flag"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	exporterQueueCapacityMetricName = "otelcol_exporter_queue_capacity"
	exporterQueueSizeMetricName     = "otelcol_exporter_queue_size"
	processorRefusedSpanMetricName  = "otelcol_processor_refused_spans"

	queueThresholdFlag    = "algo-queue-threshold"
	queueThresholdDefault = 0.7
)

// AutoscalerDecision represents the decision made by the autoscaler algorithm
// Positive values indicate that the autoscaler should scale up, negative values
// indicate that the autoscaler should scale down, and zero indicates that the
// autoscaler should not scale.
type AutoscalerDecision int

type AutoscalerAlgorithm interface {
	Decide(ctx context.Context, metrics []MetricFetchResult) AutoscalerDecision
}

type exporterQueueAndBatchQueue struct {
	queueThreshold float64
}

var ScaleBasedOnExporterQueueAndBatchQueue = &exporterQueueAndBatchQueue{}

func (e *exporterQueueAndBatchQueue) RegisterFlags() {
	flag.Float64Var(&e.queueThreshold, queueThresholdFlag, queueThresholdDefault, "Threshold for exporter queue size")
}

// Decide scales based on the exporter queue and batch queue sizes.
// If more than 50% of the pods
func (e *exporterQueueAndBatchQueue) Decide(ctx context.Context, metrics []MetricFetchResult) AutoscalerDecision {
	logger := log.FromContext(ctx)
	currentReplicas := len(metrics)
	scaleUpPods := 0
	scaleDownPods := 0
	for _, podMetrics := range metrics {
		if podMetrics.Error != nil {
			continue
		}

		refusedByMemoryLimiter := e.getCounterValue(podMetrics, processorRefusedSpanMetricName, "processor", "memory_limiter")
		logger.V(5).Info("refused spans by memory limiter", "value", refusedByMemoryLimiter, "pod", podMetrics.PodName)

		queueSizes := e.groupMetricByLabelKey(podMetrics, exporterQueueSizeMetricName, "exporter")
		logger.V(5).Info("exporter queue sizes", "values", queueSizes, "pod", podMetrics.PodName)

		queueCapacities := e.groupMetricByLabelKey(podMetrics, exporterQueueCapacityMetricName, "exporter")
		logger.V(5).Info("exporter queue capacities", "values", queueCapacities, "pod", podMetrics.PodName)

		slowExporters := e.countSlowExporters(queueSizes, queueCapacities)
		logger.V(5).Info("slow exporters", "count", slowExporters, "pod", podMetrics.PodName)

		if refusedByMemoryLimiter > 0 {
			if slowExporters > 0 {
				logger.V(0).Info("avoiding scaling up collectors because backends are too slow")
			} else {
				scaleUpPods++
			}
		} else {
			if slowExporters > 0 {
				logger.V(0).Info("avoiding scaling down collectors because backends are too slow")
			}
			if currentReplicas > 1 {
				scaleDownPods++
			}
		}
	}

	if scaleUpPods*2 > currentReplicas {
		currentReplicas++
	} else if scaleDownPods*2 > currentReplicas {
		currentReplicas--
	}
	return AutoscalerDecision(currentReplicas)
}

func (e *exporterQueueAndBatchQueue) countSlowExporters(queueSizes map[string]float64, queueCapacities map[string]float64) int {
	result := 0
	for exporter, queueSize := range queueSizes {
		queueCapacity, ok := queueCapacities[exporter]
		if !ok {
			continue
		}

		if e.isExporterTooSlow(queueSize, queueCapacity) {
			result++
		}
	}

	return result
}

func (e *exporterQueueAndBatchQueue) isExporterTooSlow(queueSize float64, queueCapacity float64) bool {
	return queueSize/queueCapacity >= e.queueThreshold
}

func (e *exporterQueueAndBatchQueue) getCounterValue(podMetrics MetricFetchResult, name string, labelKey string, labelValue string) float64 {
	for metricName, metricFamily := range podMetrics.Metrics {
		if metricName == name {
			for _, exporterMetric := range metricFamily.Metric {
				if exporterMetric.Counter != nil {
					for _, labelPair := range exporterMetric.Label {
						if labelPair.Name != nil && *labelPair.Name == labelKey &&
							labelPair.Value != nil && *labelPair.Value == labelValue {
							return exporterMetric.Counter.GetValue()
						}
					}
				}
			}
		}
	}

	return 0
}

func (e *exporterQueueAndBatchQueue) groupMetricByLabelKey(podMetrics MetricFetchResult, name string, labelKey string) map[string]float64 {
	groupedMetrics := make(map[string]float64)
	for metricName, metricFamily := range podMetrics.Metrics {
		if metricName == name {
			for _, exporterMetric := range metricFamily.Metric {
				if exporterMetric.Gauge != nil {
					for _, labelPair := range exporterMetric.Label {
						if labelPair.Name != nil && *labelPair.Name == labelKey {
							groupedMetrics[*labelPair.Value] = exporterMetric.Gauge.GetValue()
						}
					}
				}
			}
		}
	}

	return groupedMetrics
}
