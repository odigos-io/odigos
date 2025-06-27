package odigosrouterconnector

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/pipelinegen"
)

// RoutingIndex maps signals (logs/metrics/traces) to dataStream pipelines
type RoutingIndex map[common.ObservabilitySignal][]string // signal -> []pipeline names

// SignalRoutingMap indexes all sources and namespaces by SourceKey and provides routing per signal
//
//	 SignalRoutingMap{
//	    {ns1/deployment/frontend}: {
//	        "logs":    ["dataStream-A"],
//	        "traces":  ["dataStream-A", "dataStream-B"],
//	        "metrics": ["dataStream-B"],
//	    },
//	    {ns2/statefulset/db}: {
//	        "traces": ["dataStream-B"],
//	    },
//	    {ns3/*/*}: {
//	        "traces": ["dataStream-Default"],
//	    },
//	}
type SignalRoutingMap map[string]RoutingIndex

// BuildSignalRoutingMap prepares a fast-access routing map based on structured group details.
// Future-proof: usable by both routing connector and custom connector logic.
func BuildSignalRoutingMap(dataStreams []pipelinegen.DataStreams) SignalRoutingMap {
	result := make(SignalRoutingMap)

	for _, dataStream := range dataStreams {

		signalsForDataStream := GetSignalsForDataStream(dataStream)

		// Build the keys for the sources
		for _, source := range dataStream.Sources {
			key := fmt.Sprintf("%s/%s/%s", source.Namespace, NormalizeKind(source.Kind), source.Name)

			if _, exists := result[key]; !exists {
				result[key] = make(RoutingIndex)
			}

			for _, signal := range signalsForDataStream {
				dataStreamName := dataStream.Name
				result[key][signal] = appendIfMissing(result[key][signal], dataStreamName)
			}
		}
	}

	return result
}

// normalizeKind ensures kind comparisons are case-insensitive and aligned with OTel semantic keys
// e.g: Deployment -> deployment, StatefulSet -> statefulset, DaemonSet -> daemonset
func NormalizeKind(kind string) string {
	normalized := strings.ToLower(kind)
	switch normalized {
	case "deployment", "statefulset", "daemonset", "cronjob":
		return normalized
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

// GetSignalsForDataStream returns all observability signals for a given data stream.
// This is used to forward all signals for signal data stream pipelines e.g. logs/groupA, traces/groupC, metrics/groupB.
func GetSignalsForDataStream(dataStream pipelinegen.DataStreams) []common.ObservabilitySignal {
	const maxSignals = 3 // OpenTelemetry supports logs, traces, metrics

	signals := []common.ObservabilitySignal{}
	seen := make(map[common.ObservabilitySignal]struct{})

	for _, destination := range dataStream.Destinations {
		for _, sig := range destination.ConfiguredSignals {
			if _, exists := seen[sig]; !exists {
				seen[sig] = struct{}{}
				signals = append(signals, sig)
			}
			if len(signals) == maxSignals {
				return signals
			}
		}
	}
	return signals
}
