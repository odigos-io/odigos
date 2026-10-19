package traces

import (
	"cmp"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
)

// CalculateKernelTracesConfig merges the kernel traces of the rules: an area is traced if any rule
// sets it, and an option is on if any rule turns it on. It applies to every distro, since the kernel
// spans don't depend on the language instrumentation.
func CalculateKernelTracesConfig(irls *[]odigosv1.InstrumentationRule) *instrumentationrules.KernelTraces {
	var result *instrumentationrules.KernelTraces
	for _, irl := range *irls {
		result = mergeKernelTraces(result, irl.Spec.KernelTraces)
	}
	return result
}

func mergeKernelTraces(t1, t2 *instrumentationrules.KernelTraces) *instrumentationrules.KernelTraces {
	if t1 == nil {
		return t2
	}
	if t2 == nil {
		return t1
	}
	return &instrumentationrules.KernelTraces{
		Filesystem:   cmp.Or(t1.Filesystem, t2.Filesystem),
		Subprocesses: mergeKernelSubprocessTraces(t1.Subprocesses, t2.Subprocesses),
		CPUDelay:     cmp.Or(t1.CPUDelay, t2.CPUDelay),
	}
}

func mergeKernelSubprocessTraces(s1, s2 *instrumentationrules.KernelSubprocessTraces) *instrumentationrules.KernelSubprocessTraces {
	if s1 == nil {
		return s2
	}
	if s2 == nil {
		return s1
	}
	return &instrumentationrules.KernelSubprocessTraces{
		RecordCommandArgs: merge2RuleBooleans(s1.RecordCommandArgs, s2.RecordCommandArgs),
	}
}
