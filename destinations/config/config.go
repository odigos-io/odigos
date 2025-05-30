package config

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/destinations"
)

type SignalSpecific interface {
	GetSignals() []common.ObservabilitySignal
}

type ExporterConfigurer interface {
	SignalSpecific
	GetType() destinations.DestinationType
	// expected to be unique across all instances of exporters used in collector config, [a-zA-Z0-9-_]+
	GetID() string
	GetConfig() map[string]string
	GetSecretRef() *corev1.LocalObjectReference
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
	Connectors GenericMap `json:"connectors,omitempty"`
	Service    Service    `json:"service"`
}

type Telemetry struct {
	Metrics  GenericMap         `json:"metrics"`
	Resource map[string]*string `json:"resource"`
}

type Service struct {
	Extensions []string            `json:"extensions"`
	Pipelines  map[string]Pipeline `json:"pipelines"`
	Telemetry  Telemetry           `json:"telemetry,omitempty"`
}

type Pipeline struct {
	Receivers  []string `json:"receivers"`
	Processors []string `json:"processors"`
	Exporters  []string `json:"exporters"`
}

// CollectorSpecConfigurer is an interface that allows destinations to configure
// their collector deployment specifications, such as environment variables and volume mounts.
type CollectorSpecConfigurer interface {
	// GetCollectorSpec returns the collector deployment specifications for this destination
	GetCollectorSpec(dest ExporterConfigurer) *CollectorSpec
}

// CollectorSpec defines the configuration for a collector deployment
type CollectorSpec struct {
	// EnvVars is a list of environment variables to be added to the collector deployment
	EnvVars []corev1.EnvVar

	// VolumeMounts is a list of volume mounts to be added to the collector deployment
	VolumeMounts []corev1.VolumeMount

	// Volumes is a list of volumes to be added to the collector deployment
	Volumes []corev1.Volume
}
