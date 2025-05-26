package pipelinegen

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/common/consts"
)

type OtelPipelines map[string]config.Pipeline

// BuildGroupPipelines constructs group pipelines for logs, metrics, traces.
// Each pipeline receives from its routing connector and exports to all destinations relevant to the group.
func BuildGroupPipelines(
	groups []GroupDetails,
	forwardConnectorByDest map[string][]string,
) map[string]config.Pipeline {
	pipelines := make(map[string]config.Pipeline)

	for _, group := range groups {
		for _, signal := range []string{"logs", "metrics", "traces"} {
			pipelineName := fmt.Sprintf("%s/%s", signal, group.Name)

			pipeline := config.Pipeline{
				Receivers:  []string{fmt.Sprintf("odigosrouterconnector/%s", signal)},
				Processors: []string{consts.GenericBatchProcessor},
				Exporters:  []string{},
			}

			// Add forward connectors for each destination in the group to route telemetry data
			// Forward connectors follow the naming pattern: forward/<signal>/<destination-id>
			for _, dest := range group.Destinations {
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
