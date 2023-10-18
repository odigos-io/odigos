package config

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	axiomDatasetKey = "AXIOM_DATASET"
)

type Axiom struct{}

func (a *Axiom) DestType() common.DestinationType {
	return common.AxiomDestinationType
}

func (a *Axiom) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	dataset, exists := dest.Spec.Data[axiomDatasetKey]
	if !exists {
		dataset = "default"
		ctrl.Log.V(0).Info("Axiom dataset not specified, using default")
	}

	currentConfig.Exporters["otlphttp/axiom"] = commonconf.GenericMap{
		"compression": "gzip",
		"endpoint":    "https://api.axiom.co",
		"headers": commonconf.GenericMap{
			"authorization":   "Bearer ${AXIOM_API_TOKEN}",
			"x-axiom-dataset": dataset,
		},
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/axiom"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/axiom"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/axiom"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/axiom"},
		}
	}
}
