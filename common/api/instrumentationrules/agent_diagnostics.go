package instrumentationrules

import (
	"github.com/odigos-io/odigos/common"
)

// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type AgentDiagnostics struct {

	// The log level of the odigos agent itself (startup, config, features, insttrumentation loading, etc.)
	OdigosLogLevel *common.OdigosLogLevel `json:"odigosLogLevel,omitempty"`

	// The log level of the OpenTelemetry components (Sdk, instrumentation libraries, detectors, etc.)
	// If unset, no OpenTelemetry components logs will be collected.
	OpenTelemetryComponentsLogLevel *common.OdigosLogLevel `json:"openTelemetryComponentsLogLevel,omitempty"`
}
