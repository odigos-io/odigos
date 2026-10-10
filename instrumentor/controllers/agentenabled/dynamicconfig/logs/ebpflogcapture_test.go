package logs

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func ebpfLogCaptureDistro() *distro.OtelDistro {
	return &distro.OtelDistro{
		Logs: &distro.Logs{
			EbpfLogCapture: &distro.EbpfLogCapture{Supported: true},
		},
	}
}

func ebpfRule(enabled *bool) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			EbpfLogCapture: &instrumentationrules.EbpfLogCapture{Enabled: enabled},
		},
	}
}

func TestCalculateEbpfLogCaptureConfig_enabledByAnyRule(t *testing.T) {
	off, on := false, true
	rules := []odigosv1.InstrumentationRule{ebpfRule(&off), ebpfRule(&on)}

	got := CalculateEbpfLogCaptureConfig(ebpfLogCaptureDistro(), &rules)

	require.NotNil(t, got)
	require.True(t, *got.Enabled)
}

func TestCalculateEbpfLogCaptureConfig_doesNotMutateInputRules(t *testing.T) {
	off, on := false, true
	disabledRule := ebpfRule(&off)
	enabledRule := ebpfRule(&on)

	rules := []odigosv1.InstrumentationRule{disabledRule, enabledRule}
	CalculateEbpfLogCaptureConfig(ebpfLogCaptureDistro(), &rules)

	require.False(t, *disabledRule.Spec.EbpfLogCapture.Enabled,
		"merging must not enable ebpf log capture on the rule that disables it")
}

func TestCalculateEbpfLogCaptureConfig_outOfScopeContainerStaysDisabled(t *testing.T) {
	off, on := false, true
	disabledRule := ebpfRule(&off)
	enabledRule := ebpfRule(&on)

	// container A is in scope for both rules
	containerA := []odigosv1.InstrumentationRule{disabledRule, enabledRule}
	gotA := CalculateEbpfLogCaptureConfig(ebpfLogCaptureDistro(), &containerA)
	require.True(t, *gotA.Enabled)

	// container B is only in scope for the rule that disables capture
	containerB := []odigosv1.InstrumentationRule{disabledRule}
	gotB := CalculateEbpfLogCaptureConfig(ebpfLogCaptureDistro(), &containerB)

	require.NotNil(t, gotB)
	require.False(t, *gotB.Enabled,
		"a container out of scope for the enabling rule must not capture logs over ebpf")
}

func TestCalculateEbpfLogCaptureConfig_unsupportedDistro(t *testing.T) {
	on := true
	rules := []odigosv1.InstrumentationRule{ebpfRule(&on)}

	require.Nil(t, CalculateEbpfLogCaptureConfig(&distro.OtelDistro{}, &rules))
}
