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
func CalculateAgentOwnLogs(irls *[]odigosv1.InstrumentationRule, d *distro.OtelDistro) *instrumentationrules.AgentOwnLogs {

	if !DistroSupportsAgentOwnLogs(d) {
		return nil
	}

	var odigosAgentOwnLogs *instrumentationrules.AgentOwnLogs
	for _, irl := range *irls {
		odigosAgentOwnLogs = mergeAgentOwnLogs(odigosAgentOwnLogs, irl.Spec.AgentOwnLogs, *d.OwnLogs)
	}
	return odigosAgentOwnLogs
}

// merges 2 configs for agent log level, returning the most verbose one for each field.
func mergeAgentOwnLogs(existing *instrumentationrules.AgentOwnLogs, incoming *instrumentationrules.AgentOwnLogs, distroSupport distro.OwnLogs) *instrumentationrules.AgentOwnLogs {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}

	odigosAgentOwnLogs := &instrumentationrules.AgentOwnLogs{}
	// return the log level that will output the most verbose logs from both options
	if distroSupport.OdigosAgentOwnLogerSupported {
		odigosAgentOwnLogs.OdigosLogLevel = mergeLogLevel(existing.OdigosLogLevel, incoming.OdigosLogLevel)
	}
	if distroSupport.OpenTelemetryComponentsLoggerSupported {
		odigosAgentOwnLogs.OpenTelemetryComponentsLogLevel = mergeLogLevel(existing.OpenTelemetryComponentsLogLevel, incoming.OpenTelemetryComponentsLogLevel)
	}
	return odigosAgentOwnLogs
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
