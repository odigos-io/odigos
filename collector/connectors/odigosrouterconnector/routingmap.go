package odigosrouterconnector

import (
	"github.com/odigos-io/odigos/collector/connectors/odigosrouterconnector/internal/utils"
	"github.com/odigos-io/odigos/common"
)

func calculateDatastreamsForSignals(config *Config, signal common.ObservabilitySignal) map[utils.DatastreamName]struct{} {
	result := make(map[utils.DatastreamName]struct{})

	for _, ds := range config.DataStreams {
		dataStreamName := utils.DatastreamName(ds.Name)
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
