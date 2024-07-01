package common

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/config"
)

/* Convenience methods to convert between k8s types and config interfaces */
func ToProcessorConfigurerArray(items []*odigosv1.Processor) []config.ProcessorConfigurer {
	configurers := make([]config.ProcessorConfigurer, len(items))
	for i := range items {
		configurers[i] = items[i]
	}
	return configurers
}

func ToExporterConfigurerArray(dests *odigosv1.DestinationList) []config.ExporterConfigurer {
	configurers := make([]config.ExporterConfigurer, len(dests.Items))
	for i := range dests.Items {
		configurers[i] = dests.Items[i]
	}
	return configurers
}
