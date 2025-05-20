package odigosrouterconnector

import (
	"fmt"

	"github.com/odigos-io/odigos/common/pipelinegen"
)

type SourceKey struct {
	Namespace string
	Kind      string
	Name      string
}

// RoutingIndex maps signals (logs/metrics/traces) to group pipelines
type RoutingIndex map[string][]string // signal -> []pipeline names

// SignalRoutingMap indexes all sources by SourceKey and provides routing per signal
//
//	 SignalRoutingMap{
//	    {Namespace: "ns1", Kind: "deployment", Name: "frontend"}: {
//	        "logs":    {"logs/groupA"},
//	        "traces":  {"traces/groupA", "traces/groupB"},
//	        "metrics": {"metrics/groupB"},
//	    },
//	    {Namespace: "ns2", Kind: "statefulset", Name: "db"}: {
//	        "traces": {"traces/groupB"},
//	    },
//	}
type SignalRoutingMap map[string]RoutingIndex

// BuildSignalRoutingMap prepares a fast-access routing map based on structured group details.
// Future-proof: usable by both routing connector and custom connector logic.
func BuildSignalRoutingMap(groups []pipelinegen.GroupDetails) SignalRoutingMap {
	result := make(SignalRoutingMap)

	for _, group := range groups {
		for _, source := range group.Sources {
			key := fmt.Sprintf("%s/%s/%s", source.Namespace, NormalizeKind(source.Kind), source.Name)

			if _, exists := result[key]; !exists {
				result[key] = make(RoutingIndex)
			}

			for _, signal := range []string{"logs", "metrics", "traces"} {
				pipeline := signal + "/" + group.Name
				result[key][signal] = appendIfMissing(result[key][signal], pipeline)
			}
		}
	}

	return result
}

// normalizeKind ensures kind comparisons are case-insensitive and aligned with OTel semantic keys
// e.g: Deployment -> deployment, StatefulSet -> statefulset, DaemonSet -> daemonset
func NormalizeKind(kind string) string {
	switch kind {
	case "Deployment", "deployment":
		return "deployment"
	case "StatefulSet", "statefulset":
		return "statefulset"
	case "DaemonSet", "daemonset":
		return "daemonset"
	default:
		return kind
	}
}

func appendIfMissing(slice []string, item string) []string {
	for _, v := range slice {
		if v == item {
			return slice
		}
	}
	return append(slice, item)
}
