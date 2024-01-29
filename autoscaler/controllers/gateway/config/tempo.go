package config

import (
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	tempoUrlKey = "TEMPO_URL"
)

type Tempo struct{}

func (t *Tempo) DestType() common.DestinationType {
	return common.TempoDestinationType
}

func (t *Tempo) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	url, exists := dest.Spec.Data[tempoUrlKey]
	if !exists {
		log.Log.V(0).Info("Tempo url not specified, gateway will not be configured for Tempo")
		return
	}

	if strings.HasPrefix(url, "https://") {
		log.Log.V(0).Info("Tempo does not currently supports tls export, gateway will not be configured for Tempo")
		return
	}

	if isTracingEnabled(dest) {
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimSuffix(url, ":4317")

		tempoExporterName := "otlp/tempo-" + dest.Name
		currentConfig.Exporters[tempoExporterName] = commonconf.GenericMap{
			"endpoint": fmt.Sprintf("%s:4317", url),
			"tls": commonconf.GenericMap{
				"insecure": true,
			},
		}

		tracesPipelineName := "traces/tempo-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{tempoExporterName},
		}
	}
}
