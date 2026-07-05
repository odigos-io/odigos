package traces

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
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
