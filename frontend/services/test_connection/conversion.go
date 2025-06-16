package testconnection

import (
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/destinations/config"
	"github.com/odigos-io/odigos/frontend/graph/model"
	corev1 "k8s.io/api/core/v1"
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

func (dc *DestinationConfigurer) GetType() destinations.DestinationType {
	// Convert the string type to destinations.DestinationType
	return destinations.DestinationType(dc.destination.Type)
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

func (dc *DestinationConfigurer) GetSecretRef() *corev1.LocalObjectReference {
	return nil // Since DestinationInput doesn't have a secret reference field, we return nil
}

func ConvertDestinationToConfigurer(destination model.DestinationInput) (config.ExporterConfigurer, error) {
	if destination.Type == "" {
		return nil, errors.New("destination type is required")
	}

	// Additional validation or conversion logic can be added here if needed

	// Return a new instance of DestinationConfigurer which implements ExporterConfigurer
	return &DestinationConfigurer{destination: destination}, nil
}
