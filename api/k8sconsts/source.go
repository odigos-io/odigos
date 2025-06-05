package k8sconsts

const (
	// StartLangDetectionFinalizer is used for Workload exclusion Sources. When a Workload exclusion Source
	// is deleted, we want to go to the startlangdetection controller. There, we will check if the Workload should
	// start inheriting Namespace instrumentation.
	// DEPRECATED: Combined into a single SourceInstrumentationFinalizer
	StartLangDetectionFinalizer = "odigos.io/source-startlangdetection-finalizer"
	// DeleteInstrumentationConfigFinalizer is used for all non-exclusion (normal) Sources. When a normal Source
	// is deleted, we want to go to the deleteinstrumentationconfig controller to un-instrument the workload/namespace.
	// DEPRECATED: Combined into a single SourceInstrumentationFinalizer
	DeleteInstrumentationConfigFinalizer = "odigos.io/source-deleteinstrumentationconfig-finalizer"

	// SourceInstrumentationFinalizer is used by the Source controllers to prevent deletion of a Source CR until
	// it can be cleaned up.
	SourceInstrumentationFinalizer = "odigos.io/source-instrumentation-finalizer"

	WorkloadNameLabel      = "odigos.io/workload-name"
	WorkloadNamespaceLabel = "odigos.io/workload-namespace"
	WorkloadKindLabel      = "odigos.io/workload-kind"

	SourceGroupLabelPrefix = "odigos.io/group-"
)
