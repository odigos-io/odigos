package instrumentationrules

// KernelTraces selects the areas to trace. An area is enabled by setting it, even as {}.
// Areas added in later versions stay off until set.
// +kubebuilder:object:generate=true
// +kubebuilder:validation:MinProperties=1
type KernelTraces struct {
	// Adds a span for each file the application opens or changes the permissions of.
	// By default, a span will be emitted even if it is not a child of a span in progress, starting a new trace.
	// This behavior may become configurable in the future.
	Filesystem *KernelFilesystemTraces `json:"filesystem,omitempty" yaml:"filesystem,omitempty"`

	// Adds a span for each process the application starts, like curl or a script.
	// Sub processes and chain of processes are supported, and the spans are children of the span in progress when the process was started.
	// By default, a span will be emitted even if it is not a child of a span in progress, starting a new trace.
	// This behavior may become configurable in the future.
	Subprocesses *KernelSubprocessTraces `json:"subprocesses,omitempty" yaml:"subprocesses,omitempty"`

	// Adds a span when a thread of the application was ready to run but waited for a CPU while handling
	// a traced request. The span is named "cpu throttling" when the CPU limit of its container or pod
	// caused the wait, or "cpu contention" when other work on the node was using the CPUs.
	// Only waits of 1 ms or more are reported. This minimum may become configurable in the future.
	// By default, a span is only added as a child of the span the thread was handling, never as the
	// start of a new trace.
	CPUDelay *KernelCPUDelayTraces `json:"cpuDelay,omitempty" yaml:"cpuDelay,omitempty"`
}

type KernelFilesystemTraces struct {
}

type KernelCPUDelayTraces struct {
}

// +kubebuilder:object:generate=true
type KernelSubprocessTraces struct {
	// Records the command line arguments of each process.
	// Off by default, since arguments can hold secrets like passwords or tokens.
	RecordCommandArgs *bool `json:"recordCommandArgs,omitempty" yaml:"recordCommandArgs,omitempty"`
}
