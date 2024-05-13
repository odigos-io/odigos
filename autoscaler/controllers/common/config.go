package common

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
)

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

/* Convenience methods to convert between k8s types and config interfaces */
func ToProcessorConfigurerArray(items []*odigosv1.Processor) []common.ProcessorConfigurer {
	configurers := make([]common.ProcessorConfigurer, len(items))
	for i := range items {
	    configurers[i] = items[i]
	}
	return configurers
}
