package source

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

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
	obj client.Object) (bool, metav1.Condition, error) {
	// Check if a Source object exists for this object
	sources, err := odigosv1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		condition := metav1.Condition{
			Type:    odigosv1.MarkedForInstrumentationStatusConditionType,
			Status:  metav1.ConditionUnknown,
			Reason:  string(odigosv1.MarkedForInstrumentationReasonError),
			Message: "cannot determine if workload is marked for instrumentation due to error: " + err.Error(),
		}
		return false, condition, err
	}

	if sources.Workload != nil {
		if !odigosv1.IsDisabledSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			message := fmt.Sprintf("workload marked for automatic instrumentation by workload source CR '%s' in namespace '%s'",
				sources.Workload.Name, sources.Workload.Namespace)
			condition := metav1.Condition{
				Type:    odigosv1.MarkedForInstrumentationStatusConditionType,
				Status:  metav1.ConditionTrue,
				Reason:  string(odigosv1.MarkedForInstrumentationReasonWorkloadSource),
				Message: message,
			}
			return true, condition, nil
		}
		if odigosv1.IsDisabledSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			message := fmt.Sprintf("workload marked to disable instrumentation by workload source CR '%s' in namespace '%s'",
				sources.Workload.Name, sources.Workload.Namespace)
			condition := metav1.Condition{
				Type:    odigosv1.MarkedForInstrumentationStatusConditionType,
				Status:  metav1.ConditionFalse,
				Reason:  string(odigosv1.MarkedForInstrumentationReasonWorkloadSourceDisabled),
				Message: message,
			}
			return false, condition, nil
		}
	}

	if sources.Namespace != nil && !odigosv1.IsDisabledSource(sources.Namespace) && !k8sutils.IsTerminating(sources.Namespace) {
		reason := odigosv1.MarkedForInstrumentationReasonNamespaceSource
		message := fmt.Sprintf("workload marked for automatic instrumentation by namespace source CR '%s' in namespace '%s'",
			sources.Namespace.Name, sources.Namespace.Namespace)
		condition := metav1.Condition{
			Type:    odigosv1.MarkedForInstrumentationStatusConditionType,
			Status:  metav1.ConditionTrue,
			Reason:  string(reason),
			Message: message,
		}
		return true, condition, nil
	}

	condition := metav1.Condition{
		Type:    odigosv1.MarkedForInstrumentationStatusConditionType,
		Status:  metav1.ConditionFalse,
		Reason:  string(odigosv1.MarkedForInstrumentationReasonNoSource),
		Message: "workload not marked for automatic instrumentation by any source CR",
	}
	return false, condition, nil
}

// SourceStatePermitsInstrumentation returns true if a Source:
// 1) Inclusive AND NOT terminating, or
// 2) Exclusive AND terminating
// This shows whether the Source and its state make it possible for the Source's workload to be instrumented.
// In general, this determines whether controllers should take an "instrumentation" or "uninstrumentation" path.
//
// However, this function alone does not guarantee that the Source's workload _will_ be instrumented, for example:
//   - A terminating Disabled Source does not mean there is another, non-terminating, inclusive Source for the workload.
//     Therefore, in that case you must check to see if there is another Source for the workload.
//   - A non-terminating Enabled Namespace Source could still have specific Workloads disabled by their own Workload Sources.
//
// This function is meant to be used as a basic filter for the top level instrumentor controllers
// (which are triggered by an event for only a single Source).
// Individual workloads should have their instrumentation state verified before acting on them.
func SourceStatePermitsInstrumentation(source *odigosv1.Source) bool {
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
