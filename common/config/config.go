package config

import (
	"fmt"
	"sort"

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
	// GetSendingQueueConfig returns destination-level sending_queue settings.
	// Nil skips applying sending_queue. Zero-value config enables queue with OTel defaults
	// (queue_size 1000 requests, batch min_size 8192 items).
	GetSendingQueueConfig() *SendingQueueConfig
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

type LogsConfig struct {
	Level string `json:"level,omitempty"`
}

type Telemetry struct {
	Metrics  MetricsConfig      `json:"metrics,omitempty"`
	Logs     LogsConfig         `json:"logs,omitempty"`
	Resource *TelemetryResource `json:"resource,omitempty" yaml:"resource,omitempty"`
}

// TelemetryResource holds user-defined resource attributes attached to the collector's
// own telemetry. It uses the otelconf declarative schema (service.telemetry.resource.attributes),
// which replaced the deprecated inline map format.
type TelemetryResource struct {
	Attributes []ResourceAttribute `json:"attributes,omitempty"`
}

type ResourceAttribute struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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
	// Sort domain names so merge order is stable. Go map iteration is nondeterministic; without a fixed
	// order, merged YAML and tests can differ between runs (e.g. telemetry reader ordering).
	domainNames := make([]string, 0, len(configDomains))
	for name := range configDomains {
		domainNames = append(domainNames, name)
	}
	sort.Strings(domainNames)
	for _, name := range domainNames {
		cfg := configDomains[name]
		mergedConfig.Receivers, err = mergeGenericMaps(mergedConfig.Receivers, cfg.Receivers)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Exporters, err = mergeGenericMaps(mergedConfig.Exporters, cfg.Exporters)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Processors, err = mergeGenericMaps(mergedConfig.Processors, cfg.Processors)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Extensions, err = mergeGenericMaps(mergedConfig.Extensions, cfg.Extensions)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Connectors, err = mergeGenericMaps(mergedConfig.Connectors, cfg.Connectors)
		if err != nil {
			return Config{}, err
		}

		mergedConfig.Service.Extensions = mergeExtensions(mergedConfig.Service.Extensions, cfg.Service.Extensions)
		mergedConfig.Service.Pipelines, err = mergePipelines(mergedConfig.Service.Pipelines, cfg.Service.Pipelines)
		if err != nil {
			return Config{}, err
		}
		mergedConfig.Service.Telemetry, err = mergeTelemetry(mergedConfig.Service.Telemetry, cfg.Service.Telemetry)
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
	mergedPipelines := make(map[string]Pipeline, len(pipelines1)+len(pipelines2))
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

func mergeTelemetryResource(resource1 *TelemetryResource, resource2 *TelemetryResource) *TelemetryResource {
	if resource1 == nil || len(resource1.Attributes) == 0 { // shortcut for common cases
		return resource2
	} else if resource2 == nil || len(resource2.Attributes) == 0 {
		return resource1
	}

	merged := &TelemetryResource{
		Attributes: make([]ResourceAttribute, 0, len(resource1.Attributes)+len(resource2.Attributes)),
	}
	merged.Attributes = append(merged.Attributes, resource1.Attributes...)
	merged.Attributes = append(merged.Attributes, resource2.Attributes...)
	return merged
}

func mergeTelemetryReaders(readers1 []GenericMap, readers2 []GenericMap) []GenericMap {
	if len(readers1) == 0 && len(readers2) == 0 {
		return nil
	}
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
	logsLevel := telemetry2.Logs.Level
	if logsLevel == "" {
		logsLevel = telemetry1.Logs.Level
	}
	mergedTelemetry := Telemetry{
		Metrics: MetricsConfig{
			Level:   level,
			Readers: mergeTelemetryReaders(telemetry1.Metrics.Readers, telemetry2.Metrics.Readers),
		},
		Logs:     LogsConfig{Level: logsLevel},
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
