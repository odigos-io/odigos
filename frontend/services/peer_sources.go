package services

import (
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

type ServiceGraphEdges = map[string]map[string]collectormetrics.ServiceGraphEdge

func edgeToModel(name string, edge collectormetrics.ServiceGraphEdge) *model.ServiceMapToSource {
	return &model.ServiceMapToSource{
		ServiceName: name,
		Requests:    int(edge.RequestCount),
		DateTime:    edge.LastUpdated.Format(time.RFC3339),
	}
}

func PeerSources(allEdges ServiceGraphEdges, serviceName string) *model.PeerSources {
	var inbound, outbound []*model.ServiceMapToSource

	if targets, ok := allEdges[serviceName]; ok {
		for targetName, edge := range targets {
			outbound = append(outbound, edgeToModel(targetName, edge))
		}
	}

	for callerName, targets := range allEdges {
		if edge, ok := targets[serviceName]; ok {
			inbound = append(inbound, edgeToModel(callerName, edge))
		}
	}

	return &model.PeerSources{
		Inbound:  inbound,
		Outbound: outbound,
	}
}
