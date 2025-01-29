package pod

import (
	corev1 "k8s.io/api/core/v1"
)

func AddOdigletInstalledAffinity(pod *corev1.Pod, nodeLabelKey, nodeLabelValue string) {
	// Ensure Affinity exists
	if pod.Spec.Affinity == nil {
		pod.Spec.Affinity = &corev1.Affinity{}
	}

	// Ensure NodeAffinity exists
	if pod.Spec.Affinity.NodeAffinity == nil {
		pod.Spec.Affinity.NodeAffinity = &corev1.NodeAffinity{}
	}

	// Ensure RequiredDuringSchedulingIgnoredDuringExecution exists
	if pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{},
		}
	}

	// Check if the term already exists to avoid duplicates
	for _, term := range pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == nodeLabelKey && expr.Operator == corev1.NodeSelectorOpIn {
				for _, val := range expr.Values {
					if val == nodeLabelValue {
						// return without adding a duplicate
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
	pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms = append(
		pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms,
		newTerm,
	)
}
