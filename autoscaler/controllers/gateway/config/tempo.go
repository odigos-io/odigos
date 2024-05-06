package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	tempoUrlKey = "TEMPO_URL"
)

type Tempo struct{}

func (t *Tempo) DestType() common.DestinationType {
	return common.TempoDestinationType
}

func (t *Tempo) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	url, exists := dest.Spec.Data[tempoUrlKey]
	if !exists {
		return errors.New("Tempo url not specified, gateway will not be configured for Tempo")
	}

	if strings.HasPrefix(url, "https://") {
		return errors.New("Tempo does not currently supports tls export, gateway will not be configured for Tempo")
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
			Exporters: []string{tempoExporterName},
		}
	}

	return nil
}
