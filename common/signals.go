package common

import "strings"

//+kubebuilder:validation:Enum=LOGS;TRACES;METRICS
type ObservabilitySignal string

const (
	LogsObservabilitySignal    ObservabilitySignal = "LOGS"
	TracesObservabilitySignal  ObservabilitySignal = "TRACES"
	MetricsObservabilitySignal ObservabilitySignal = "METRICS"
)

func GetSignal(s string) (ObservabilitySignal, bool) {
	val := ObservabilitySignal(strings.ToUpper(s))
	switch val {
	case LogsObservabilitySignal, TracesObservabilitySignal, MetricsObservabilitySignal:
		return val, true
	default:
		return "", false
	}
}
