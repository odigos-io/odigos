package common

//+kubebuilder:validation:Enum=LOGS;TRACES;METRICS
type ObservabilitySignal string

const (
	LogsObservabilitySignal    ObservabilitySignal = "LOGS"
	TracesObservabilitySignal  ObservabilitySignal = "TRACES"
	MetricsObservabilitySignal ObservabilitySignal = "METRICS"
)
