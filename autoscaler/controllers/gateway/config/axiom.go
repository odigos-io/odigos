package config

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
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

func (a *Axiom) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	dataset, exists := dest.Spec.Data[axiomDatasetKey]
	if !exists {
		dataset = "default"
		ctrl.Log.V(0).Info("Axiom dataset not specified, using default")
	}

	axiomExporterName := "otlphttp/axiom-" + dest.Name
	currentConfig.Exporters[axiomExporterName] = commonconf.GenericMap{
		"compression": "gzip",
		"endpoint":    "https://api.axiom.co",
		"headers": commonconf.GenericMap{
			"authorization":   "Bearer ${AXIOM_API_TOKEN}",
			"x-axiom-dataset": dataset,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/axiom-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/axiom-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{axiomExporterName},
		}
	}

	return nil
}
