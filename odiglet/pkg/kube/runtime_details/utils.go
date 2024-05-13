package runtime_details

import (
	"context"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func isObjectLabeled(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationEnabled {
			return true
		}
	}

	return false
}

func isInstrumentationDisabledExplicitly(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels != nil {
		val, exists := labels[consts.OdigosInstrumentationLabel]
		if exists && val == consts.InstrumentationDisabled {
			return true
		}
	}

	return false
}

func isNamespaceLabeled(ctx context.Context, obj client.Object, c client.Client) bool {
	var ns corev1.Namespace
	err := c.Get(ctx, client.ObjectKey{Name: obj.GetNamespace()}, &ns)
	if err != nil {
		log.Logger.Error(err, "error fetching namespace object")
		return false
	}

	return isObjectLabeled(&ns)
}

func isWorkloadInstrumentationEffectiveEnabled(ctx context.Context, c client.Client, workload client.Object) bool {

	// ignore if instrumentation is disabled explicitly
	if isInstrumentationDisabledExplicitly(workload) {
		return false
	}

	// if the workload has instrumentation enabled explicitly
	if isObjectLabeled(workload) {
		return true
	}

	// workload is not labeled for instrumentation, check if the namespace is labeled
	if isNamespaceLabeled(ctx, workload, c) {
		return true
	}

	return false
}
