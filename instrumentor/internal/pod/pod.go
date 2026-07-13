package pod

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	k8snode "github.com/odigos-io/odigos/k8sutils/pkg/node"
	corev1 "k8s.io/api/core/v1"
)

func AddOdigletInstalledAffinity(pod *corev1.Pod) {
	odigletInstalledLabel := k8snode.DetermineNodeOdigletInstalledLabelByTier()
	odigletInstalledRequirement := corev1.NodeSelectorRequirement{
		Key:      odigletInstalledLabel,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{k8sconsts.OdigletInstalledLabelValue},
	}

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

	nodeSelectorTerms := &pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(*nodeSelectorTerms) == 0 {
		*nodeSelectorTerms = append(*nodeSelectorTerms, corev1.NodeSelectorTerm{
			MatchExpressions: []corev1.NodeSelectorRequirement{odigletInstalledRequirement},
		})
		return
	}

	for i := range *nodeSelectorTerms {
		if termHasOdigletInstalledRequirement((*nodeSelectorTerms)[i], odigletInstalledRequirement) {
			continue
		}
		(*nodeSelectorTerms)[i].MatchExpressions = append((*nodeSelectorTerms)[i].MatchExpressions, odigletInstalledRequirement)
	}
}

func termHasOdigletInstalledRequirement(term corev1.NodeSelectorTerm, requirement corev1.NodeSelectorRequirement) bool {
	for _, expr := range term.MatchExpressions {
		if expr.Key != requirement.Key || expr.Operator != requirement.Operator {
			continue
		}
		for _, val := range expr.Values {
			if val == k8sconsts.OdigletInstalledLabelValue {
				return true
			}
		}
	}

	return false
}
