package dynamicconfig

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsAgentOwnLogs(d *distro.OtelDistro) bool {
	return d.OwnLogs != nil
}

// calculates the agent log level for the container, by merging the log levels from all instrumentation rules.
func CalculateAgentDiagnostics(irls *[]odigosv1.InstrumentationRule, d *distro.OtelDistro) *instrumentationrules.AgentDiagnostics {

	if !DistroSupportsAgentOwnLogs(d) {
		return nil
	}

	var odigosAgentDiagnostics *instrumentationrules.AgentDiagnostics
	for _, irl := range *irls {
		odigosAgentDiagnostics = mergeAgentDiagnostics(odigosAgentDiagnostics, irl.Spec.AgentDiagnostics, *d.OwnLogs)
	}
	return odigosAgentDiagnostics
}

// merges 2 configs for agent log level, returning the most verbose one for each field.
func mergeAgentDiagnostics(existing *instrumentationrules.AgentDiagnostics, incoming *instrumentationrules.AgentDiagnostics, distroSupport distro.OwnDiagnostics) *instrumentationrules.AgentDiagnostics {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}

	odigosAgentDiagnostics := &instrumentationrules.AgentDiagnostics{}
	// return the log level that will output the most verbose logs from both options
	if distroSupport.OdigosAgentOwnLogerSupported {
		odigosAgentDiagnostics.OdigosLogLevel = mergeLogLevel(existing.OdigosLogLevel, incoming.OdigosLogLevel)
	}
	if distroSupport.OpenTelemetryComponentsLoggerSupported {
		odigosAgentDiagnostics.OpenTelemetryComponentsLogLevel = mergeLogLevel(existing.OpenTelemetryComponentsLogLevel, incoming.OpenTelemetryComponentsLogLevel)
	}
	return odigosAgentDiagnostics
}

// merge 2 log levels, returning the most verbose one.
func mergeLogLevel(existing *common.OdigosLogLevel, incoming *common.OdigosLogLevel) *common.OdigosLogLevel {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}

	if incoming.Compare(*existing) > 0 {
		return incoming
	} else {
		return existing
	}
}
