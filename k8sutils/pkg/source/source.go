package source

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

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

// OtelServiceNameBySource returns the ReportedName for the given workload object.
// OTel service name is only valid for workload sources (not namespace sources).
// If none is configured, it returns the default name which is the k8s workload resource name.
func OtelServiceNameBySource(ctx context.Context, k8sClient client.Client, obj client.Object) (string, error) {
	sources, err := odigosv1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return "", err
	}

	// use the otel service name attribute on the source if it exists
	if sources.Workload != nil {
		if sources.Workload.Spec.OtelServiceName != "" {
			return sources.Workload.Spec.OtelServiceName, nil
		}
	}

	// otherwise, fallback to the name of the workload (deployment/ds/sst name)
	return obj.GetName(), nil
}

// GetClientObjectFromSource returns the client.Object reference by the Source's spec.workload
// field, if the object exists.
// It is not valid to call this function with a namespace Source.
func GetClientObjectFromSource(ctx context.Context, kubeClient client.Client, source *odigosv1.Source) (client.Object, error) {
	obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
	err := kubeClient.Get(ctx, client.ObjectKey{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
	if err != nil {
		return nil, err
	}

	return obj, nil
}

func GetSourceCRD(ctx context.Context, kubeClient client.Client, nsName string, workloadName string, workloadKind k8sconsts.WorkloadKind) (*v1alpha1.Source, error) {
	sourceList := v1alpha1.SourceList{}
	err := kubeClient.List(ctx, &sourceList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			k8sconsts.WorkloadNamespaceLabel: nsName,
			k8sconsts.WorkloadNameLabel:      workloadName,
			k8sconsts.WorkloadKindLabel:      string(workloadKind),
		}),
	})

	if err != nil {
		return nil, err
	}
	if len(sourceList.Items) == 0 {
		return nil, apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: "source"}, workloadName)
	}
	if len(sourceList.Items) > 1 {
		return nil, fmt.Errorf(`expected to get 1 source "%s", got %d`, workloadName, len(sourceList.Items))
	}

	return &sourceList.Items[0], err
}

func GetSourceDataStreamsLabels(source *odigosv1.Source) map[string]string {
	sourceDataStreamsLabels := map[string]string{}
	// iterate over the source labels and add them to the sourceDataStreamsLabels
	for key, value := range source.Labels {
		if strings.HasPrefix(key, k8sconsts.SourceDataStreamLabelPrefix) {
			sourceDataStreamsLabels[key] = value
		}
	}
	return sourceDataStreamsLabels
}

// IsDataStreamLabel returns true if the label is a datastream label.
func IsDataStreamLabel(labelKey string) bool {
	return strings.HasPrefix(labelKey, k8sconsts.SourceDataStreamLabelPrefix)
}
