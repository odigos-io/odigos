package logs

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func ebpfLogCaptureDistro(supported bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		Logs: &distro.Logs{
			EbpfLogCapture: &distro.EbpfLogCapture{Supported: supported},
		},
	}
}

func TestDistroSupportsEbpfLogCapture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		distro   *distro.OtelDistro
		expected bool
	}{
		{
			name:     "distro without logs support",
			distro:   &distro.OtelDistro{},
			expected: false,
		},
		{
			name:     "logs without ebpf log capture",
			distro:   &distro.OtelDistro{Logs: &distro.Logs{}},
			expected: false,
		},
		{
			name:     "ebpf log capture not supported",
			distro:   ebpfLogCaptureDistro(false),
			expected: false,
		},
		{
			name:     "ebpf log capture supported",
			distro:   ebpfLogCaptureDistro(true),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, DistroSupportsEbpfLogCapture(tt.distro))
		})
	}
}

// eBPF log capture replaces the filelog receiver for the workload, so the OR semantics
// across the rules matching a container decide how logs are collected.
func TestCalculateEbpfLogCaptureConfig(t *testing.T) {
	t.Parallel()

	enabled := true
	disabled := false

	ruleWith := func(ebpfLogCapture *instrumentationrules.EbpfLogCapture) odigosv1.InstrumentationRule {
		return odigosv1.InstrumentationRule{
			Spec: odigosv1.InstrumentationRuleSpec{EbpfLogCapture: ebpfLogCapture},
		}
	}

	tests := []struct {
		name     string
		distro   *distro.OtelDistro
		irls     []odigosv1.InstrumentationRule
		expected *instrumentationrules.EbpfLogCapture
	}{
		{
			name:     "distro does not support ebpf log capture",
			distro:   ebpfLogCaptureDistro(false),
			irls:     []odigosv1.InstrumentationRule{ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled})},
			expected: nil,
		},
		{
			name:     "no rules",
			distro:   ebpfLogCaptureDistro(true),
			irls:     []odigosv1.InstrumentationRule{},
			expected: nil,
		},
		{
			name:     "rules without ebpf log capture",
			distro:   ebpfLogCaptureDistro(true),
			irls:     []odigosv1.InstrumentationRule{ruleWith(nil), ruleWith(nil)},
			expected: nil,
		},
		{
			name:     "single enabling rule",
			distro:   ebpfLogCaptureDistro(true),
			irls:     []odigosv1.InstrumentationRule{ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled})},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
		{
			name:     "single disabling rule",
			distro:   ebpfLogCaptureDistro(true),
			irls:     []odigosv1.InstrumentationRule{ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &disabled})},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &disabled},
		},
		{
			name:   "a later enabling rule wins over a disabling one",
			distro: ebpfLogCaptureDistro(true),
			irls: []odigosv1.InstrumentationRule{
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &disabled}),
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled}),
			},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
		{
			name:   "a later disabling rule does not turn off an enabled one",
			distro: ebpfLogCaptureDistro(true),
			irls: []odigosv1.InstrumentationRule{
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled}),
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &disabled}),
			},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
		{
			name:   "a rule that does not configure ebpf log capture does not clear an enabled one",
			distro: ebpfLogCaptureDistro(true),
			irls: []odigosv1.InstrumentationRule{
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled}),
				ruleWith(nil),
			},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
		{
			name:   "a rule with no explicit value does not clear an enabled one",
			distro: ebpfLogCaptureDistro(true),
			irls: []odigosv1.InstrumentationRule{
				ruleWith(&instrumentationrules.EbpfLogCapture{Enabled: &enabled}),
				ruleWith(&instrumentationrules.EbpfLogCapture{}),
			},
			expected: &instrumentationrules.EbpfLogCapture{Enabled: &enabled},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			irls := tt.irls
			require.Equal(t, tt.expected, CalculateEbpfLogCaptureConfig(tt.distro, &irls))
		})
	}
}
