package agentenabled

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func TestCalculateContainerAgentConfig_LegacyOtherAgentStillBlocks(t *testing.T) {
	injection := common.PodManifestEnvInjectionMethod
	cfg := &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &injection}
	d := &distro.OtelDistro{Name: "test-distro"}
	rd := &odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.JavaProgrammingLanguage,
		// Pre-upgrade status shape: only the deprecated singular field is set.
		OtherAgent: &odigosv1.OtherAgent{Name: "Datadog Agent"},
	}

	got := calculateContainerAgentConfig("app", d, cfg, rd, false, "", false)
	require.False(t, got.AgentEnabled)
	require.Equal(t, odigosv1.AgentEnabledReasonOtherAgentDetected, got.AgentEnabledReason)
	require.Contains(t, got.AgentEnabledMessage, "Datadog Agent")
}

func TestCalculateContainerAgentConfig_PluralOtherAgentsStillBlocks(t *testing.T) {
	injection := common.PodManifestEnvInjectionMethod
	cfg := &common.OdigosConfiguration{AgentEnvVarsInjectionMethod: &injection}
	d := &distro.OtelDistro{Name: "test-distro"}
	rd := &odigosv1.RuntimeDetailsByContainer{
		ContainerName: "app",
		Language:      common.JavaProgrammingLanguage,
		OtherAgents:   []odigosv1.OtherAgent{{Name: "Datadog Agent"}},
	}

	got := calculateContainerAgentConfig("app", d, cfg, rd, false, "", false)
	require.False(t, got.AgentEnabled)
	require.Equal(t, odigosv1.AgentEnabledReasonOtherAgentDetected, got.AgentEnabledReason)
}
