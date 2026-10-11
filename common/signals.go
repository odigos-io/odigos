package common

// +kubebuilder:validation:Enum=LOGS;TRACES;METRICS;PROFILES
type ObservabilitySignal string

const (
	LogsObservabilitySignal     ObservabilitySignal = "LOGS"
	TracesObservabilitySignal   ObservabilitySignal = "TRACES"
	MetricsObservabilitySignal  ObservabilitySignal = "METRICS"
	ProfilesObservabilitySignal ObservabilitySignal = "PROFILES"
)
