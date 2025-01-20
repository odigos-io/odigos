package source

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IsObjectInstrumentedBySource returns true if the given object has an active, non-excluding Source.
// 1) Is the object actively included by a workload Source: true
// 2) Is the object instrumentation disabled on the workload Source (overrides namespace instrumentation): false
// 3) Is the object actively included by a namespace Source: true
// 4) False
func IsObjectInstrumentedBySource(ctx context.Context, k8sClient client.Client, obj client.Object) (bool, error) {
	// Check if a Source object exists for this object
	sources, err := v1alpha1.GetSources(ctx, k8sClient, obj)
	if err != nil {
		return false, err
	}

	if sources.Workload != nil {
		if !v1alpha1.IsExcludedSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			return true, nil
		}
		if v1alpha1.IsExcludedSource(sources.Workload) && !k8sutils.IsTerminating(sources.Workload) {
			return false, nil
		}
	}

	if sources.Namespace != nil && !v1alpha1.IsExcludedSource(sources.Namespace) && !k8sutils.IsTerminating(sources.Namespace) {
		return true, nil
	}

	return false, nil
}

// IsSourceRelevant returns true if the Source may require action from a controller.
// Specifically, the Source must be either:
// 1) Inclusive AND NOT terminating, or
// 2) Exclusive AND terminating
// This function alone should not be used to determine any instrumentation changes.
func IsSourceRelevant(source *v1alpha1.Source) bool {
	return v1alpha1.IsExcludedSource(source) == k8sutils.IsTerminating(source)
}
