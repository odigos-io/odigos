package collectorconfig

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/config"
	semconv "go.opentelemetry.io/otel/semconv/v1.5.0"
)

const (
	// Pass the destination kubeletstats receiver into own-metrics -> victoria metrics.
	ownKubeletFilterName      = "filter/own-kubelet"
	ownKubeletMetricsPipeline = "metrics/own-kubelet"
)

var ownMetricsKubeletMetricNames = []string{
	"container.cpu.usage",
	"container.memory.usage",
}

// the odigos containers we want resource usage for. deviceplugin is intentionally left out.
var ownMetricsKubeletContainerNames = []string{
	k8sconsts.OdigletContainerName,
	k8sconsts.OdigosNodeCollectorContainerName,
	k8sconsts.OdigosClusterCollectorContainerName,
}

// A processor that filters out most of the kubelet scraped metrics other than the ones in ownMetricsKubeletMetricNames
func ownMetricsKubeletProcessorConfig(odigosNamespace string) config.GenericMap {
	notOurContainer := make([]string, 0, len(ownMetricsKubeletContainerNames))
	for _, containerName := range ownMetricsKubeletContainerNames {
		notOurContainer = append(notOurContainer,
			fmt.Sprintf("resource.attributes[%q] != %q", string(semconv.K8SContainerNameKey), containerName))
	}

	notOurMetric := make([]string, 0, len(ownMetricsKubeletMetricNames))
	for _, metricName := range ownMetricsKubeletMetricNames {
		notOurMetric = append(notOurMetric, fmt.Sprintf("name != %q", metricName))
	}

	return config.GenericMap{
		ownKubeletFilterName: config.GenericMap{
			"error_mode": "ignore",
			"metrics": config.GenericMap{
				"metric": []string{
					fmt.Sprintf("resource.attributes[%q] != %q", string(semconv.K8SNamespaceNameKey), odigosNamespace),
					strings.Join(notOurContainer, " and "),
					strings.Join(notOurMetric, " and "),
				},
			},
		},
	}
}

func ownKubeletPipeline() map[string]config.Pipeline {
	return map[string]config.Pipeline{
		// Reuses the destination metrics kubeletstats receiver.
		ownKubeletMetricsPipeline: {
			Receivers:  []string{kubeletstatsReceiverName},
			Processors: []string{ownKubeletFilterName},
			Exporters:  []string{odigletMetricsExporterName},
		},
	}
}
