package k8sconsts

const (
	// SourceInstrumentationFinalizer is deprecated and no longer added.
	// Existing finalizers are stripped by the Source webhook and reconciler.
	// DEPRECATED: Source deletion no longer uses finalizers.
	SourceInstrumentationFinalizer = "odigos.io/source-instrumentation-finalizer"

	WorkloadNameLabel      = "odigos.io/workload-name"
	WorkloadNamespaceLabel = "odigos.io/workload-namespace"
	WorkloadKindLabel      = "odigos.io/workload-kind"

	SourceDataStreamLabelPrefix = "odigos.io/data-stream-"
)
