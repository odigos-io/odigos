package config

import (
	"errors"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var ErrorHoneycombTracingDisabled = errors.New("attempting to configure Honeycomb tracing, but tracing is disabled")

const (
	honeycombEndpoint = "HONEYCOMB_ENDPOINT"
)

type Honeycomb struct{}

func (h *Honeycomb) DestType() common.DestinationType {
	return common.HoneycombDestinationType
}

func (h *Honeycomb) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	if !isTracingEnabled(dest) {
		return ErrorHoneycombTracingDisabled
	}

	log.Log.V(0).Info("Honeycomb tracing is enabled, configuring Honeycomb destination")

	endpoint, exists := dest.Spec.Data[honeycombEndpoint]
	if !exists {
		endpoint = "api.honeycomb.io"
	}

	exporterName := "otlp/honeycomb-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:443", endpoint),
		"headers": commonconf.GenericMap{
			"x-honeycomb-team": "${HONEYCOMB_API_KEY}",
		},
	}

	tracePipelineName := "traces/honeycomb-" + dest.Name
	currentConfig.Service.Pipelines[tracePipelineName] = commonconf.Pipeline{
		Exporters: []string{exporterName},
	}

	return nil
}
