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

	if isTracingEnabled(dest) {
		currentConfig.Exporters["otlphttp/axiomtraces"] = commonconf.GenericMap{
			"compression": "gzip",
			"endpoint":    "https://api.axiom.co/v1/traces",
			"headers": commonconf.GenericMap{
				"authorization":   "Bearer ${AXIOM_API_TOKEN}",
				"x-axiom-dataset": dataset,
			},
		}

		currentConfig.Service.Pipelines["traces/axiom"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/axiomtraces"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Exporters["otlphttp/axiomlogs"] = commonconf.GenericMap{
			"compression": "gzip",
			"endpoint":    "https://api.axiom.co/v1/logs",
			"headers": commonconf.GenericMap{
				"authorization":   "Bearer ${AXIOM_API_TOKEN}",
				"x-axiom-dataset": dataset,
			},
		}

		currentConfig.Service.Pipelines["logs/axiom"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlphttp/axiomlogs"},
		}
	}
}
