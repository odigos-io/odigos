package config

import (
	"fmt"
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"strings"
)

const (
	lokiUrlKey = "LOKI_URL"
)

type Loki struct{}

func (l *Loki) DestType() common.DestinationType {
	return common.LokiDestinationType
}

func (l *Loki) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	if url, exists := dest.Spec.Data[lokiUrlKey]; exists && isLoggingEnabled(dest) {
		url := addProtocol(url)
		lokiExporterName := "loki/loki"
		currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s", url),
			"labels": commonconf.GenericMap{
				"attributes": commonconf.GenericMap{
					"k8s.container.name": "k8s_container_name",
					"k8s.pod.name":       "k8s_pod_name",
					"k8s.namespace.name": "k8s_namespace_name",
				},
			},
		}

		currentConfig.Service.Pipelines["logs/loki"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{lokiExporterName},
		}
	}
}
