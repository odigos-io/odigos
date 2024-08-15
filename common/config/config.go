package config

import (
	"github.com/odigos-io/odigos/common"
)

type SignalSpecific interface {
	GetSignals() []common.ObservabilitySignal
}

type ExporterConfigurer interface {
	SignalSpecific
	GetType() common.DestinationType
	// expected to be unique across all instances of exporters used in collector config, [a-zA-Z0-9-_]+
	GetID() string
	GetConfig() map[string]string
}

type ProcessorConfigurer interface {
	SignalSpecific
	GetType() string
	// expected to be unique across all instances of exporters used in collector config, [a-zA-Z0-9-_]+
	GetID() string
	GetConfig() (GenericMap, error)
}

type GenericMap map[string]interface{}

type Config struct {
	Receivers  GenericMap `json:"receivers"`
	Exporters  GenericMap `json:"exporters"`
	Processors GenericMap `json:"processors"`
	Extensions GenericMap `json:"extensions"`
	Connectors GenericMap `json:"connectors"`
	Service    Service    `json:"service"`
}

type Telemetry struct {
	Metrics GenericMap `json:"metrics"`
	Resource map[string]*string `json:"resource"`
}

type Service struct {
	Extensions []string            `json:"extensions"`
	Pipelines  map[string]Pipeline `json:"pipelines"`
	Telemetry Telemetry            `json:"telemetry,omitempty"`
}

type Pipeline struct {
	Receivers  []string `json:"receivers"`
	Processors []string `json:"processors"`
	Exporters  []string `json:"exporters"`
}
