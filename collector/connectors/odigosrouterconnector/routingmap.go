package odigosrouterconnector

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/pipelinegen"
)

// RoutingIndex maps signals (logs/metrics/traces) to group pipelines
type RoutingIndex map[string][]string // signal -> []pipeline names

// SignalRoutingMap indexes all sources by SourceKey and provides routing per signal
//
//	 SignalRoutingMap{
//	    {ns1/deployment/frontend}: {
//	        "logs":    {"groupA"},
//	        "traces":  {"groupA", "groupB"},
//	        "metrics": {"groupB"},
//	    },
//	    {ns2/statefulset/db}: {
//	        "traces": {"groupB"},
//	    },
//	}
type SignalRoutingMap map[string]RoutingIndex

// BuildSignalRoutingMap prepares a fast-access routing map based on structured group details.
// Future-proof: usable by both routing connector and custom connector logic.
func BuildSignalRoutingMap(groups []pipelinegen.GroupDetails) SignalRoutingMap {
	result := make(SignalRoutingMap)

	for _, group := range groups {

		signalsForGroup := GetSignalsForGroup(group)

		for _, source := range group.Sources {
			key := fmt.Sprintf("%s/%s/%s", source.Namespace, NormalizeKind(source.Kind), source.Name)

			if _, exists := result[key]; !exists {
				result[key] = make(RoutingIndex)
			}

			for _, signal := range signalsForGroup {
				signalStr := strings.ToLower(string(signal))
				pipeline := group.Name
				result[key][signalStr] = appendIfMissing(result[key][signalStr], pipeline)
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

// GetSignalsForGroup returns all observability signals for a given group.
// This is used to forward all signals for signal group pipelines e.g. logs/groupA, traces/groupC, metrics/groupB.
func GetSignalsForGroup(group pipelinegen.GroupDetails) []common.ObservabilitySignal {
	signals := []common.ObservabilitySignal{}
	seen := make(map[common.ObservabilitySignal]struct{})

	for _, destination := range group.Destinations {
		for _, sig := range destination.ConfiguredSignals {
			if _, exists := seen[sig]; !exists {
				seen[sig] = struct{}{}
				signals = append(signals, sig)
			}
		}
	}
	return signals
}
