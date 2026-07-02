package pod

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestAddOdigletInstalledAffinityCreatesRequiredAffinity(t *testing.T) {
	t.Setenv("ODIGOS_TIER", "community")
	p := &corev1.Pod{}

	AddOdigletInstalledAffinity(p)

	terms := p.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 1)
	requireOdigletInstalledRequirement(t, terms[0])
}

func TestAddOdigletInstalledAffinityAndsWithExistingRequiredTerms(t *testing.T) {
	t.Setenv("ODIGOS_TIER", "community")
	p := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "topology.kubernetes.io/zone",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"us-east-1a"},
								}},
							},
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "node.kubernetes.io/instance-type",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"c6i.large"},
								}},
							},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(p)

	terms := p.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 2, "adding odiglet affinity must not create an extra OR term")
	for _, term := range terms {
		require.Len(t, term.MatchExpressions, 2)
		requireOdigletInstalledRequirement(t, term)
	}
}

func TestAddOdigletInstalledAffinityIsIdempotentPerRequiredTerm(t *testing.T) {
	t.Setenv("ODIGOS_TIER", "community")
	p := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									odigletInstalledRequirementForTest(),
									{
										Key:      "topology.kubernetes.io/zone",
										Operator: corev1.NodeSelectorOpIn,
										Values:   []string{"us-east-1a"},
									},
								},
							},
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{{
									Key:      "node.kubernetes.io/instance-type",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"c6i.large"},
								}},
							},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(p)
	AddOdigletInstalledAffinity(p)

	terms := p.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	require.Len(t, terms, 2)
	for _, term := range terms {
		require.Equal(t, 1, countOdigletInstalledRequirements(term))
	}
}

func requireOdigletInstalledRequirement(t *testing.T, term corev1.NodeSelectorTerm) {
	t.Helper()
	require.Equal(t, 1, countOdigletInstalledRequirements(term))
}

func countOdigletInstalledRequirements(term corev1.NodeSelectorTerm) int {
	count := 0
	for _, expr := range term.MatchExpressions {
		if expr.Key == k8sconsts.OdigletOSSInstalledLabel &&
			expr.Operator == corev1.NodeSelectorOpIn &&
			len(expr.Values) == 1 &&
			expr.Values[0] == k8sconsts.OdigletInstalledLabelValue {
			count++
		}
	}
	return count
}

func odigletInstalledRequirementForTest() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      k8sconsts.OdigletOSSInstalledLabel,
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{k8sconsts.OdigletInstalledLabelValue},
	}
}
