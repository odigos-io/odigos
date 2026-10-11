package agentenabled

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonopamp "github.com/odigos-io/odigos/common/opamp"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func rubyLegacyWebhookDistro() *distro.OtelDistro {
	device := "instrumentation.odigos.io/generic"
	return &distro.OtelDistro{
		Name:            "ruby-community-legacy",
		Language:        common.RubyProgrammingLanguage,
		ConfigAsEnvVars: true,
		EnvironmentVariables: distro.EnvironmentVariables{
			OpAmpClientEnvironments: true,
		},
		RuntimeAgent: &distro.RuntimeAgent{
			DirectoryNames:           []string{"{{ODIGOS_AGENTS_DIR}}/ruby-legacy"},
			Device:                   &device,
			K8sAttrsViaEnvVars:       true,
			OpAmpTransportsSupported: []commonopamp.OpAmpTransport{commonopamp.OpAmpTransportUnix, commonopamp.OpAmpTransportHTTP},
		},
		Traces: &distro.Traces{
			PayloadCollection: &distro.PayloadCollection{Supported: true},
			HeadSampling: &distro.HeadSampling{
				Supported:                true,
				HttpQueryParamsSupported: true,
			},
		},
	}
}

// The Ruby 2.7 agent applies payload collection and head sampling updates over
// OpAMP, so the webhook has to give it a transport plus the identity attributes
// the OpAMP server resolves the workload from.
func TestInjectOdigosToContainer_opampEnvVarsForRubyLegacy(t *testing.T) {
	legacy := rubyLegacyWebhookDistro()

	for _, tc := range []struct {
		name         string
		mountMethod  common.MountMethod
		expectedEnv  string
		forbiddenEnv string
	}{
		{
			name:         "host path prefers the unix socket",
			mountMethod:  common.K8sHostPathMountMethod,
			expectedEnv:  k8sconsts.OpampUnixSocketEnvName,
			forbiddenEnv: commonconsts.OpampServerHostEnvName,
		},
		{
			name:         "init container falls back to http",
			mountMethod:  common.K8sInitContainerMountMethod,
			expectedEnv:  commonconsts.OpampServerHostEnvName,
			forbiddenEnv: k8sconsts.OpampUnixSocketEnvName,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mount := tc.mountMethod
			cfg := common.OdigosConfiguration{MountMethod: &mount}
			container := &corev1.Container{Name: "app"}
			containerConfig := &odigosv1.ContainerAgentConfig{
				ContainerName:  "app",
				AgentEnabled:   true,
				OtelDistroName: "ruby-community-legacy",
				DistroParams: map[string]string{
					distro.RuntimeVersionMajorMinorDistroParameterName: "2.7",
				},
			}
			pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
			_, _, err := (&PodsWebhook{}).injectOdigosToContainer(containerConfig, container, &odigosv1.InstrumentationConfig{}, pw, "app", cfg, legacy, nil)
			require.NoError(t, err)

			envNames := make(map[string]struct{}, len(container.Env))
			for _, env := range container.Env {
				envNames[env.Name] = struct{}{}
			}

			require.Contains(t, envNames, tc.expectedEnv)
			require.NotContains(t, envNames, tc.forbiddenEnv)
			for _, name := range []string{
				k8sconsts.OdigosEnvVarPodName,
				k8sconsts.OdigosEnvVarContainerName,
				k8sconsts.OdigosEnvVarNamespace,
			} {
				require.Contains(t, envNames, name, "OpAMP client needs %s to identify the workload", name)
			}
		})
	}
}
