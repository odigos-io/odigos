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

func MergeConfigs(configs ...Config) (Config, error) {
	mergedConfig := Config{}
	var err error
	for _, config := range configs {
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

		mergedConfig.Service.Extensions, err = mergeExtensions(mergedConfig.Service.Extensions, config.Service.Extensions)
		if err != nil {
			return Config{}, err
		}
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

func mergeExtensions(extensions1 []string, extensions2 []string) ([]string, error) {
	// TODO: check for duplicates and return an error
	mergedExtensions := append(extensions1, extensions2...)
	return mergedExtensions, nil
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

func mergeTelemetry(telemetry1 Telemetry, telemetry2 Telemetry) (Telemetry, error) {
	if len(telemetry1.Metrics) > 0 && len(telemetry2.Metrics) == 0 {
		return telemetry1, nil
	} else if len(telemetry1.Metrics) == 0 && len(telemetry2.Metrics) > 0 {
		return telemetry2, nil
	}
	// if both are empty return either one
	if len(telemetry1.Metrics) == 0 && len(telemetry2.Metrics) == 0 {
		return telemetry1, nil
	}
	return Telemetry{}, fmt.Errorf("service telemetry config is allowed to be set only once")
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
