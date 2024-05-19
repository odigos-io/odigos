package config

import (
	"github.com/odigos-io/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
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
		ctrl.Log.V(0).Info("Axiom dataset not specified, using default")
	}

	axiomExporterName := "otlphttp/axiom-" + dest.GetName()
	currentConfig.Exporters[axiomExporterName] = GenericMap{
		"compression": "gzip",
		"endpoint":    "https://api.axiom.co",
		"headers": GenericMap{
			"authorization":   "Bearer ${AXIOM_API_TOKEN}",
			"x-axiom-dataset": dataset,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/axiom-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/axiom-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	return nil
}
