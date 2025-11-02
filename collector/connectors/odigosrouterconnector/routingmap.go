package odigosrouterconnector

import (
	"github.com/odigos-io/odigos/collector/extension/odigosrek8ssourcesexention"
	"github.com/odigos-io/odigos/common"
)

func calculateDatastreamsForSignals(config *Config, signal common.ObservabilitySignal) map[odigosrek8ssourcesexention.DatastreamName]struct{} {
	result := make(map[odigosrek8ssourcesexention.DatastreamName]struct{})

	for _, ds := range config.DataStreams {
		dataStreamName := odigosrek8ssourcesexention.DatastreamName(ds.Name)
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
