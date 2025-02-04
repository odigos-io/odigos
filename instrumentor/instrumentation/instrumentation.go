package instrumentation

import (
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"

	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
)

func RevertInstrumentationDevices(original *corev1.PodTemplateSpec) bool {
	changed := false
	for _, container := range original.Spec.Containers {
		for resourceName := range container.Resources.Limits {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Limits, resourceName)
				changed = true
			}
		}
		// Is it needed?
		for resourceName := range container.Resources.Requests {
			if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
				delete(container.Resources.Requests, resourceName)
				changed = true
			}
		}
	}
	return changed
}

// RemoveInjectInstrumentationLabel removes the "odigos.io/inject-instrumentation" label if it exists.
func RemoveInjectInstrumentationLabel(original *corev1.PodTemplateSpec) bool {
	if original.Labels != nil {
		if _, ok := original.Labels[k8sconsts.OdigosInjectInstrumentationLabel]; ok {
			delete(original.Labels, k8sconsts.OdigosInjectInstrumentationLabel)
			return true
		}
	}
	return false
}
