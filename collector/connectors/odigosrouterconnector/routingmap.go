package odigosrouterconnector

import (
	"github.com/odigos-io/odigos/collector/extension/odigosk8sresourcesexention"
	"github.com/odigos-io/odigos/common"
)

func calculateDatastreamsForSignals(config *Config, signal common.ObservabilitySignal) map[odigosk8sresourcesexention.DatastreamName]struct{} {
	result := make(map[odigosk8sresourcesexention.DatastreamName]struct{})

	for _, ds := range config.DataStreams {
		dataStreamName := odigosk8sresourcesexention.DatastreamName(ds.Name)
		for _, destination := range ds.Destinations {
			for _, sig := range destination.ConfiguredSignals {
				if sig == signal {
					result[dataStreamName] = struct{}{}
				}
			}
		}
	}
	return result
}
