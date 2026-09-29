package logs

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
)

// eBPF log capture hooks write syscalls, so it applies to every container regardless of distro.
// Trace correlation is best-effort and depends on the agent, not on this config.
func CalculateEbpfLogCaptureConfig(irls *[]odigosv1.InstrumentationRule) *instrumentationrules.EbpfLogCapture {
	var result *instrumentationrules.EbpfLogCapture
	for _, irl := range *irls {
		result = mergeEbpfLogCapture(result, irl.Spec.EbpfLogCapture)
	}
	return result
}

func mergeEbpfLogCapture(existing *instrumentationrules.EbpfLogCapture, incoming *instrumentationrules.EbpfLogCapture) *instrumentationrules.EbpfLogCapture {
	if incoming == nil {
		return existing
	}
	if existing == nil {
		return incoming
	}
	// OR logic: if any rule enables it, it's enabled
	if incoming.Enabled != nil && *incoming.Enabled {
		enabled := true
		existing.Enabled = &enabled
	}
	return existing
}
