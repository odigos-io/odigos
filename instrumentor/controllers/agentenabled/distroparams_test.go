package agentenabled

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withRequiredParams(d *distroTypes.OtelDistro, requiredParams ...string) *distroTypes.OtelDistro {
	d.RequireParameters = requiredParams
	return d
}

func runtimeDetailsForParams(rd odigosv1.RuntimeDetailsByContainer) *odigosv1.RuntimeDetailsByContainer {
	rd.ContainerName = "app"
	rd.Language = common.JavascriptProgrammingLanguage
	return &rd
}

// Distro params are the only channel through which runtime inspection results reach the webhook, so
// a missing or malformed value must disable the agent rather than produce an agent that cannot
// load. The CRI sourced "append" values are equally sensitive: dropping one silently discards a
// value the application set (e.g. NODE_OPTIONS from the image) when the webhook rewrites the env.
func TestCalculateDistroParams(t *testing.T) {
	musl := common.Musl
	criError := "failed to read env vars from container runtime"
	podManifestDecision := common.EnvInjectionDecisionPodManifest
	loaderDecision := common.EnvInjectionDecisionLoader

	for _, tc := range []struct {
		name               string
		distro             *distroTypes.OtelDistro
		runtimeDetails     *odigosv1.RuntimeDetailsByContainer
		envInjectionMethod *common.EnvInjectionDecision
		expectedParams     DistroParam
		expectDisabled     bool
		messageContains    string
	}{
		{
			// nil (and not an empty map) keeps the instrumentation config unchanged, which avoids a
			// rollout of every workload on every reconcile.
			name:           "no required params and no env injection produces no params",
			distro:         appendEnvVarsDistro(),
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{}),
			expectedParams: nil,
		},
		{
			name:           "libc type is taken from the runtime details",
			distro:         withRequiredParams(appendEnvVarsDistro(), common.LibcTypeDistroParameterName),
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{LibCType: &musl}),
			expectedParams: DistroParam{common.LibcTypeDistroParameterName: string(common.Musl)},
		},
		{
			name:            "a missing libc type disables the agent",
			distro:          withRequiredParams(appendEnvVarsDistro(), common.LibcTypeDistroParameterName),
			runtimeDetails:  runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{}),
			expectDisabled:  true,
			messageContains: "failed to detect libc type",
		},
		{
			name:           "the runtime version is reduced to major.minor",
			distro:         withRequiredParams(appendEnvVarsDistro(), distroTypes.RuntimeVersionMajorMinorDistroParameterName),
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{RuntimeVersion: "20.11.1"}),
			expectedParams: DistroParam{distroTypes.RuntimeVersionMajorMinorDistroParameterName: "20.11"},
		},
		{
			name:            "a missing runtime version disables the agent",
			distro:          withRequiredParams(appendEnvVarsDistro(), distroTypes.RuntimeVersionMajorMinorDistroParameterName),
			runtimeDetails:  runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{}),
			expectDisabled:  true,
			messageContains: "failed to detect runtime version",
		},
		{
			name:            "an unparsable runtime version disables the agent",
			distro:          withRequiredParams(appendEnvVarsDistro(), distroTypes.RuntimeVersionMajorMinorDistroParameterName),
			runtimeDetails:  runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{RuntimeVersion: "not-a-version"}),
			expectDisabled:  true,
			messageContains: "failed to parse runtime version from detection",
		},
		{
			// a typo in a distro yaml must not produce an agent with a silently missing parameter.
			name:            "an unsupported required parameter disables the agent",
			distro:          withRequiredParams(appendEnvVarsDistro(), "NO_SUCH_PARAMETER"),
			runtimeDetails:  runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{}),
			expectDisabled:  true,
			messageContains: "unsupported parameter 'NO_SUCH_PARAMETER'",
		},
		{
			name:               "append env vars are taken from the container runtime for pod manifest injection",
			distro:             appendEnvVarsDistro("NODE_OPTIONS"),
			envInjectionMethod: &podManifestDecision,
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: "NODE_OPTIONS", Value: "--max-old-space-size=4096"}},
			}),
			expectedParams: DistroParam{"NODE_OPTIONS": "--max-old-space-size=4096"},
		},
		{
			// the container runtime value is only relevant when odigos rewrites the pod manifest.
			name:               "append env vars are not collected for loader injection",
			distro:             appendEnvVarsDistro("NODE_OPTIONS"),
			envInjectionMethod: &loaderDecision,
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: "NODE_OPTIONS", Value: "--max-old-space-size=4096"}},
			}),
			expectedParams: nil,
		},
		{
			name:               "an append env var missing from the container runtime is not set",
			distro:             appendEnvVarsDistro("NODE_OPTIONS"),
			envInjectionMethod: &podManifestDecision,
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: "SOME_OTHER_ENV", Value: "some-value"}},
			}),
			expectedParams: nil,
		},
		{
			// an unreadable container runtime means odigos cannot tell whether it would overwrite a
			// value the image set, so the agent must not be injected.
			name:               "a container runtime error disables the agent",
			distro:             appendEnvVarsDistro("NODE_OPTIONS"),
			envInjectionMethod: &podManifestDecision,
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{
				CriErrorMessage: &criError,
			}),
			expectDisabled:  true,
			messageContains: criError,
		},
		{
			name:               "required and append params are merged",
			distro:             withRequiredParams(appendEnvVarsDistro("NODE_OPTIONS"), common.LibcTypeDistroParameterName),
			envInjectionMethod: &podManifestDecision,
			runtimeDetails: runtimeDetailsForParams(odigosv1.RuntimeDetailsByContainer{
				LibCType:                &musl,
				EnvFromContainerRuntime: []odigosv1.EnvVar{{Name: "NODE_OPTIONS", Value: "--enable-source-maps"}},
			}),
			expectedParams: DistroParam{
				common.LibcTypeDistroParameterName: string(common.Musl),
				"NODE_OPTIONS":                     "--enable-source-maps",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			params, disabledConfig := calculateDistroParams(tc.distro, tc.runtimeDetails, tc.envInjectionMethod)

			if tc.expectDisabled {
				require.NotNil(t, disabledConfig, "expected the agent to be disabled")
				assert.False(t, disabledConfig.AgentEnabled)
				assert.Equal(t, "app", disabledConfig.ContainerName)
				assert.Equal(t, odigosv1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonMissingDistroParameter),
					disabledConfig.AgentEnabledReason)
				assert.Contains(t, disabledConfig.AgentEnabledMessage, tc.messageContains)
				return
			}

			require.Nil(t, disabledConfig)
			assert.Equal(t, tc.expectedParams, params)
		})
	}
}
