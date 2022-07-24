package config

import (
	"fmt"
	"github.com/ghodss/yaml"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	"github.com/keyval-dev/odigos/common"
)

var availableConfigers = []Configer{&Honeycomb{}}

type genericMap map[string]interface{}

type Config struct {
	Receivers  genericMap `json:"receivers"`
	Exporters  genericMap `json:"exporters"`
	Processors genericMap `json:"processors"`
	Extensions genericMap `json:"extensions"`
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

type Configer interface {
	DestType() odigosv1.DestinationType
	ModifyConfig(dest *odigosv1.Destination, currentConfig *Config)
}

func Calculate(dests *odigosv1.DestinationList) (string, error) {
	currentConfig := getBasicConfig()
	configers, err := loadConfigers()
	if err != nil {
		return "", err
	}

	for _, dest := range dests.Items {
		configer, exists := configers[dest.Spec.Type]
		if !exists {
			return "", fmt.Errorf("no configer for %s", dest.Spec.Type)
		}

		configer.ModifyConfig(&dest, currentConfig)
	}

	data, err := yaml.Marshal(currentConfig)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func getBasicConfig() *Config {
	empty := struct{}{}
	return &Config{
		Receivers: genericMap{
			"otlp": genericMap{
				"protocols": genericMap{
					"grpc": empty,
				},
			},
		},
		Processors: genericMap{
			"batch": empty,
		},
		Extensions: genericMap{
			"health_check": empty,
			"zpages":       empty,
		},
		Exporters: map[string]interface{}{},
		Service: Service{
			Pipelines:  map[string]Pipeline{},
			Extensions: []string{"health_check", "zpages"},
		},
	}
}

func loadConfigers() (map[odigosv1.DestinationType]Configer, error) {
	configers := map[odigosv1.DestinationType]Configer{}
	for _, configer := range availableConfigers {
		if _, exists := configers[configer.DestType()]; exists {
			return nil, fmt.Errorf("duplicate configer for %s", configer.DestType())
		}

		configers[configer.DestType()] = configer
	}

	return configers, nil
}

func isSignalExists(dest *odigosv1.Destination, signal common.ObservabilitySignal) bool {
	for _, s := range dest.Spec.Signals {
		if s == signal {
			return true
		}
	}

	return false
}

func isTracingEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.TracesObservabilitySignal)
}

func isMetricsEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.MetricsObservabilitySignal)
}

func isLoggingEnabled(dest *odigosv1.Destination) bool {
	return isSignalExists(dest, common.LogsObservabilitySignal)
}
