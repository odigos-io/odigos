package services

import (
	"strings"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

type ServiceGraphEdges = map[string]map[string]collectormetrics.ServiceGraphEdge

// BaseServiceName extracts the service name from a composite node ID.
// Composite IDs follow the format "serviceName|dim1|dim2|..." where the
// service name is everything before the first "|" separator.
// If the ID contains no "|", it is returned as-is.
func BaseServiceName(compositeID string) string {
	if i := strings.Index(compositeID, "|"); i >= 0 {
		return compositeID[:i]
	}
	return compositeID
}

func edgeToModel(compositeKey string, edge collectormetrics.ServiceGraphEdge) *model.ServiceMapToSource {
	return &model.ServiceMapToSource{
		NodeID:      compositeKey,
		ServiceName: BaseServiceName(compositeKey),
		IsVirtual:   edge.ToNodeIsVirtual,
		Requests:    int(edge.RequestCount),
		DateTime:    edge.LastUpdated.Format(time.RFC3339),
	}
}

// mergeEdge aggregates an edge into m keyed by base service name,
// summing request counts and keeping the most recent timestamp.
func mergeEdge(m map[string]*model.ServiceMapToSource, compositeKey string, edge collectormetrics.ServiceGraphEdge) {
	base := BaseServiceName(compositeKey)
	if existing, ok := m[base]; ok {
		existing.Requests += int(edge.RequestCount)
		ts := edge.LastUpdated.Format(time.RFC3339)
		if ts > existing.DateTime {
			existing.DateTime = ts
		}
		if edge.ToNodeIsVirtual {
			existing.IsVirtual = true
		}
	} else {
		m[base] = edgeToModel(compositeKey, edge)
	}
}

func mapValues(m map[string]*model.ServiceMapToSource) []*model.ServiceMapToSource {
	out := make([]*model.ServiceMapToSource, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// PeerSources finds all inbound and outbound peers for a given base service name.
// The allEdges map may contain composite keys (e.g. "redis|ns1") so we compare
// against the base name extracted from each key. Results are deduplicated by
// base service name, summing request counts and keeping the latest timestamp.
func PeerSources(allEdges ServiceGraphEdges, serviceName string) *model.PeerSources {
	inboundMap := make(map[string]*model.ServiceMapToSource)
	outboundMap := make(map[string]*model.ServiceMapToSource)

	for callerKey, targets := range allEdges {
		if BaseServiceName(callerKey) == serviceName {
			for targetKey, edge := range targets {
				mergeEdge(outboundMap, targetKey, edge)
			}
		}
		for targetKey, edge := range targets {
			if BaseServiceName(targetKey) == serviceName {
				mergeEdge(inboundMap, callerKey, edge)
			}
		}
	}

	return &model.PeerSources{
		Inbound:  mapValues(inboundMap),
		Outbound: mapValues(outboundMap),
	}
}
