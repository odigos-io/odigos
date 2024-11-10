package pod

import (
	corev1 "k8s.io/api/core/v1"
)

func AddOdigletInstalledAffinity(original *corev1.PodTemplateSpec, nodeLabelKey, nodeLabelValue string) {
	// Ensure Affinity exists
	if original.Spec.Affinity == nil {
		original.Spec.Affinity = &corev1.Affinity{}
	}

	// Ensure NodeAffinity exists
	if original.Spec.Affinity.NodeAffinity == nil {
		original.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}

	// Ensure RequiredDuringSchedulingIgnoredDuringExecution exists
	if original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}
	}

	// Check if the term already exists
	for _, term := range original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == nodeLabelKey && expr.Operator == corev1.NodeSelectorOpIn {
				for _, val := range expr.Values {
					if val == nodeLabelValue {
						// The term already exists, so return without adding a duplicate
						return
					}
				}
			}
		}
	}

	// Append the new NodeSelectorTerm if it doesn't exist
	newTerm := corev1.NodeSelectorTerm{
		MatchExpressions: []corev1.NodeSelectorRequirement{
			{
				Key:      nodeLabelKey,
				Operator: corev1.NodeSelectorOpIn,
				Values:   []string{nodeLabelValue},
			},
		},
	}
	original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		newTerm,
	)
}

// RemoveNodeAffinityFromPodTemplate removes a specific NodeAffinity rule from a PodTemplateSpec if it exists.
func RemoveOdigletInstalledAffinity(original *corev1.PodTemplateSpec, nodeLabelKey, nodeLabelValue string) {
	if original.Spec.Affinity == nil || original.Spec.Affinity.NodeAffinity == nil {
		// No affinity or node affinity present, nothing to remove
		return
	}

	if original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		// No required node affinity present, nothing to remove
		return
	}

	// Iterate over NodeSelectorTerms and remove terms that match the key and value
	filteredTerms := []corev1.NodeSelectorTerm{}
	for _, term := range original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		filteredExpressions := []corev1.NodeSelectorRequirement{}
		for _, expr := range term.MatchExpressions {
			// Only keep expressions that don't match the given key and value
			if !(expr.Key == nodeLabelKey && expr.Operator == corev1.NodeSelectorOpIn && containsValue(expr.Values, nodeLabelValue)) {
				filteredExpressions = append(filteredExpressions, expr)
			}
		}

		// Only add the term if it still has expressions after filtering
		if len(filteredExpressions) > 0 {
			term.MatchExpressions = filteredExpressions
			filteredTerms = append(filteredTerms, term)
		}
	}

	// Update the NodeSelectorTerms with the filtered list or set to nil if empty
	if len(filteredTerms) > 0 {
		original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = filteredTerms
	} else {
		original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = nil
	}

	// Clean up empty NodeAffinity if needed
	if original.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil &&
		original.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution == nil {
		original.Spec.Affinity.NodeAffinity = nil
	}

	// Clean up empty Affinity if needed
	if original.Spec.Affinity.NodeAffinity == nil && original.Spec.Affinity.PodAffinity == nil && original.Spec.Affinity.PodAntiAffinity == nil {
		original.Spec.Affinity = nil
	}
}

// Helper function to check if a value is in a slice
func containsValue(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
