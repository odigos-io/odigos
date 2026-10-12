package traces

import (
	"slices"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
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
	empty := ""

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
			name:      "empty path",
			rawPath:   "",
			wantRoute: "",
		},
		{
			name:      "unparsable path is used as is",
			rawPath:   "/health\x7f",
			wantRoute: "/health\x7f",
		},
		{
			name:      "query only path keeps the raw path as the route",
			rawPath:   "?type=readiness",
			wantRoute: "?type=readiness",
			wantQueries: []commonapisampling.QueryParamMatcher{
				{Name: "type", ValueExact: &readiness},
			},
		},
		{
			name:      "query param without a value",
			rawPath:   "/health?debug",
			wantRoute: "/health",
			wantQueries: []commonapisampling.QueryParamMatcher{
				{Name: "debug", ValueExact: &empty},
			},
		},
		{
			name:      "repeated query param is sorted by value",
			rawPath:   "/health?a=2&a=1",
			wantRoute: "/health",
			wantQueries: []commonapisampling.QueryParamMatcher{
				{Name: "a", ValueExact: &one},
				{Name: "a", ValueExact: &two},
			},
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

func samplingTestDistro(headSampling bool, httpQueryParams bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		Name: "test-distro",
		Traces: &distro.Traces{
			HeadSampling: &distro.HeadSampling{
				Supported:                headSampling,
				HttpQueryParamsSupported: httpQueryParams,
			},
		},
	}
}

func samplingTestPodWorkload() k8sconsts.PodWorkload {
	return k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
}

func samplingHealthProbesConfig(enabled bool, keepPercentage *float64) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{
			K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{
				Enabled:        &enabled,
				KeepPercentage: keepPercentage,
			},
		},
	}
}

// deployment with a single container named "app" and the given http get probe paths.
// an empty path means the probe is not set at all.
func samplingTestDeployment(startup, liveness, readiness string) *workload.DeploymentWorkload {
	container := corev1.Container{Name: "app"}
	httpGetProbe := func(path string) *corev1.Probe {
		return &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: path}},
		}
	}
	if startup != "" {
		container.StartupProbe = httpGetProbe(startup)
	}
	if liveness != "" {
		container.LivenessProbe = httpGetProbe(liveness)
	}
	if readiness != "" {
		container.ReadinessProbe = httpGetProbe(readiness)
	}

	return &workload.DeploymentWorkload{Deployment: &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "app", Namespace: "default"},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{Containers: []corev1.Container{container}},
			},
		},
	}}
}

func TestDistroSupportsHeadSampling(t *testing.T) {
	require.False(t, DistroSupportsHeadSampling(&distro.OtelDistro{}))
	require.False(t, DistroSupportsHeadSampling(&distro.OtelDistro{Traces: &distro.Traces{}}))
	require.False(t, DistroSupportsHeadSampling(samplingTestDistro(false, false)))
	require.True(t, DistroSupportsHeadSampling(samplingTestDistro(true, false)))
}

func TestCalculateK8sHealthProbeSamplingPercentage(t *testing.T) {
	fivePercent := 5.0

	require.Equal(t, 0.0, calculateK8sHealthProbeSamplingPercentage(&common.OdigosConfiguration{}))
	require.Equal(t, 0.0, calculateK8sHealthProbeSamplingPercentage(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{},
	}))
	require.Equal(t, 0.0, calculateK8sHealthProbeSamplingPercentage(samplingHealthProbesConfig(true, nil)))
	require.Equal(t, 5.0, calculateK8sHealthProbeSamplingPercentage(samplingHealthProbesConfig(true, &fivePercent)))
}

func TestIsK8sHealthProbesSamplingEnabled(t *testing.T) {
	require.False(t, isK8sHealthProbesSamplingEnabled(&common.OdigosConfiguration{}))
	require.False(t, isK8sHealthProbesSamplingEnabled(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{},
	}))
	require.False(t, isK8sHealthProbesSamplingEnabled(&common.OdigosConfiguration{
		Sampling: &common.SamplingConfiguration{K8sHealthProbesSampling: &common.K8sHealthProbesSamplingConfiguration{}},
	}))
	require.False(t, isK8sHealthProbesSamplingEnabled(samplingHealthProbesConfig(false, nil)))
	require.True(t, isK8sHealthProbesSamplingEnabled(samplingHealthProbesConfig(true, nil)))
}

func TestQueryParamsMatch(t *testing.T) {
	a := "a"
	b := "b"

	require.True(t, queryParamsMatch(nil, nil))
	require.False(t, queryParamsMatch(nil, []commonapisampling.QueryParamMatcher{{Name: "type"}}))
	require.True(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type"}},
		[]commonapisampling.QueryParamMatcher{{Name: "type"}},
	))
	require.False(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type"}},
		[]commonapisampling.QueryParamMatcher{{Name: "kind"}},
	))
	require.True(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &a}},
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &a}},
	))
	require.False(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &a}},
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &b}},
	))
	require.False(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &a}},
		[]commonapisampling.QueryParamMatcher{{Name: "type"}},
	))
	require.False(t, queryParamsMatch(
		[]commonapisampling.QueryParamMatcher{{Name: "type"}},
		[]commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &a}},
	))
}

func TestCalculateKubeletHealthProbesSamplingRules_gating(t *testing.T) {
	workloadObj := samplingTestDeployment("", "/healthz", "")

	t.Run("no sampling config", func(t *testing.T) {
		require.Nil(t, calculateKubeletHealthProbesSamplingRules(&common.OdigosConfiguration{}, workloadObj, "app"))
	})
	t.Run("health probes sampling disabled", func(t *testing.T) {
		require.Nil(t, calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(false, nil), workloadObj, "app"))
	})
	t.Run("nil workload", func(t *testing.T) {
		require.Nil(t, calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, nil), nil, "app"))
	})
	t.Run("container not in the pod spec", func(t *testing.T) {
		require.Nil(t, calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, nil), workloadObj, "sidecar"))
	})
	t.Run("container has no http get probes", func(t *testing.T) {
		require.Nil(t, calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, nil), samplingTestDeployment("", "", ""), "app"))
	})
}

func TestCalculateKubeletHealthProbesSamplingRules_allProbeKinds(t *testing.T) {
	tenPercent := 10.0
	workloadObj := samplingTestDeployment("/startupz", "/healthz", "/healthz")

	rules := calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, &tenPercent), workloadObj, "app")

	// the startup probe comes first, and the liveness and readiness probes share one rule
	// because they use the same path and query params.
	require.Len(t, rules, 2)
	require.Equal(t, "kubelet health probe: StartupProbe", rules[0].Name)
	require.Equal(t, "/startupz", rules[0].Operation.HttpServer.Route)
	require.Equal(t, "kubelet health probe: LivenessProbe,ReadinessProbe", rules[1].Name)
	require.Equal(t, "/healthz", rules[1].Operation.HttpServer.Route)

	for _, rule := range rules {
		require.Equal(t, "GET", rule.Operation.HttpServer.Method)
		require.NotNil(t, rule.PercentageAtMost)
		require.Equal(t, 10.0, *rule.PercentageAtMost)
		require.NotEmpty(t, rule.Id)
	}
	require.NotEqual(t, rules[0].Id, rules[1].Id)
}

func TestCalculateKubeletHealthProbesSamplingRules_idIsIndependentOfWorkloadAndPercentage(t *testing.T) {
	twentyPercent := 20.0

	first := calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, nil), samplingTestDeployment("", "/healthz", ""), "app")
	second := calculateKubeletHealthProbesSamplingRules(samplingHealthProbesConfig(true, &twentyPercent), samplingTestDeployment("", "", "/healthz"), "app")

	require.Len(t, first, 1)
	require.Len(t, second, 1)
	// the rule id intentionally ignores the scope, the name and the percentage, so the same
	// probe path produces the same id in every container of every workload.
	require.Equal(t, first[0].Id, second[0].Id)
	require.NotEqual(t, first[0].Name, second[0].Name)
	require.Equal(t, 0.0, *first[0].PercentageAtMost)
}

func TestCalculateSamplingCategoryRulesForContainer_noRules(t *testing.T) {
	samplingRules := []odigosv1.Sampling{}

	noisyOps, relevantOps, costRules := CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.JavaProgrammingLanguage, samplingTestPodWorkload(), "app",
		samplingTestDistro(true, true), samplingTestDeployment("", "/healthz", ""), &common.OdigosConfiguration{})

	require.Nil(t, noisyOps)
	require.Nil(t, relevantOps)
	require.Nil(t, costRules)
}

func TestCalculateSamplingCategoryRulesForContainer_autoHealthProbeRules(t *testing.T) {
	samplingRules := []odigosv1.Sampling{}

	noisyOps, _, _ := CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.JavaProgrammingLanguage, samplingTestPodWorkload(), "app",
		samplingTestDistro(true, true), samplingTestDeployment("", "/healthz", ""),
		samplingHealthProbesConfig(true, nil))

	require.Len(t, noisyOps, 1)
	require.Equal(t, "kubelet health probe: LivenessProbe", noisyOps[0].Name)
}

func TestCalculateSamplingCategoryRulesForContainer_httpQueryParamsFilteredByDistro(t *testing.T) {
	readiness := "readiness"
	samplingRules := []odigosv1.Sampling{{
		Spec: odigosv1.SamplingSpec{
			NoisyOperations: []odigosv1.NoisyOperation{
				{
					Name: "user rule with query params",
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
							Route:       "/health",
							QueryParams: []commonapisampling.QueryParamMatcher{{Name: "type", ValueExact: &readiness}},
						},
					},
				},
				{
					Name: "user rule without query params",
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/health"},
					},
				},
			},
		},
	}}
	// the liveness probe path carries a query param, so its auto-rule is filtered by the same gate
	workloadObj := samplingTestDeployment("", "/health?type=liveness", "")

	t.Run("distro supports http query params", func(t *testing.T) {
		noisyOps, _, _ := CalculateSamplingCategoryRulesForContainer(
			&samplingRules, common.JavaProgrammingLanguage, samplingTestPodWorkload(), "app",
			samplingTestDistro(true, true), workloadObj, samplingHealthProbesConfig(true, nil))

		names := make([]string, 0, len(noisyOps))
		for _, op := range noisyOps {
			names = append(names, op.Name)
		}
		require.ElementsMatch(t, []string{
			"kubelet health probe: LivenessProbe",
			"user rule with query params",
			"user rule without query params",
		}, names)
	})

	t.Run("distro does not support http query params", func(t *testing.T) {
		noisyOps, _, _ := CalculateSamplingCategoryRulesForContainer(
			&samplingRules, common.JavaProgrammingLanguage, samplingTestPodWorkload(), "app",
			samplingTestDistro(true, false), workloadObj, samplingHealthProbesConfig(true, nil))

		require.Len(t, noisyOps, 1)
		require.Equal(t, "user rule without query params", noisyOps[0].Name)
	})

	t.Run("distro without a traces section", func(t *testing.T) {
		noisyOps, _, _ := CalculateSamplingCategoryRulesForContainer(
			&samplingRules, common.JavaProgrammingLanguage, samplingTestPodWorkload(), "app",
			&distro.OtelDistro{}, workloadObj, samplingHealthProbesConfig(true, nil))

		require.Len(t, noisyOps, 1)
		require.Equal(t, "user rule without query params", noisyOps[0].Name)
	})
}

func TestCalculateSamplingCategoryRulesForContainer_conversionKeepsRuleFields(t *testing.T) {
	fifty := 50.0
	ninety := 90.0
	durationMs := 250
	pw := samplingTestPodWorkload()

	noisyOperation := &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/noisy"},
	}
	relevantOperation := &commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/relevant"},
	}
	costOperation := &commonapisampling.TailSamplingOperationMatcher{
		HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{RoutePrefix: "/cost"},
	}

	samplingRules := []odigosv1.Sampling{{
		Spec: odigosv1.SamplingSpec{
			Name:  "rules set",
			Notes: "these notes must not reach the instrumentation config",
			NoisyOperations: []odigosv1.NoisyOperation{{
				Name:             "noisy",
				Disabled:         true,
				SourceScopes:     &odigosv1.SourcesScopes{Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage}},
				Operation:        noisyOperation,
				PercentageAtMost: &fifty,
			}},
			HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{{
				Name:              "relevant",
				Disabled:          true,
				Error:             true,
				DurationAtLeastMs: &durationMs,
				Operation:         relevantOperation,
				PercentageAtLeast: &ninety,
			}},
			CostReductionRules: []odigosv1.CostReductionRule{{
				Name:             "cost",
				Disabled:         true,
				Operation:        costOperation,
				PercentageAtMost: fifty,
			}},
		},
	}}

	noisyOps, relevantOps, costRules := CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Len(t, noisyOps, 1)
	require.Equal(t, "noisy", noisyOps[0].Name)
	require.True(t, noisyOps[0].Disabled)
	require.Same(t, noisyOperation, noisyOps[0].Operation)
	require.Equal(t, &fifty, noisyOps[0].PercentageAtMost)
	require.Equal(t, odigosv1.ComputeNoisyOperationHash(&samplingRules[0].Spec.NoisyOperations[0]), noisyOps[0].Id)

	require.Len(t, relevantOps, 1)
	require.Equal(t, "relevant", relevantOps[0].Name)
	require.True(t, relevantOps[0].Disabled)
	require.True(t, relevantOps[0].Error)
	require.Equal(t, &durationMs, relevantOps[0].DurationAtLeastMs)
	require.Same(t, relevantOperation, relevantOps[0].Operation)
	require.Equal(t, &ninety, relevantOps[0].PercentageAtLeast)
	require.Equal(t, odigosv1.ComputeHighlyRelevantOperationHash(&samplingRules[0].Spec.HighlyRelevantOperations[0]), relevantOps[0].Id)

	require.Len(t, costRules, 1)
	require.Equal(t, "cost", costRules[0].Name)
	require.True(t, costRules[0].Disabled)
	require.Same(t, costOperation, costRules[0].Operation)
	require.Equal(t, fifty, costRules[0].PercentageAtMost)
	require.Equal(t, odigosv1.ComputeCostReductionRuleHash(&samplingRules[0].Spec.CostReductionRules[0]), costRules[0].Id)
}

func TestCalculateSamplingCategoryRulesForContainer_sourceScopeFiltering(t *testing.T) {
	pw := samplingTestPodWorkload()
	javaScope := &odigosv1.SourcesScopes{Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage}}
	otherWorkloadScope := &odigosv1.SourcesScopes{Sources: []k8sconsts.PodWorkload{{
		Name: "other", Namespace: pw.Namespace, Kind: pw.Kind,
	}}}

	samplingRules := []odigosv1.Sampling{{
		Spec: odigosv1.SamplingSpec{
			NoisyOperations: []odigosv1.NoisyOperation{
				{Name: "in scope", SourceScopes: javaScope},
				{Name: "out of scope", SourceScopes: otherWorkloadScope},
			},
			HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{
				{Name: "in scope", SourceScopes: javaScope},
				{Name: "out of scope", SourceScopes: otherWorkloadScope},
			},
			CostReductionRules: []odigosv1.CostReductionRule{
				{Name: "in scope", SourceScopes: javaScope},
				{Name: "out of scope", SourceScopes: otherWorkloadScope},
			},
		},
	}}

	noisyOps, relevantOps, costRules := CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Len(t, noisyOps, 1)
	require.Equal(t, "in scope", noisyOps[0].Name)
	require.Len(t, relevantOps, 1)
	require.Equal(t, "in scope", relevantOps[0].Name)
	require.Len(t, costRules, 1)
	require.Equal(t, "in scope", costRules[0].Name)

	// the same rules for a container of a different language are all filtered out
	noisyOps, relevantOps, costRules = CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.GoProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Empty(t, noisyOps)
	require.Empty(t, relevantOps)
	require.Empty(t, costRules)
}

// the InstrumentationConfig is rewritten whenever its content changes, so the output must not
// depend on the order the Sampling CRs came back from the k8s list operation.
func TestCalculateSamplingCategoryRulesForContainer_outputIsOrderIndependent(t *testing.T) {
	ten := 10.0
	twenty := 20.0
	pw := samplingTestPodWorkload()

	newSampling := func(route string, noisyPercentage float64, relevantPercentage float64, costPercentage float64) odigosv1.Sampling {
		return odigosv1.Sampling{
			Spec: odigosv1.SamplingSpec{
				NoisyOperations: []odigosv1.NoisyOperation{{
					Name: "noisy " + route,
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: route},
					},
					PercentageAtMost: &noisyPercentage,
				}},
				HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{{
					Name: "relevant " + route,
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
					},
					PercentageAtLeast: &relevantPercentage,
				}},
				CostReductionRules: []odigosv1.CostReductionRule{{
					Name: "cost " + route,
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
					},
					PercentageAtMost: costPercentage,
				}},
			},
		}
	}

	ascending := []odigosv1.Sampling{newSampling("/a", ten, ten, ten), newSampling("/b", twenty, twenty, twenty)}
	descending := []odigosv1.Sampling{ascending[1], ascending[0]}

	noisyAsc, relevantAsc, costAsc := CalculateSamplingCategoryRulesForContainer(
		&ascending, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})
	noisyDesc, relevantDesc, costDesc := CalculateSamplingCategoryRulesForContainer(
		&descending, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Equal(t, noisyAsc, noisyDesc)
	require.Equal(t, relevantAsc, relevantDesc)
	require.Equal(t, costAsc, costDesc)

	// lower percentage first
	require.Equal(t, []string{"noisy /a", "noisy /b"}, []string{noisyAsc[0].Name, noisyAsc[1].Name})
	require.Equal(t, []string{"relevant /a", "relevant /b"}, []string{relevantAsc[0].Name, relevantAsc[1].Name})
	require.Equal(t, []string{"cost /a", "cost /b"}, []string{costAsc[0].Name, costAsc[1].Name})
}

// an unset percentage means "keep nothing", so such rules sort before the rules that keep a
// non zero percentage.
func TestCalculateSamplingCategoryRulesForContainer_unsetPercentageSortsAsZero(t *testing.T) {
	twenty := 20.0
	pw := samplingTestPodWorkload()

	samplingRules := []odigosv1.Sampling{{
		Spec: odigosv1.SamplingSpec{
			NoisyOperations: []odigosv1.NoisyOperation{
				{
					Name: "keep 20 percent",
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/a"},
					},
					PercentageAtMost: &twenty,
				},
				{
					Name: "unset percentage",
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/b"},
					},
				},
			},
			HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{
				{
					Name: "keep 20 percent",
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/a"},
					},
					PercentageAtLeast: &twenty,
				},
				{
					Name: "unset percentage",
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/b"},
					},
				},
			},
		},
	}}

	noisyOps, relevantOps, _ := CalculateSamplingCategoryRulesForContainer(
		&samplingRules, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Equal(t, []string{"unset percentage", "keep 20 percent"}, []string{noisyOps[0].Name, noisyOps[1].Name})
	require.Equal(t, []string{"unset percentage", "keep 20 percent"}, []string{relevantOps[0].Name, relevantOps[1].Name})
}

// rules with the same percentage are ordered by their id, which is derived from the rule
// content, so the order is the same on every reconcile.
func TestCalculateSamplingCategoryRulesForContainer_equalPercentagesSortedById(t *testing.T) {
	pw := samplingTestPodWorkload()
	routes := []string{"/a", "/b", "/c", "/d"}

	newSampling := func(route string) odigosv1.Sampling {
		return odigosv1.Sampling{
			Spec: odigosv1.SamplingSpec{
				NoisyOperations: []odigosv1.NoisyOperation{{
					Name: route,
					Operation: &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: route},
					},
				}},
				HighlyRelevantOperations: []odigosv1.HighlyRelevantOperation{{
					Name: route,
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
					},
				}},
				CostReductionRules: []odigosv1.CostReductionRule{{
					Name: route,
					Operation: &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: route},
					},
				}},
			},
		}
	}

	forward := make([]odigosv1.Sampling, 0, len(routes))
	for _, route := range routes {
		forward = append(forward, newSampling(route))
	}
	reversed := slices.Clone(forward)
	slices.Reverse(reversed)

	noisyOps, relevantOps, costRules := CalculateSamplingCategoryRulesForContainer(
		&forward, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})
	reversedNoisyOps, reversedRelevantOps, reversedCostRules := CalculateSamplingCategoryRulesForContainer(
		&reversed, common.JavaProgrammingLanguage, pw, "app",
		samplingTestDistro(true, true), nil, &common.OdigosConfiguration{})

	require.Equal(t, noisyOps, reversedNoisyOps)
	require.Equal(t, relevantOps, reversedRelevantOps)
	require.Equal(t, costRules, reversedCostRules)

	require.True(t, slices.IsSortedFunc(noisyOps, func(a, b commonapisampling.NoisyOperation) int {
		return strings.Compare(a.Id, b.Id)
	}))
	require.True(t, slices.IsSortedFunc(relevantOps, func(a, b commonapisampling.HighlyRelevantOperation) int {
		return strings.Compare(a.Id, b.Id)
	}))
	require.True(t, slices.IsSortedFunc(costRules, func(a, b commonapisampling.CostReductionRule) int {
		return strings.Compare(a.Id, b.Id)
	}))
}
