package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/stretchr/testify/require"
)

func kernelRule(k *instrumentationrules.KernelTraces) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{Spec: odigosv1.InstrumentationRuleSpec{KernelTraces: k}}
}

func TestCalculateKernelTracesConfig(t *testing.T) {
	filesystem := &instrumentationrules.KernelTraces{
		Filesystem: &instrumentationrules.KernelFilesystemTraces{},
	}
	cpuDelay := &instrumentationrules.KernelTraces{
		CPUDelay: &instrumentationrules.KernelCPUDelayTraces{},
	}
	subprocesses := func(recordCommandArgs *bool) *instrumentationrules.KernelTraces {
		return &instrumentationrules.KernelTraces{
			Subprocesses: &instrumentationrules.KernelSubprocessTraces{RecordCommandArgs: recordCommandArgs},
		}
	}

	tests := []struct {
		name  string
		rules []odigosv1.InstrumentationRule
		want  *instrumentationrules.KernelTraces
	}{
		{
			name:  "no rules",
			rules: nil,
			want:  nil,
		},
		{
			name:  "rules without kernel traces",
			rules: []odigosv1.InstrumentationRule{kernelRule(nil)},
			want:  nil,
		},
		{
			name:  "areas are unioned",
			rules: []odigosv1.InstrumentationRule{kernelRule(filesystem), kernelRule(subprocesses(nil)), kernelRule(cpuDelay)},
			want: &instrumentationrules.KernelTraces{
				Filesystem:   &instrumentationrules.KernelFilesystemTraces{},
				Subprocesses: &instrumentationrules.KernelSubprocessTraces{},
				CPUDelay:     &instrumentationrules.KernelCPUDelayTraces{},
			},
		},
		{
			name:  "command args on if any rule turns them on",
			rules: []odigosv1.InstrumentationRule{kernelRule(subprocesses(boolPtr(false))), kernelRule(subprocesses(boolPtr(true))), kernelRule(subprocesses(nil))},
			want: &instrumentationrules.KernelTraces{
				Subprocesses: &instrumentationrules.KernelSubprocessTraces{RecordCommandArgs: boolPtr(true)},
			},
		},
		{
			name:  "command args off if set off and never on",
			rules: []odigosv1.InstrumentationRule{kernelRule(subprocesses(nil)), kernelRule(subprocesses(boolPtr(false)))},
			want: &instrumentationrules.KernelTraces{
				Subprocesses: &instrumentationrules.KernelSubprocessTraces{RecordCommandArgs: boolPtr(false)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, CalculateKernelTracesConfig(&tt.rules))
		})
	}
}
