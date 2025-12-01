package config

import (
	"fmt"

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
	GetOrderHint() int
}

type GenericMap map[string]interface{}

type Config struct {
	Receivers  GenericMap `json:"receivers,omitempty"`
	Exporters  GenericMap `json:"exporters,omitempty"`
	Processors GenericMap `json:"processors,omitempty"`
	Extensions GenericMap `json:"extensions,omitempty"`
	Connectors GenericMap `json:"connectors,omitempty"`
	Service    Service    `json:"service,omitempty"`
}

type MetricsConfig struct {
	Level   string       `json:"level,omitempty"`
	Readers []GenericMap `json:"readers,omitempty"`
}

type Telemetry struct {
	Metrics  MetricsConfig      `json:"metrics,omitempty"`
	Resource map[string]*string `json:"resource,omitempty"`
}

type Service struct {
	Extensions []string            `json:"extensions,omitempty"`
	Pipelines  map[string]Pipeline `json:"pipelines,omitempty"`
	Telemetry  Telemetry           `json:"telemetry,omitempty"`
}

type Pipeline struct {
	Receivers  []string `json:"receivers,omitempty"`
	Processors []string `json:"processors,omitempty"`
	Exporters  []string `json:"exporters,omitempty"`
}

func MergeConfigs(configDomains map[string]Config) (Config, error) {
	mergedConfig := Config{}
	var err error
	for _, config := range configDomains {
		mergedConfig.Receivers, err = mergeGenericMaps(mergedConfig.Receivers, config.Receivers)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Exporters, err = mergeGenericMaps(mergedConfig.Exporters, config.Exporters)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Processors, err = mergeGenericMaps(mergedConfig.Processors, config.Processors)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Extensions, err = mergeGenericMaps(mergedConfig.Extensions, config.Extensions)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Connectors, err = mergeGenericMaps(mergedConfig.Connectors, config.Connectors)
		if err != nil {
			return Config{}, err
		}

		mergedConfig.Service.Extensions = mergeExtensions(mergedConfig.Service.Extensions, config.Service.Extensions)
		mergedConfig.Service.Pipelines, err = mergePipelines(mergedConfig.Service.Pipelines, config.Service.Pipelines)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Service.Telemetry, err = mergeTelemetry(mergedConfig.Service.Telemetry, config.Service.Telemetry)
		if err != nil {
			return Config{}, err
		}
	}
	return mergedConfig, nil
}

func mergeExtensions(extensions1 []string, extensions2 []string) []string {
	// TODO: check for duplicates and return an error
	return append(extensions1, extensions2...)
}

func mergePipelines(pipelines1 map[string]Pipeline, pipelines2 map[string]Pipeline) (map[string]Pipeline, error) {
	// Create a copy of pipelines1 to avoid modifying the input
	mergedPipelines := make(map[string]Pipeline, len(pipelines1))
	for k, v := range pipelines1 {
		mergedPipelines[k] = v
	}

	// Merge pipelines2
	for k, v := range pipelines2 {
		if _, exists := mergedPipelines[k]; exists {
			return nil, fmt.Errorf("duplicate pipeline %s in configs", k)
		}
		mergedPipelines[k] = v
	}
	return mergedPipelines, nil
}

func mergeMetricsLevel(level1 string, level2 string) (string, error) {
	if level1 != "" && level2 != "" && level1 != level2 {
		return "", fmt.Errorf("service telemetry metrics level is allowed to be set only once")
	}
	if level1 != "" {
		return level1, nil
	} else {
		return level2, nil
	}
}

func mergeTelemetryResource(resource1 map[string]*string, resource2 map[string]*string) map[string]*string {
	if len(resource1) == 0 { // shortcut for common cases
		return resource2
	} else if len(resource2) == 0 {
		return resource1
	}

	mergedResource := map[string]*string{}
	for k, v := range resource1 {
		mergedResource[k] = v
	}
	for k, v := range resource2 {
		mergedResource[k] = v
	}
	return mergedResource
}

func mergeTelemetryReaders(readers1 []GenericMap, readers2 []GenericMap) []GenericMap {
	if len(readers1) == 0 {
		return readers2
	} else if len(readers2) == 0 {
		return readers1
	}
	mergedReaders := make([]GenericMap, 0, len(readers1)+len(readers2))
	mergedReaders = append(mergedReaders, readers1...)
	mergedReaders = append(mergedReaders, readers2...)
	return mergedReaders
}

func mergeTelemetry(telemetry1 Telemetry, telemetry2 Telemetry) (Telemetry, error) {
	level, err := mergeMetricsLevel(telemetry1.Metrics.Level, telemetry2.Metrics.Level)
	if err != nil {
		return Telemetry{}, err
	}

	mergedTelemetry := Telemetry{
		Metrics: MetricsConfig{
			Level:   level,
			Readers: mergeTelemetryReaders(telemetry1.Metrics.Readers, telemetry2.Metrics.Readers),
		},
		Resource: mergeTelemetryResource(telemetry1.Resource, telemetry2.Resource),
	}
	return mergedTelemetry, nil
}

func mergeGenericMaps(maps ...GenericMap) (GenericMap, error) {
	mergedMap := GenericMap{}
	for _, m := range maps {
		for k, v := range m {
			if _, exists := mergedMap[k]; exists {
				return GenericMap{}, fmt.Errorf("duplicate key %s in configs", k)
			}
			mergedMap[k] = v
		}
	}
	return mergedMap, nil
}
