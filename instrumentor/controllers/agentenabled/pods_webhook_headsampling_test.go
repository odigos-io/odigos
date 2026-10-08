package agentenabled

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectOdigosToContainer_headSamplingEnvVarForRubyLegacy(t *testing.T) {
	legacy := rubyLegacyWebhookDistro()

	pct := 0.0
	queryValue := "liveness"
	headSamplingCfg := &sampling.HeadSamplingConfig{
		DryRun: false,
		NoisyOperations: []sampling.NoisyOperation{
			{
				Id:               "health-probe",
				PercentageAtMost: &pct,
				Operation: &sampling.HeadSamplingOperationMatcher{
					HttpServer: &sampling.HeadSamplingHttpServerOperationMatcher{
						Route:  "/health",
						Method: "GET",
						QueryParams: []sampling.QueryParamMatcher{
							{Name: "type", ValueExact: &queryValue},
						},
					},
				},
			},
		},
	}

	mount := common.K8sHostPathMountMethod
	cfg := common.OdigosConfiguration{MountMethod: &mount}
	container := &corev1.Container{Name: "app"}
	containerConfig := &odigosv1.ContainerAgentConfig{
		ContainerName:  "app",
		AgentEnabled:   true,
		OtelDistroName: "ruby-community-legacy",
		DistroParams: map[string]string{
			distro.RuntimeVersionMajorMinorDistroParameterName: "2.7",
		},
		Traces: &agentsignalconfig.AgentTracesConfig{
			HeadSampling: headSamplingCfg,
		},
	}

	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	ic := &odigosv1.InstrumentationConfig{}
	webhook := &PodsWebhook{}
	_, _, err := webhook.injectOdigosToContainer(containerConfig, container, ic, pw, "app", cfg, legacy, nil)
	require.NoError(t, err)

	var got string
	for _, env := range container.Env {
		if env.Name == "ODIGOS_AGENT_HEAD_SAMPLING" {
			got = env.Value
			break
		}
	}
	require.NotEmpty(t, got, "expected ODIGOS_AGENT_HEAD_SAMPLING to be injected")

	var parsed sampling.HeadSamplingConfig
	require.NoError(t, json.Unmarshal([]byte(got), &parsed))
	require.Len(t, parsed.NoisyOperations, 1)
	require.Equal(t, "health-probe", parsed.NoisyOperations[0].Id)
	require.NotNil(t, parsed.NoisyOperations[0].PercentageAtMost)
	require.Equal(t, pct, *parsed.NoisyOperations[0].PercentageAtMost)
	require.NotNil(t, parsed.NoisyOperations[0].Operation)
	require.NotNil(t, parsed.NoisyOperations[0].Operation.HttpServer)
	require.Equal(t, "/health", parsed.NoisyOperations[0].Operation.HttpServer.Route)
	require.Equal(t, "GET", parsed.NoisyOperations[0].Operation.HttpServer.Method)
	require.Len(t, parsed.NoisyOperations[0].Operation.HttpServer.QueryParams, 1)
	require.Equal(t, "type", parsed.NoisyOperations[0].Operation.HttpServer.QueryParams[0].Name)
	require.NotNil(t, parsed.NoisyOperations[0].Operation.HttpServer.QueryParams[0].ValueExact)
	require.Equal(t, queryValue, *parsed.NoisyOperations[0].Operation.HttpServer.QueryParams[0].ValueExact)
}

func TestInjectOdigosToContainer_skipsHeadSamplingWhenUnsupported(t *testing.T) {
	pct := 0.0
	mount := common.K8sHostPathMountMethod
	cfg := common.OdigosConfiguration{MountMethod: &mount}
	container := &corev1.Container{Name: "app"}
	device := "instrumentation.odigos.io/generic"
	distroMetadata := &distro.OtelDistro{
		Name:            "no-head-sampling",
		ConfigAsEnvVars: true,
		RuntimeAgent: &distro.RuntimeAgent{
			DirectoryNames:     []string{"{{ODIGOS_AGENTS_DIR}}/none"},
			Device:             &device,
			K8sAttrsViaEnvVars: true,
		},
	}
	containerConfig := &odigosv1.ContainerAgentConfig{
		ContainerName: "app",
		AgentEnabled:  true,
		Traces: &agentsignalconfig.AgentTracesConfig{
			HeadSampling: &sampling.HeadSamplingConfig{
				NoisyOperations: []sampling.NoisyOperation{
					{Id: "drop-all", PercentageAtMost: &pct},
				},
			},
		},
	}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	webhook := &PodsWebhook{}
	_, _, err := webhook.injectOdigosToContainer(containerConfig, container, &odigosv1.InstrumentationConfig{}, pw, "app", cfg, distroMetadata, []metav1.OwnerReference{})
	require.NoError(t, err)

	for _, env := range container.Env {
		require.NotEqual(t, "ODIGOS_AGENT_HEAD_SAMPLING", env.Name)
	}
}
