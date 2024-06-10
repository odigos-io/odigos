package collectormetrics

import (
	"context"
	"flag"

	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	processMemory                  = "otelcol_process_memory_rss"
	exporterQueueSizeMetricName    = "otelcol_exporter_queue_size"
	goMemLimitPercentageMax        = "algo-gomemlimit-percentage-max"
	defaultGoMemLimitPercentageMax = 80.0
	goMemLimitPercentageMin        = "algo-gomemlimit-percentage-min"
	defaultGoMemLimitPercentageMin = 55.0
)

// AutoscalerDecision represents the decision made by the autoscaler algorithm
// Positive values indicate that the autoscaler should scale up, negative values
// indicate that the autoscaler should scale down, and zero indicates that the
// autoscaler should not scale.
type AutoscalerDecision int

type AutoscalerAlgorithm interface {
	Decide(ctx context.Context, metrics []MetricFetchResult, config *odigosv1.OdigosConfiguration) AutoscalerDecision
}

type memoryAndExporterRetries struct {
	goMemLimitPercentageMax float64
	goMemLimitPercentageMin float64
}

var ScaleBasedOnMemoryAndExporterRetries = &memoryAndExporterRetries{}

func (e *memoryAndExporterRetries) RegisterFlags() {
	flag.Float64Var(&e.goMemLimitPercentageMax, goMemLimitPercentageMax, defaultGoMemLimitPercentageMax, "Percentage of the memory limit to consider for scaling up")
	flag.Float64Var(&e.goMemLimitPercentageMin, goMemLimitPercentageMin, defaultGoMemLimitPercentageMin, "Percentage of the memory limit to consider for scaling down")
}

// Decide scales based on the exporter queue and batch queue sizes.
// If more than 50% of the pods
func (e *memoryAndExporterRetries) Decide(ctx context.Context, metrics []MetricFetchResult, config *odigosv1.OdigosConfiguration) AutoscalerDecision {
	memCfg := gateway.GetMemoryConfigurations(config)
	maxMemory := float64(memCfg.GomemlimitMiB) * e.goMemLimitPercentageMax / 100.0
	minMemory := float64(memCfg.GomemlimitMiB) * e.goMemLimitPercentageMin / 100.0
	logger := log.FromContext(ctx)
	currentReplicas := len(metrics)

	numberOfRetryingExporters := 0
	totalMemory := 0.0
	for _, podMetrics := range metrics {
		if podMetrics.Error != nil {
			continue
		}

		var podMemory float64
		for _, metricFamily := range podMetrics.Metrics {
			if metricFamily.Name != nil && *metricFamily.Name == processMemory {
				podMemory = metricFamily.Metric[0].Gauge.GetValue()
				logger.V(5).Info("memory", "value", podMemory, "pod", podMetrics.PodName)
				totalMemory += podMemory
			} else if metricFamily.Name != nil && *metricFamily.Name == exporterQueueSizeMetricName {
				for _, metric := range metricFamily.Metric {
					if metric.Gauge != nil {
						queueSize := metric.Gauge.GetValue()
						if queueSize > 0 {
							numberOfRetryingExporters++
						}

						logger.V(5).Info("exporter queue size", "value", queueSize, "pod", podMetrics.PodName)
					}
				}
			}
		}
	}

	if numberOfRetryingExporters > 0 {
		logger.V(0).Info("Exporting are retrying, skipping autoscaling until backend is healthy", "number of exporters", numberOfRetryingExporters)
		return AutoscalerDecision(currentReplicas)
	}

	avgMemory := totalMemory / float64(len(metrics))
	avgMemoryMb := avgMemory / 1024 / 1024
	logger.V(5).Info("avg memory", "value", avgMemoryMb, "max memory", maxMemory, "min memory", minMemory)

	if avgMemoryMb > maxMemory {
		currentReplicas++
	} else if avgMemoryMb < minMemory && currentReplicas > 1 {
		currentReplicas--
	}

	return AutoscalerDecision(currentReplicas)
}
