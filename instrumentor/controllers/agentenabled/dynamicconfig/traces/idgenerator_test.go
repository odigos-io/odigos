package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	agentInjectionEnabled "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/require"
)

func TestTimedWallEnabled(t *testing.T) {
	require.False(t, TimedWallEnabled(&common.OdigosConfiguration{}))
	require.True(t, TimedWallEnabled(&common.OdigosConfiguration{TraceIdSuffix: "00"}))
}

func TestCalculateIdGeneratorConfig_disabled(t *testing.T) {
	config, disabledInfo := CalculateIdGeneratorConfig(&common.OdigosConfiguration{})

	require.Nil(t, config)
	require.Nil(t, disabledInfo)
}

// The suffix is a single hex byte that keeps trace ids of different clusters apart, so both the
// parsing and the byte width matter.
func TestCalculateIdGeneratorConfig_validSuffixes(t *testing.T) {
	tt := []struct {
		suffix   string
		expected uint8
	}{
		{suffix: "0", expected: 0},
		{suffix: "00", expected: 0},
		{suffix: "01", expected: 1},
		{suffix: "a3", expected: 163},
		{suffix: "A3", expected: 163},
		{suffix: "ff", expected: 255},
	}

	for _, tc := range tt {
		t.Run(tc.suffix, func(t *testing.T) {
			config, disabledInfo := CalculateIdGeneratorConfig(&common.OdigosConfiguration{TraceIdSuffix: tc.suffix})

			require.Nil(t, disabledInfo)
			require.NotNil(t, config)
			require.NotNil(t, config.TimedWall)
			require.Equal(t, tc.expected, config.TimedWall.SourceId)
		})
	}
}

// A malformed suffix must disable the agent with an explicit reason rather than silently sending
// trace ids that collide with another cluster's.
func TestCalculateIdGeneratorConfig_invalidSuffixes(t *testing.T) {
	for _, suffix := range []string{"zz", "100", "-1", "0x1f", "a3 ", "٣"} {
		t.Run(suffix, func(t *testing.T) {
			config, disabledInfo := CalculateIdGeneratorConfig(&common.OdigosConfiguration{TraceIdSuffix: suffix})

			require.Nil(t, config)
			require.NotNil(t, disabledInfo)
			require.Equal(t,
				odigosv1.AgentEnabledReason(agentInjectionEnabled.AgentEnabledReasonInjectionConflict),
				disabledInfo.AgentEnabledReason)
			require.Contains(t, disabledInfo.AgentEnabledMessage, "failed to parse trace id suffix")
			require.Contains(t, disabledInfo.AgentEnabledMessage, "single byte hex value")
		})
	}
}
