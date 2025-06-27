package source

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

// IsObjectInstrumentedBySource returns true if the given object has an active, non-excluding Source.
// 1) Is the object actively included by a workload Source: true
// 2) Is the object instrumentation disabled on the workload Source (overrides namespace instrumentation): false
// 3) Is the object actively included by a namespace Source: true
// 4) False
func IsObjectInstrumentedBySource(ctx context.Context, sources *odigosv1.WorkloadSources, err error) (bool, metav1.Condition, error) {
	// Check if a Source object exists for this object
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

func CalculateDataStreamsLabels(workloadSources *odigosv1.WorkloadSources) map[string]string {
	workloadLabels := getSourceDataStreamsLabels(workloadSources.Workload)
	namespaceLabels := getSourceDataStreamsLabels(workloadSources.Namespace)

	// Start merging logic:
	mergedLabels := make(map[string]string)

	// Start from namespace labels (if any)
	for k, v := range namespaceLabels {
		mergedLabels[k] = v
	}

	// Add any workload, override namespace labels
	for k, v := range workloadLabels {
		mergedLabels[k] = v
	}

	return mergedLabels
}

// IsDataStreamLabel returns true if the label is a datastream label.
func IsDataStreamLabel(labelKey string) bool {
	return strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix)
}

// GetSourceDataStreamsLabels extracts only datastream labels from the Source object.
func getSourceDataStreamsLabels(source *odigosv1.Source) map[string]string {
	result := make(map[string]string)

	if source == nil {
		return result
	}

	for k, v := range source.Labels {
		if IsDataStreamLabel(k) {
			result[k] = v
		}
	}

	return result
}
