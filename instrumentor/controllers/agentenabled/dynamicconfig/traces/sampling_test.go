package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	distrotypes "github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseHTTPGetPath(t *testing.T) {
	t.Parallel()

	readiness := "readiness"
	one := "1"
	two := "2"

	tests := []struct {
		name        string
		rawPath     string
		wantRoute   string
		wantQueries []commonapisampling.QueryParamMatcher
	}{
		{
			name:      "path only",
			rawPath:   "/healthz",
			wantRoute: "/healthz",
		},
		{
			name:      "path with single query param",
			rawPath:   "/health?type=readiness",
			wantRoute: "/health",
			wantQueries: []commonapisampling.QueryParamMatcher{
				{Name: "type", ValueExact: &readiness},
			},
		},
		{
			name:      "path with multiple query params",
			rawPath:   "/health?b=2&a=1",
			wantRoute: "/health",
			wantQueries: []commonapisampling.QueryParamMatcher{
				{Name: "a", ValueExact: &one},
				{Name: "b", ValueExact: &two},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotRoute, gotQueries := parseHTTPGetPath(tt.rawPath)
			require.Equal(t, tt.wantRoute, gotRoute)
			require.Equal(t, tt.wantQueries, gotQueries)
		})
	}
}

func TestCalculateKubeletHttpGetProbePaths_splitsQueryParams(t *testing.T) {
	liveness := "liveness"
	readiness := "readiness"

	enabled := true
	keepPercentage := 0.0
	effectiveConfig := &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        &enabled,
				KeepPercentage: &keepPercentage,
			},
		},
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name: "app",
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{Path: "/health?type=liveness"},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{Path: "/health?type=readiness"},
								},
							},
						},
					},
				},
			},
		},
	}

	rules := calculateKubeletHealthProbesSamplingRules(
		effectiveConfig,
		&workload.DeploymentWorkload{Deployment: deployment},
		"app",
	)

	require.Len(t, rules, 2)
	require.Equal(t, "/health", rules[0].Operation.HttpServer.Route)
	require.Equal(t, []commonapisampling.QueryParamMatcher{
		{Name: "type", ValueExact: &liveness},
	}, rules[0].Operation.HttpServer.QueryParams)
	require.Equal(t, []commonapisampling.QueryParamMatcher{
		{Name: "type", ValueExact: &readiness},
	}, rules[1].Operation.HttpServer.QueryParams)
}

func TestCalculateKubeletHttpGetProbePaths_mergesSamePathAndQueryParams(t *testing.T) {
	pathsAndNames := addProbePathAndName(nil, "/healthz", nil, "LivenessProbe")
	pathsAndNames = addProbePathAndName(pathsAndNames, "/healthz", nil, "ReadinessProbe")

	require.Len(t, pathsAndNames, 1)
	require.Equal(t, "LivenessProbe,ReadinessProbe", pathsAndNames[0].RuleName)
}

func samplingWithAllCategories(name string, disabled bool) odigosv1.Sampling {
	percentageAtMost := 5.0
	percentageAtLeast := 100.0

	return odigosv1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "odigos-system"},
		Spec: odigosv1.SamplingSpec{
			Name:     name,
			Disabled: disabled,
			NoisyOperations: []odigosv1.NoisyOperation{
				{
					Name:             name + "-noisy",
					Operation:        &commonapisampling.HeadSamplingOperationMatcher{HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/" + name + "-healthz", Method: "GET"}},
					PercentageAtMost: &percentageAtMost,
				},
			},
			HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{
				{
					Name:              name + "-relevant",
					Error:             true,
					PercentageAtLeast: &percentageAtLeast,
				},
			},
			CostReductionRules: []odigosv1.CostReductionRule{
				{
					Name:             name + "-cost",
					PercentageAtMost: percentageAtMost,
				},
			},
		},
	}
}

func calculateSamplingCategoryRules(t *testing.T, samplings []odigosv1.Sampling) ([]commonapisampling.NoisyOperation, []commonapisampling.HighlyRelevantOperation, []commonapisampling.CostReductionRule) {
	t.Helper()

	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	return CalculateSamplingCategoryRulesForContainer(
		&samplings,
		common.JavaProgrammingLanguage,
		pw,
		"app",
		&distrotypes.OtelDistro{},
		nil,
		&common.OdigosConfiguration{},
	)
}

// a Sampling object with spec.disabled must not contribute any rule to the workload config,
// the same way the scheduler ignores it when deciding whether tail sampling is needed.
func TestCalculateSamplingCategoryRulesForContainer_disabledSamplingContributesNoRules(t *testing.T) {
	noisyOps, relevantOps, costRules := calculateSamplingCategoryRules(t, []odigosv1.Sampling{
		samplingWithAllCategories("legacy", true),
	})

	require.Empty(t, noisyOps)
	require.Empty(t, relevantOps)
	require.Empty(t, costRules)
}

func TestCalculateSamplingCategoryRulesForContainer_disabledSamplingDoesNotAffectEnabledOne(t *testing.T) {
	noisyOps, relevantOps, costRules := calculateSamplingCategoryRules(t, []odigosv1.Sampling{
		samplingWithAllCategories("legacy", true),
		samplingWithAllCategories("active", false),
	})

	require.Len(t, noisyOps, 1)
	require.Equal(t, "active-noisy", noisyOps[0].Name)
	require.Len(t, relevantOps, 1)
	require.Equal(t, "active-relevant", relevantOps[0].Name)
	require.Len(t, costRules, 1)
	require.Equal(t, "active-cost", costRules[0].Name)
}

func TestCalculateSamplingCategoryRulesForContainer_enabledSamplingKeepsRules(t *testing.T) {
	noisyOps, relevantOps, costRules := calculateSamplingCategoryRules(t, []odigosv1.Sampling{
		samplingWithAllCategories("active", false),
	})

	require.Len(t, noisyOps, 1)
	require.Len(t, relevantOps, 1)
	require.Len(t, costRules, 1)
}

// per rule "disabled" keeps the rule in the config (it still participates in metrics),
// which is a different mechanism than disabling the whole Sampling object.
func TestCalculateSamplingCategoryRulesForContainer_perRuleDisabledIsPropagated(t *testing.T) {
	sampling := samplingWithAllCategories("active", false)
	sampling.Spec.NoisyOperations[0].Disabled = true
	sampling.Spec.HighlyRelevantOperations[0].Disabled = true
	sampling.Spec.CostReductionRules[0].Disabled = true

	noisyOps, relevantOps, costRules := calculateSamplingCategoryRules(t, []odigosv1.Sampling{sampling})

	require.Len(t, noisyOps, 1)
	require.True(t, noisyOps[0].Disabled)
	require.Len(t, relevantOps, 1)
	require.True(t, relevantOps[0].Disabled)
	require.Len(t, costRules, 1)
	require.True(t, costRules[0].Disabled)
}
