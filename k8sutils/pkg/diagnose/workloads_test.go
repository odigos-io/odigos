package diagnose

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func TestCategorizeSourcesForDiagnose(t *testing.T) {
	sources := []odigosv1.Source{
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{
			Kind: k8sconsts.WorkloadKindNamespace, Name: "payments", Namespace: "payments",
		}}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{
			Kind: k8sconsts.WorkloadKindDeployment, Name: "checkout", Namespace: "shop",
		}}},
		{Spec: odigosv1.SourceSpec{
			DisableInstrumentation: true,
			Workload: k8sconsts.PodWorkload{
				Kind: k8sconsts.WorkloadKindDeployment, Name: "skip-me", Namespace: "payments",
			},
		}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{
			Kind: k8sconsts.WorkloadKindStaticPod, Name: "static", Namespace: "shop",
		}}},
	}

	plan := categorizeSourcesForDiagnose(sources, nil)

	require.True(t, plan.namespaceSources["payments"])
	require.Equal(t, []k8sconsts.PodWorkload{{
		Namespace: "shop", Name: "checkout", Kind: k8sconsts.WorkloadKindDeployment,
	}}, plan.explicitWorkloads)
	require.Equal(t, []disabledWorkloadExclusion{{
		Namespace: "payments", Kind: k8sconsts.WorkloadKindDeployment, Name: "skip-me",
	}}, plan.disabledExclusions)
}

func TestCategorizeSourcesForDiagnoseNamespaceFilter(t *testing.T) {
	sources := []odigosv1.Source{
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{
			Kind: k8sconsts.WorkloadKindDeployment, Name: "a", Namespace: "keep",
		}}},
		{Spec: odigosv1.SourceSpec{Workload: k8sconsts.PodWorkload{
			Kind: k8sconsts.WorkloadKindDeployment, Name: "b", Namespace: "drop",
		}}},
	}

	plan := categorizeSourcesForDiagnose(sources, []string{"keep"})
	require.Len(t, plan.explicitWorkloads, 1)
	require.Equal(t, "a", plan.explicitWorkloads[0].Name)
}

func TestIsWorkloadExcluded(t *testing.T) {
	exclusions := []disabledWorkloadExclusion{
		{Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment, Name: "exact"},
		{Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment, Name: "foo-.*", Regex: true},
	}

	require.True(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment, Name: "exact",
	}, exclusions))
	require.True(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment, Name: "foo-bar",
	}, exclusions))
	require.False(t, isWorkloadExcluded(k8sconsts.PodWorkload{
		Namespace: "ns", Kind: k8sconsts.WorkloadKindDeployment, Name: "other",
	}, exclusions))
}
