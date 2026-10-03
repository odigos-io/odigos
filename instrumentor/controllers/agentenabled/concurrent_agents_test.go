package agentenabled

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Concurrent agents mean two instrumentation agents loaded into the same process, which can crash
// the workload. The container override is the only way to opt a single container in or out, so it
// must win over the cluster wide config in both directions.
func TestResolveAllowConcurrentAgents(t *testing.T) {
	boolPtr := func(b bool) *bool { return &b }

	for _, tc := range []struct {
		name              string
		globalAllow       *bool
		containerOverride *odigosv1.ContainerOverride
		expected          bool
	}{
		{"unset global and no override blocks concurrent agents", nil, nil, false},
		{"global true and no override allows", boolPtr(true), nil, true},
		{"global false and no override blocks", boolPtr(false), nil, false},
		{"override true wins over unset global", nil, &odigosv1.ContainerOverride{AllowConcurrentAgents: boolPtr(true)}, true},
		{"override true wins over global false", boolPtr(false), &odigosv1.ContainerOverride{AllowConcurrentAgents: boolPtr(true)}, true},
		{"override false wins over global true", boolPtr(true), &odigosv1.ContainerOverride{AllowConcurrentAgents: boolPtr(false)}, false},
		{"override without the field falls back to global", boolPtr(true), &odigosv1.ContainerOverride{ContainerName: "app"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &common.OdigosConfiguration{AllowConcurrentAgents: tc.globalAllow}

			got := resolveAllowConcurrentAgents(cfg, tc.containerOverride)
			assert.Equal(t, tc.expected, got)
		})
	}
}

// A distro that needs neither required parameters, a runtime agent, nor appended env vars, so the
// resulting agent config is decided only by the other-agents check.
func newMinimalDistro() *distro.OtelDistro {
	return &distro.OtelDistro{
		Name:     "test-distro",
		Language: common.JavascriptProgrammingLanguage,
	}
}

func newRuntimeDetailsWithOtherAgents(agentNames ...string) *odigosv1.RuntimeDetailsByContainer {
	otherAgents := make([]odigosv1.OtherAgent, 0, len(agentNames))
	for _, name := range agentNames {
		otherAgents = append(otherAgents, odigosv1.OtherAgent{Name: name})
	}

	return &odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.JavascriptProgrammingLanguage,
		OtherAgents:   otherAgents,
	}
}

func newPodManifestConfig() *common.OdigosConfiguration {
	injectionMethod := common.PodManifestEnvInjectionMethod
	return &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &injectionMethod}
}

func TestCalculateContainerAgentConfig_OtherAgentDetected(t *testing.T) {
	agentConfig := calculateContainerAgentConfig("app", newMinimalDistro(), newPodManifestConfig(),
		newRuntimeDetailsWithOtherAgents("datadog", "newrelic"), false, "", false)

	assert.False(t, agentConfig.AgentEnabled, "odigos must not inject alongside a detected agent by default")
	assert.Equal(t, odigosv1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonOtherAgentDetected),
		agentConfig.AgentEnabledReason)
	assert.Contains(t, agentConfig.AgentEnabledMessage, "datadog")
	assert.Contains(t, agentConfig.AgentEnabledMessage, "newrelic")
	assert.Empty(t, agentConfig.OtelDistroName, "no distro should be selected when the agent is not enabled")
}

func TestCalculateContainerAgentConfig_OtherAgentDetectedAndConcurrentAgentsAllowed(t *testing.T) {
	distroUnderTest := newMinimalDistro()

	agentConfig := calculateContainerAgentConfig("app", distroUnderTest, newPodManifestConfig(),
		newRuntimeDetailsWithOtherAgents("datadog"), false, "", true)

	require.True(t, agentConfig.AgentEnabled, "opting in must inject alongside the detected agent")
	assert.Equal(t, odigosv1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonEnabledWithOtherAgents),
		agentConfig.AgentEnabledReason, "the user must still be warned about the unsupported configuration")
	assert.Contains(t, agentConfig.AgentEnabledMessage, "datadog")
	assert.Equal(t, distroUnderTest.Name, agentConfig.OtelDistroName)
}

// The other-agents branch is the only one that sets a reason on an enabled agent, so a container
// with no other agents must stay reason-free regardless of the concurrent agents setting.
func TestCalculateContainerAgentConfig_NoOtherAgents(t *testing.T) {
	for _, allowConcurrentAgents := range []bool{false, true} {
		agentConfig := calculateContainerAgentConfig("app", newMinimalDistro(), newPodManifestConfig(),
			newRuntimeDetailsWithOtherAgents(), false, "", allowConcurrentAgents)

		require.True(t, agentConfig.AgentEnabled)
		assert.Empty(t, agentConfig.AgentEnabledReason)
		assert.Empty(t, agentConfig.AgentEnabledMessage)
	}
}
