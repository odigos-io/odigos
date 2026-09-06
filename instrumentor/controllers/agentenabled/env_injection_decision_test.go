package agentenabled

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The path the webhook preloads for loader injection. Hard coded on purpose: a container that
// already has this value in LD_PRELOAD is one odigos itself injected, and must not be treated as a
// conflict.
const odigosLoaderPathInContainer = "/var/odigos/loader/loader.so"

func injectionMethodPtr(m common.EnvInjectionMethod) *common.EnvInjectionMethod {
	return &m
}

func envInjectionDecisionPtr(d common.EnvInjectionDecision) *common.EnvInjectionDecision {
	return &d
}

// ldPreloadDistro is a distro whose agent can be loaded via LD_PRELOAD, optionally also declaring
// "append" env vars (nodejs style, where both injection methods are possible).
func ldPreloadDistro(appendEnvVarNames ...string) *distro.OtelDistro {
	d := appendEnvVarsDistro(appendEnvVarNames...)
	d.RuntimeAgent = &distro.RuntimeAgent{LdPreloadInjectionSupported: true}
	return d
}

// appendEnvVarsDistro is a distro with no LD_PRELOAD support. Without any "append" env var it
// models the distros that need no env injection at all (golang, dotnet, php, ruby).
func appendEnvVarsDistro(appendEnvVarNames ...string) *distro.OtelDistro {
	d := &distro.OtelDistro{
		Name:     "test-distro",
		Language: common.JavascriptProgrammingLanguage,
	}
	for _, envName := range appendEnvVarNames {
		d.EnvironmentVariables.AppendOdigosVariables = append(d.EnvironmentVariables.AppendOdigosVariables,
			distro.AppendOdigosEnvironmentVariable{EnvName: envName})
	}
	return d
}

func runtimeDetailsForInjection(secureExecutionMode *bool, ldPreloadValue *string) *odigosv1.RuntimeDetailsByContainer {
	rd := &odigosv1.RuntimeDetailsByContainer{
		ContainerName:       "app",
		Language:            common.JavascriptProgrammingLanguage,
		SecureExecutionMode: secureExecutionMode,
	}
	if ldPreloadValue != nil {
		rd.EnvVars = []odigosv1.EnvVar{
			{Name: "SOME_OTHER_ENV", Value: "some-value"},
			{Name: commonconsts.LdPreloadEnvVarName, Value: *ldPreloadValue},
		}
	}
	return rd
}

// getEnvInjectionDecision picks how the agent env vars reach the application. Choosing the loader
// where it is not usable silently breaks the application (LD_PRELOAD of another agent is dropped,
// or the process refuses to preload under secure execution), while falling back where the loader
// is fine loses the restart-free injection. Both directions are covered here.
func TestGetEnvInjectionDecision(t *testing.T) {
	notSecure := false
	secure := true
	foreignLdPreload := "/opt/other-agent/agent.so"
	odigosLdPreload := odigosLoaderPathInContainer
	odigosAndForeignLdPreload := foreignLdPreload + ":" + odigosLoaderPathInContainer

	for _, tc := range []struct {
		name             string
		injectionMethod  *common.EnvInjectionMethod
		distro           *distro.OtelDistro
		runtimeDetails   *odigosv1.RuntimeDetailsByContainer
		expectedDecision *common.EnvInjectionDecision
		expectDisabled   bool
		messageContains  string
	}{
		{
			// the config is defaulted during reconciliation, so this is only a safety net.
			name:            "no configured injection method disables the agent",
			injectionMethod: nil,
			distro:          ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:  runtimeDetailsForInjection(&notSecure, nil),
			expectDisabled:  true,
			messageContains: "no injection method configured",
		},
		{
			name:             "loader is used when the distro supports it and the runtime allows it",
			injectionMethod:  injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, nil),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionLoader),
		},
		{
			name:             "loader is preferred over the pod manifest fallback",
			injectionMethod:  injectionMethodPtr(common.LoaderFallbackToPodManifestInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, nil),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionLoader),
		},
		{
			name:             "an already injected odigos loader is not a conflict",
			injectionMethod:  injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, &odigosLdPreload),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionLoader),
		},
		{
			// odigos already preloads next to another agent in this container, so re-deciding the
			// injection method must not flip to a conflict and drop the instrumentation.
			name:             "odigos loader alongside another preloaded library is not a conflict",
			injectionMethod:  injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, &odigosAndForeignLdPreload),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionLoader),
		},
		{
			name:            "a foreign LD_PRELOAD fails the strict loader method",
			injectionMethod: injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:          ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:  runtimeDetailsForInjection(&notSecure, &foreignLdPreload),
			expectDisabled:  true,
			messageContains: foreignLdPreload,
		},
		{
			name:             "a foreign LD_PRELOAD falls back to the pod manifest",
			injectionMethod:  injectionMethodPtr(common.LoaderFallbackToPodManifestInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, &foreignLdPreload),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionPodManifest),
		},
		{
			name:            "secure execution mode fails the strict loader method",
			injectionMethod: injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:          ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:  runtimeDetailsForInjection(&secure, nil),
			expectDisabled:  true,
			messageContains: "secure execution mode",
		},
		{
			// an old odiglet does not report the field, and preloading into a setuid process is
			// unsafe, so unknown must be treated as secure execution.
			name:            "unknown secure execution mode is assumed to be secure",
			injectionMethod: injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:          ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:  runtimeDetailsForInjection(nil, nil),
			expectDisabled:  true,
			messageContains: "secure execution mode",
		},
		{
			name:             "secure execution mode falls back to the pod manifest",
			injectionMethod:  injectionMethodPtr(common.LoaderFallbackToPodManifestInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&secure, nil),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionPodManifest),
		},
		{
			// nothing to fall back to: the distro has no env var to append to the manifest.
			name:             "fallback with nothing to append means no injection",
			injectionMethod:  injectionMethodPtr(common.LoaderFallbackToPodManifestInjectionMethod),
			distro:           ldPreloadDistro(),
			runtimeDetails:   runtimeDetailsForInjection(&secure, nil),
			expectedDecision: nil,
		},
		{
			name:             "the pod manifest method never uses the loader",
			injectionMethod:  injectionMethodPtr(common.PodManifestEnvInjectionMethod),
			distro:           ldPreloadDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, nil),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionPodManifest),
		},
		{
			name:             "a distro without loader support uses the pod manifest",
			injectionMethod:  injectionMethodPtr(common.LoaderEnvInjectionMethod),
			distro:           appendEnvVarsDistro("NODE_OPTIONS"),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, nil),
			expectedDecision: envInjectionDecisionPtr(common.EnvInjectionDecisionPodManifest),
		},
		{
			// golang, dotnet, php and ruby: no loader and no env var to append.
			name:             "a distro with no env injection at all is not injected",
			injectionMethod:  injectionMethodPtr(common.PodManifestEnvInjectionMethod),
			distro:           appendEnvVarsDistro(),
			runtimeDetails:   runtimeDetailsForInjection(&notSecure, nil),
			expectedDecision: nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			effectiveConfig := &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: tc.injectionMethod}

			decision, disabledInfo := getEnvInjectionDecision(effectiveConfig, tc.runtimeDetails, tc.distro)

			if tc.expectDisabled {
				require.NotNil(t, disabledInfo, "expected the agent to be disabled")
				assert.Nil(t, decision, "no injection method should be returned when the agent is disabled")
				assert.Equal(t, odigosv1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonInjectionConflict),
					disabledInfo.AgentEnabledReason)
				assert.Contains(t, disabledInfo.AgentEnabledMessage, tc.messageContains)
				return
			}

			require.Nil(t, disabledInfo)
			assert.Equal(t, tc.expectedDecision, decision)
		})
	}
}
