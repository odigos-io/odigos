package runtimemigration

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	corev1 "k8s.io/api/core/v1"
)

func revertInstrumentationDevices(original *corev1.PodTemplateSpec) bool {
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
func removeInjectInstrumentationLabel(original *corev1.PodTemplateSpec) bool {

	// odigosInjectInstrumentationLabel is the label used to enable the mutating webhook.
	// it is removed in favor of running pods webhook for all pods, and never mutating deployment objects.
	odigosInjectInstrumentationLabel := "odigos.io/inject-instrumentation"

	if original.Labels != nil {
		if _, ok := original.Labels[odigosInjectInstrumentationLabel]; ok {
			delete(original.Labels, odigosInjectInstrumentationLabel)
			return true
		}
	}
	return false
}
