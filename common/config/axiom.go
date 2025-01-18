package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	axiomDatasetKey = "AXIOM_DATASET"
)

type Axiom struct{}

func (a *Axiom) DestType() common.DestinationType {
	return common.AxiomDestinationType
}

func (a *Axiom) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	dataset, exists := dest.GetConfig()[axiomDatasetKey]
	if !exists {
		dataset = "default"
	}

	axiomExporterName := "otlphttp/axiom-" + dest.GetID()
	currentConfig.Exporters[axiomExporterName] = GenericMap{
		"compression": "gzip",
		"endpoint":    "https://api.axiom.co",
		"headers": GenericMap{
			"authorization":   "Bearer ${AXIOM_API_TOKEN}",
			"x-axiom-dataset": dataset,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/axiom-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/axiom-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	return nil
}
