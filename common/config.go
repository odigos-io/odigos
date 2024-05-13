package common

type SignalSpecific interface {
	GetSignals() []ObservabilitySignal
}

type ExporterConfigurer interface {
	SignalSpecific
	GetType() DestinationType
	GetName() string
	GetConfig() map[string]string
}

type ProcessorConfigurer interface {
	SignalSpecific
	GetType() string
	GetName() string
	GetConfig() (GenericMap, error)
}

type GenericMap map[string]interface{}

type Config struct {
	Receivers  GenericMap `json:"receivers"`
	Exporters  GenericMap `json:"exporters"`
	Processors GenericMap `json:"processors"`
	Extensions GenericMap `json:"extensions"`
	Service    Service    `json:"service"`
}

type Service struct {
	Extensions []string            `json:"extensions"`
	Pipelines  map[string]Pipeline `json:"pipelines"`
}

type Pipeline struct {
	Receivers  []string `json:"receivers"`
	Processors []string `json:"processors"`
	Exporters  []string `json:"exporters"`
}
