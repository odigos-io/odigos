package testconnection

import (
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

// Implement the ExporterConfigurer interface
type DestinationConfigurer struct {
	destination model.DestinationInput
}

func (dc *DestinationConfigurer) GetSignals() []common.ObservabilitySignal {
	var signals []common.ObservabilitySignal
	if dc.destination.ExportedSignals.Traces {
		signals = append(signals, common.TracesObservabilitySignal)
	}
	if dc.destination.ExportedSignals.Metrics {
		signals = append(signals, common.MetricsObservabilitySignal)
	}
	if dc.destination.ExportedSignals.Logs {
		signals = append(signals, common.LogsObservabilitySignal)
	}
	return signals
}

func (dc *DestinationConfigurer) GetType() common.DestinationType {
	// Convert the string type to common.DestinationType
	return common.DestinationType(dc.destination.Type)
}

func (dc *DestinationConfigurer) GetID() string {
	// Generate a unique ID for the Exporter, you can base this on the destination name or type
	return dc.destination.Name
}

func (dc *DestinationConfigurer) GetConfig() map[string]string {
	configMap := make(map[string]string)
	for _, field := range dc.destination.Fields {
		configMap[field.Key] = field.Value
	}
	return configMap
}

func ConvertDestinationToConfigurer(destination model.DestinationInput) (config.ExporterConfigurer, error) {

	if destination.Type == "" {
		return nil, errors.New("destination type is required")
	}

	// Additional validation or conversion logic can be added here if needed

	// Return a new instance of DestinationConfigurer which implements ExporterConfigurer
	return &DestinationConfigurer{destination: destination}, nil
}
