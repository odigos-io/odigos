package agentenabled

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInjectOdigosToContainer_payloadCollectionEnvVarForRubyLegacy(t *testing.T) {
	legacy := rubyLegacyWebhookDistro()

	maxLen := int64(1024)
	payloadCfg := &instrumentationrules.PayloadCollection{
		DbQuery: &instrumentationrules.DbQueryPayloadCollection{
			MaxPayloadLength: &maxLen,
		},
		HttpRequest: &instrumentationrules.HttpPayloadCollection{},
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
			PayloadCollection: payloadCfg,
		},
	}

	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	ic := &odigosv1.InstrumentationConfig{}
	webhook := &PodsWebhook{}
	_, _, err := webhook.injectOdigosToContainer(containerConfig, container, ic, pw, "app", cfg, legacy, nil)
	require.NoError(t, err)

	var got string
	for _, env := range container.Env {
		if env.Name == k8sconsts.OdigosAgentPayloadCollectionEnvVar {
			got = env.Value
			break
		}
	}
	require.NotEmpty(t, got, "expected %s to be injected", k8sconsts.OdigosAgentPayloadCollectionEnvVar)

	var parsed instrumentationrules.PayloadCollection
	require.NoError(t, json.Unmarshal([]byte(got), &parsed))
	require.NotNil(t, parsed.DbQuery)
	require.NotNil(t, parsed.DbQuery.MaxPayloadLength)
	require.Equal(t, maxLen, *parsed.DbQuery.MaxPayloadLength)
	require.NotNil(t, parsed.HttpRequest)
}

func TestInjectOdigosToContainer_skipsPayloadCollectionWhenUnsupported(t *testing.T) {
	mount := common.K8sHostPathMountMethod
	cfg := common.OdigosConfiguration{MountMethod: &mount}
	container := &corev1.Container{Name: "app"}
	device := "instrumentation.odigos.io/generic"
	distroMetadata := &distro.OtelDistro{
		Name:            "no-payloads",
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
			PayloadCollection: &instrumentationrules.PayloadCollection{
				DbQuery: &instrumentationrules.DbQueryPayloadCollection{},
			},
		},
	}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	webhook := &PodsWebhook{}
	_, _, err := webhook.injectOdigosToContainer(containerConfig, container, &odigosv1.InstrumentationConfig{}, pw, "app", cfg, distroMetadata, []metav1.OwnerReference{})
	require.NoError(t, err)

	for _, env := range container.Env {
		require.NotEqual(t, k8sconsts.OdigosAgentPayloadCollectionEnvVar, env.Name)
	}
}
