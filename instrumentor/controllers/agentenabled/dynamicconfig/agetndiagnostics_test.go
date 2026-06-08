package dynamicconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros"
	"github.com/stretchr/testify/require"
)

func TestCalculateAgentDiagnosticsForNodejsUsesMostVerboseLevels(t *testing.T) {
	getter, err := distros.NewCommunityGetter()
	require.NoError(t, err)
	nodejs := getter.GetDistroByName("nodejs-community")
	require.NotNil(t, nodejs)

	infoLevel := common.LogLevelInfo
	warnLevel := common.LogLevelWarn
	debugLevel := common.LogLevelDebug
	instrumentationRules := []odigosv1.InstrumentationRule{
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				AgentDiagnostics: &instrumentationrules.AgentDiagnostics{
					OdigosLogLevel:                  &debugLevel,
					OpenTelemetryComponentsLogLevel: &warnLevel,
				},
			},
		},
		{
			Spec: odigosv1.InstrumentationRuleSpec{
				AgentDiagnostics: &instrumentationrules.AgentDiagnostics{
					OdigosLogLevel:                  &infoLevel,
					OpenTelemetryComponentsLogLevel: &debugLevel,
				},
			},
		},
	}

	got := CalculateAgentDiagnostics(&instrumentationRules, nodejs)

	require.NotNil(t, got)
	require.NotNil(t, got.OdigosLogLevel)
	require.Equal(t, common.LogLevelDebug, *got.OdigosLogLevel)
	require.NotNil(t, got.OpenTelemetryComponentsLogLevel)
	require.Equal(t, common.LogLevelDebug, *got.OpenTelemetryComponentsLogLevel)
}
