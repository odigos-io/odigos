package pipelinegen

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

// BuildDataStreamPipelines constructs data stream pipelines for logs, metrics, traces, profiles.
// Each pipeline receives from its routing connector and exports to all destinations relevant to the data stream.
func buildDataStreamPipelines(
	dataStreams []DataStreams,
	forwardConnectorByDest map[string][]string,
) map[string]config.Pipeline {
	pipelines := make(map[string]config.Pipeline)

	for _, dataStream := range dataStreams {
		for _, signal := range []string{"logs", "metrics", "traces", "profiles"} {
			pipelineName := fmt.Sprintf("%s/%s", signal, dataStream.Name)

			pipeline := config.Pipeline{
				Receivers: []string{fmt.Sprintf("odigosrouterconnector/%s", signal)},
				Exporters: []string{},
			}
			// every group pipeline should have a generic batch processor, except profiles:
			// the batch processor does not support the profiles signal in the pinned collector build.
			if signal != "profiles" {
				pipeline.Processors = []string{consts.GenericBatchProcessorConfigKey}
			}

			// Add forward connectors for each destination in the group to route telemetry data
			// Forward connectors follow the naming pattern: forward/<signal>/<destination-id>
			for _, dest := range dataStream.Destinations {
				connectors, exists := forwardConnectorByDest[dest.DestinationName]
				if !exists {
					continue
				}

				for _, connectorName := range connectors {
					if strings.HasPrefix(connectorName, fmt.Sprintf("forward/%s/", signal)) {
						pipeline.Exporters = append(pipeline.Exporters, connectorName)
					}
				}
			}

			if len(pipeline.Exporters) > 0 {
				pipelines[pipelineName] = pipeline
			}
		}
	}

	return pipelines
}
