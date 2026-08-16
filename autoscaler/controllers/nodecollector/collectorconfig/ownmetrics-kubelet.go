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
	ownKubeletK8sAttrName     = "k8sattributes/own-kubelet"
	ownKubeletMetricsPipeline = "metrics/own-kubelet"
)

var ownMetricsKubeletMetricNames = []string{
	"container.cpu.usage",
	"container.memory.usage",
}

// ownMetricsKubeletProcessorConfig keeps odiglet DaemonSet + gateway Deployment cpu/memory.
// kubeletstats does not emit workload owner attrs, so k8sattributes fills those before the filter.
// deviceplugin is intentionally dropped.
func ownMetricsKubeletProcessorConfig(odigosNamespace string) config.GenericMap {
	notOurMetric := make([]string, 0, len(ownMetricsKubeletMetricNames))
	for _, metricName := range ownMetricsKubeletMetricNames {
		notOurMetric = append(notOurMetric, fmt.Sprintf("name != %q", metricName))
	}

	notOurWorkload := fmt.Sprintf(
		"resource.attributes[%q] != %q and resource.attributes[%q] != %q",
		string(semconv.K8SDaemonSetNameKey), k8sconsts.OdigletDaemonSetName,
		string(semconv.K8SDeploymentNameKey), k8sconsts.OdigosClusterCollectorDeploymentName,
	)

	return config.GenericMap{
		ownKubeletK8sAttrName: config.GenericMap{
			"auth_type": "serviceAccount",
			"extract": config.GenericMap{
				"metadata": []string{
					string(semconv.K8SDaemonSetNameKey),
					string(semconv.K8SDeploymentNameKey),
				},
			},
			"pod_association": []config.GenericMap{{
				"sources": []config.GenericMap{{
					"from": "resource_attribute",
					"name": string(semconv.K8SPodUIDKey),
				}},
			}},
		},
		ownKubeletFilterName: config.GenericMap{
			"error_mode": "ignore",
			"metrics": config.GenericMap{
				"metric": []string{
					fmt.Sprintf("resource.attributes[%q] != %q", string(semconv.K8SNamespaceNameKey), odigosNamespace),
					notOurWorkload,
					fmt.Sprintf("resource.attributes[%q] == %q",
						string(semconv.K8SContainerNameKey), k8sconsts.OdigletDevicePluginContainerName),
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
			Processors: []string{ownKubeletK8sAttrName, ownKubeletFilterName},
			Exporters:  []string{odigletMetricsExporterName},
		},
	}
}
