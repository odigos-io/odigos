package pod

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	k8snode "github.com/odigos-io/odigos/k8sutils/pkg/node"
)

func odigletRequirement() corev1.NodeSelectorRequirement {
	return corev1.NodeSelectorRequirement{
		Key:      k8snode.DetermineNodeOdigletInstalledLabelByTier(),
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{k8sconsts.OdigletInstalledLabelValue},
	}
}

func requiredTerms(t *testing.T, pod *corev1.Pod) []corev1.NodeSelectorTerm {
	t.Helper()
	require.NotNil(t, pod.Spec.Affinity)
	require.NotNil(t, pod.Spec.Affinity.NodeAffinity)
	require.NotNil(t, pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
	return pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
}

func TestAddOdigletInstalledAffinityNoExistingAffinity(t *testing.T) {
	pod := &corev1.Pod{}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 1)
	assert.Equal(t, []corev1.NodeSelectorRequirement{odigletRequirement()}, terms[0].MatchExpressions)
}

// node selector terms are ORed, so the odiglet requirement must be ANDed into the existing term
// rather than appended as a new one. appending would let the pod schedule on a node without
// odiglet, and would also nullify the workload's own node affinity.
func TestAddOdigletInstalledAffinityExistingTermIsPreservedAndAnded(t *testing.T) {
	userRequirement := corev1.NodeSelectorRequirement{
		Key:      "node.kubernetes.io/instance-type",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"g4dn.xlarge"},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{userRequirement}},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 1, "the odiglet requirement must not be added as a separate ORed term")
	assert.Equal(t, []corev1.NodeSelectorRequirement{userRequirement, odigletRequirement()}, terms[0].MatchExpressions)
}

func TestAddOdigletInstalledAffinityMultipleExistingTerms(t *testing.T) {
	zoneA := corev1.NodeSelectorRequirement{
		Key:      "topology.kubernetes.io/zone",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"us-east-1a"},
	}
	zoneB := corev1.NodeSelectorRequirement{
		Key:      "topology.kubernetes.io/zone",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"us-east-1b"},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{zoneA}},
							{MatchExpressions: []corev1.NodeSelectorRequirement{zoneB}},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 2)
	assert.Equal(t, []corev1.NodeSelectorRequirement{zoneA, odigletRequirement()}, terms[0].MatchExpressions)
	assert.Equal(t, []corev1.NodeSelectorRequirement{zoneB, odigletRequirement()}, terms[1].MatchExpressions)
}

// a term may constrain only match fields (e.g. metadata.name); the requirement is still ANDed into it.
func TestAddOdigletInstalledAffinityExistingMatchFieldsTerm(t *testing.T) {
	matchField := corev1.NodeSelectorRequirement{
		Key:      "metadata.name",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"node-1"},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchFields: []corev1.NodeSelectorRequirement{matchField}},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 1)
	assert.Equal(t, []corev1.NodeSelectorRequirement{matchField}, terms[0].MatchFields)
	assert.Equal(t, []corev1.NodeSelectorRequirement{odigletRequirement()}, terms[0].MatchExpressions)
}

func TestAddOdigletInstalledAffinityIsIdempotent(t *testing.T) {
	userRequirement := corev1.NodeSelectorRequirement{
		Key:      "disktype",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"ssd"},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{userRequirement}},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(pod)
	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 1)
	assert.Equal(t, []corev1.NodeSelectorRequirement{userRequirement, odigletRequirement()}, terms[0].MatchExpressions)
}

// only some terms already carry the requirement; the rest must still get it, and none may be duplicated.
func TestAddOdigletInstalledAffinityPartiallyPresent(t *testing.T) {
	userRequirement := corev1.NodeSelectorRequirement{
		Key:      "disktype",
		Operator: corev1.NodeSelectorOpIn,
		Values:   []string{"ssd"},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{MatchExpressions: []corev1.NodeSelectorRequirement{odigletRequirement()}},
							{MatchExpressions: []corev1.NodeSelectorRequirement{userRequirement}},
						},
					},
				},
			},
		},
	}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 2)
	assert.Equal(t, []corev1.NodeSelectorRequirement{odigletRequirement()}, terms[0].MatchExpressions)
	assert.Equal(t, []corev1.NodeSelectorRequirement{userRequirement, odigletRequirement()}, terms[1].MatchExpressions)
}

// preferred affinity and other affinity types must be left untouched.
func TestAddOdigletInstalledAffinityLeavesOtherAffinityUntouched(t *testing.T) {
	preferred := []corev1.PreferredSchedulingTerm{
		{
			Weight: 10,
			Preference: corev1.NodeSelectorTerm{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{Key: "disktype", Operator: corev1.NodeSelectorOpIn, Values: []string{"ssd"}},
				},
			},
		},
	}
	podAntiAffinity := &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{TopologyKey: "kubernetes.io/hostname"},
		},
	}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Affinity: &corev1.Affinity{
				NodeAffinity: &corev1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: preferred,
				},
				PodAntiAffinity: podAntiAffinity,
			},
		},
	}

	AddOdigletInstalledAffinity(pod)

	terms := requiredTerms(t, pod)
	require.Len(t, terms, 1)
	assert.Equal(t, []corev1.NodeSelectorRequirement{odigletRequirement()}, terms[0].MatchExpressions)
	assert.Equal(t, preferred, pod.Spec.Affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution)
	assert.Equal(t, podAntiAffinity, pod.Spec.Affinity.PodAntiAffinity)
}
