package logs

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
)

func DistroSupportsEbpfLogCapture(d *distro.OtelDistro) bool {
	return d.Logs != nil && d.Logs.EbpfLogCapture != nil && d.Logs.EbpfLogCapture.Supported
}

func CalculateEbpfLogCaptureConfig(d *distro.OtelDistro, irls *[]odigosv1.InstrumentationRule) *instrumentationrules.EbpfLogCapture {

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
