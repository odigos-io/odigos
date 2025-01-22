package config

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	tempoUrlKey = "TEMPO_URL"
)

var urlPortExistRegex = regexp.MustCompile(`:\d+`)

type Tempo struct{}

func (t *Tempo) DestType() common.DestinationType {
	return common.TempoDestinationType
}

func (t *Tempo) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	url, exists := dest.GetConfig()[tempoUrlKey]
	if !exists {
		return nil, errors.New("Tempo url not specified, gateway will not be configured for Tempo")
	}

	if strings.HasPrefix(url, "https://") {
		return nil, errors.New("Tempo does not currently supports tls export, gateway will not be configured for Tempo")
	}

	var pipelineNames []string
	if isTracingEnabled(dest) {
		url = strings.TrimPrefix(url, "http://")
		endpoint := url

		if !urlPortExistRegex.MatchString(url) {
			endpoint = fmt.Sprintf("%s:4317", url)
		}

		tempoExporterName := "otlp/tempo-" + dest.GetID()
		currentConfig.Exporters[tempoExporterName] = GenericMap{
			"endpoint": endpoint,
			"tls": GenericMap{
				"insecure": true,
			},
		}

		tracesPipelineName := "traces/tempo-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{tempoExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
