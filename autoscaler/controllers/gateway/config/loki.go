package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
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
		url = strings.TrimSuffix(url, ":3100")
		lokiExporterName := "loki/loki"
		currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:3100/loki/api/v1/push", url),
		}

		currentConfig.Processors["resource"] = commonconf.GenericMap{
			"attributes": []commonconf.GenericMap{
				{
					"key":    "loki.resource.labels",
					"action": "upsert",
					"value":  "k8s.container.name, k8s.pod.name, k8s.namespace.name",
				},
			},
		}

		currentConfig.Service.Pipelines["logs/loki"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"resource", "batch"},
			Exporters:  []string{lokiExporterName},
		}
	}
}
