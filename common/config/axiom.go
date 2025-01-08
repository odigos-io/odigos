package config

import (
	"github.com/odigos-io/odigos/common"
)

const (
	axiomDatasetKey = "AXIOM_DATASET"
)

type Axiom struct{}

// compile time checks
var _ Configer = (*Axiom)(nil)

func (a *Axiom) DestType() common.DestinationType {
	return common.AxiomDestinationType
}

func (a *Axiom) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	dataset, exists := dest.GetConfig()[axiomDatasetKey]
	if !exists {
		dataset = "default"
		// ctrl.Log.V(0).Info("Axiom dataset not specified, using default")
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

	pipelineNames := []string{}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/axiom-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/axiom-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}
