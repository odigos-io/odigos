package services

import (
	"strings"
	"time"

	"github.com/odigos-io/odigos/frontend/graph/model"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
)

type ServiceGraphEdges = map[string]map[string]collectormetrics.ServiceGraphEdge

// ServiceGraphNodeAttributesForServer prepares labels for the TO node row (metrics are edge-centric; API is node-centric).
func ServiceGraphNodeAttributesForServer(attrs map[string]string) map[string]string {
	return serviceGraphLabelsForPrefix(attrs, "server")
}

// ServiceGraphNodeAttributesForClient prepares labels for the FROM node row (inbound peers).
func ServiceGraphNodeAttributesForClient(attrs map[string]string) map[string]string {
	return serviceGraphLabelsForPrefix(attrs, "client")
}

// serviceGraphLabelsForPrefix filters to one edge side, strips the role prefix, converts underscores to dots for UI,
// and omits *service_name (already in serviceName).
func serviceGraphLabelsForPrefix(attrs map[string]string, prefix string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	prefixUnderscore := prefix + "_"
	// Prometheus normalizes service.name → service_name on the wire.
	skipServiceName := prefixUnderscore + "service_name"
	out := make(map[string]string)
	for k, v := range attrs {
		if k == prefix || !strings.HasPrefix(k, prefixUnderscore) {
			continue
		}
		if k == skipServiceName {
			continue
		}
		suffix := strings.TrimPrefix(k, prefixUnderscore)
		if suffix == "" {
			continue
		}
		// turns a metric label tail (underscores) into a UI-friendly dotted key.
		out[strings.ReplaceAll(suffix, "_", ".")] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

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

// Maps a collector service-graph edge metric to the GraphQL ServiceMapToSource type.
func EdgeToModel(compositeKey string, edge collectormetrics.ServiceGraphEdge, nodeAttrs map[string]string) *model.ServiceMapToSource {
	return &model.ServiceMapToSource{
		NodeID:         compositeKey,
		ServiceName:    BaseServiceName(compositeKey),
		IsVirtual:      edge.ToNodeIsVirtual,
		Requests:       int(edge.RequestCount),
		DateTime:       edge.LastUpdated.Format(time.RFC3339),
		NodeAttributes: model.NonIdentifyingAttribute{}.FromStringMap(nodeAttrs),
	}
}

// mergeEdge aggregates an edge into m keyed by base service name,
// summing request counts and keeping the most recent timestamp.
func mergeEdge(m map[string]*model.ServiceMapToSource, compositeKey string, edge collectormetrics.ServiceGraphEdge, nodeAttrs map[string]string) {
	base := BaseServiceName(compositeKey)
	if existing, ok := m[base]; ok {
		existing.Requests += int(edge.RequestCount)
		ts := edge.LastUpdated.Format(time.RFC3339)
		if ts > existing.DateTime {
			existing.DateTime = ts
			existing.NodeAttributes = model.NonIdentifyingAttribute{}.FromStringMap(nodeAttrs)
		}
		if edge.ToNodeIsVirtual {
			existing.IsVirtual = true
		}
	} else {
		m[base] = EdgeToModel(compositeKey, edge, nodeAttrs)
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
				mergeEdge(outboundMap, targetKey, edge, ServiceGraphNodeAttributesForServer(edge.Attributes))
			}
		}
		for targetKey, edge := range targets {
			if BaseServiceName(targetKey) == serviceName {
				mergeEdge(inboundMap, callerKey, edge, ServiceGraphNodeAttributesForClient(edge.Attributes))
			}
		}
	}

	return &model.PeerSources{
		Inbound:  mapValues(inboundMap),
		Outbound: mapValues(outboundMap),
	}
}
