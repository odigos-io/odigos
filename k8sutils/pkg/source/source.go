package source

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsObjectInstrumentedBySource returns true if the given object has an active, non-excluding Source.
// 1) Is the object actively included by a workload Source: true
// 2) Is the object instrumentation disabled on the workload Source (overrides namespace instrumentation): false
// 3) Is the object actively included by a namespace Source: true
// 4) False
func IsObjectInstrumentedBySource(ctx context.Context,
	k8sClient client.Client,
	obj client.Object) (bool, odigosv1.MarkedForInstrumentationReason, string, error) {
	// Check if a Source object exists for this object
	sources, err := odigosv1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		reason := odigosv1.MarkedForInstrumentationReasonError
		return false, reason, "cannot determine if workload is marked for instrumentation due to error", err
	}

	if sources.Workload != nil {
		if !odigosv1.IsDisabledSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			reason := odigosv1.MarkedForInstrumentationReasonWorkloadSource
			message := fmt.Sprintf("workload marked for automatic instrumentation by workload source CR '%s' in namespace '%s'",
				sources.Workload.Name, sources.Workload.Namespace)
			return true, reason, message, nil
		}
		if odigosv1.IsDisabledSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			reason := odigosv1.MarkedForInstrumentationReasonWorkloadSourceDisabled
			message := fmt.Sprintf("workload marked to disable instrumentation by workload source CR '%s' in namespace '%s'",
				sources.Workload.Name, sources.Workload.Namespace)
			return false, reason, message, nil
		}
	}

	if sources.Namespace != nil && !odigosv1.IsDisabledSource(sources.Namespace) && !k8sutils.IsTerminating(sources.Namespace) {
		reason := odigosv1.MarkedForInstrumentationReasonNamespaceSource
		message := fmt.Sprintf("workload marked for automatic instrumentation by namespace source CR '%s' in namespace '%s'",
			sources.Namespace.Name, sources.Namespace.Namespace)
		return true, reason, message, nil
	}

	reason := odigosv1.MarkedForInstrumentationReasonNoSource
	message := "workload not marked for automatic instrumentation by any source CR"
	return false, reason, message, nil
}

// IsSourceRelevant returns true if a Source:
// 1) Inclusive AND NOT terminating, or
// 2) Exclusive AND terminating
// This function alone should not be used to determine any instrumentation changes, and is provided
// for the Instrumentor controllers to filter events.
func IsSourceRelevant(source *odigosv1.Source) bool {
	return odigosv1.IsDisabledSource(source) == k8sutils.IsTerminating(source)
}

// OtelServiceNameBySource returns the ReportedName for the given workload object.
// OTel service name is only valid for workload sources (not namespace sources).
// If none is configured, an empty string is returned.
func OtelServiceNameBySource(ctx context.Context, k8sClient client.Client, obj client.Object) (string, error) {
	sources, err := odigosv1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return "", err
	}

	if sources.Workload != nil {
		return sources.Workload.Spec.OtelServiceName, nil
	}

	return "", nil
}
