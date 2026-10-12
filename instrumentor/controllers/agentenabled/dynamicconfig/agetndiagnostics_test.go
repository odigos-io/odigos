package dynamicconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func diagnosticsDistro(odigosAgentOwnLogger bool, openTelemetryComponentsLogger bool) *distro.OtelDistro {
	return &distro.OtelDistro{
		OwnDiagnostics: &distro.OwnDiagnostics{
			OdigosAgentOwnLogerSupported:           odigosAgentOwnLogger,
			OpenTelemetryComponentsLoggerSupported: openTelemetryComponentsLogger,
		},
	}
}

func diagnosticsRule(odigosLogLevel *common.OdigosLogLevel, openTelemetryComponentsLogLevel *common.OdigosLogLevel) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			AgentDiagnostics: &instrumentationrules.AgentDiagnostics{
				OdigosLogLevel:                  odigosLogLevel,
				OpenTelemetryComponentsLogLevel: openTelemetryComponentsLogLevel,
			},
		},
	}
}

func TestDistroSupportsAgentDiagnostics(t *testing.T) {
	t.Parallel()

	require.False(t, DistroSupportsAgentDiagnostics(&distro.OtelDistro{}))
	require.True(t, DistroSupportsAgentDiagnostics(&distro.OtelDistro{OwnDiagnostics: &distro.OwnDiagnostics{}}))
}

// Asking an agent for a logger it does not have is a config the agent cannot honor,
// so the distro support flags gate each logger independently.
//
// Which log level wins when two rules request different ones is deliberately not asserted:
// common.OdigosLogLevel.Compare ignores its argument, so mergeLogLevel currently keeps the
// last rule instead of the most verbose one that the comment above it describes.
func TestCalculateAgentDiagnostics(t *testing.T) {
	t.Parallel()

	infoLevel := common.LogLevelInfo
	debugLevel := common.LogLevelDebug

	tests := []struct {
		name     string
		distro   *distro.OtelDistro
		irls     []odigosv1.InstrumentationRule
		expected *instrumentationrules.AgentDiagnostics
	}{
		{
			name:     "distro does not support agent diagnostics",
			distro:   &distro.OtelDistro{},
			irls:     []odigosv1.InstrumentationRule{diagnosticsRule(&debugLevel, &debugLevel)},
			expected: nil,
		},
		{
			name:     "no rules",
			distro:   diagnosticsDistro(true, true),
			irls:     []odigosv1.InstrumentationRule{},
			expected: nil,
		},
		{
			name:     "rules without agent diagnostics",
			distro:   diagnosticsDistro(true, true),
			irls:     []odigosv1.InstrumentationRule{{Spec: odigosv1.InstrumentationRuleSpec{}}},
			expected: nil,
		},
		{
			name:   "a single rule configures both loggers",
			distro: diagnosticsDistro(true, true),
			irls:   []odigosv1.InstrumentationRule{diagnosticsRule(&debugLevel, &infoLevel)},
			expected: &instrumentationrules.AgentDiagnostics{
				OdigosLogLevel:                  &debugLevel,
				OpenTelemetryComponentsLogLevel: &infoLevel,
			},
		},
		{
			name:   "rules without a diagnostics config are skipped",
			distro: diagnosticsDistro(true, true),
			irls: []odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{}},
				diagnosticsRule(&debugLevel, &infoLevel),
				{Spec: odigosv1.InstrumentationRuleSpec{}},
			},
			expected: &instrumentationrules.AgentDiagnostics{
				OdigosLogLevel:                  &debugLevel,
				OpenTelemetryComponentsLogLevel: &infoLevel,
			},
		},
		{
			name:   "a rule that leaves a logger unset does not clear it",
			distro: diagnosticsDistro(true, true),
			irls: []odigosv1.InstrumentationRule{
				diagnosticsRule(&debugLevel, nil),
				diagnosticsRule(nil, &infoLevel),
			},
			expected: &instrumentationrules.AgentDiagnostics{
				OdigosLogLevel:                  &debugLevel,
				OpenTelemetryComponentsLogLevel: &infoLevel,
			},
		},
		{
			name:   "odigos own logger not supported by the distro is dropped",
			distro: diagnosticsDistro(false, true),
			irls: []odigosv1.InstrumentationRule{
				diagnosticsRule(&debugLevel, &infoLevel),
				diagnosticsRule(&debugLevel, &infoLevel),
			},
			expected: &instrumentationrules.AgentDiagnostics{OpenTelemetryComponentsLogLevel: &infoLevel},
		},
		{
			name:   "opentelemetry components logger not supported by the distro is dropped",
			distro: diagnosticsDistro(true, false),
			irls: []odigosv1.InstrumentationRule{
				diagnosticsRule(&debugLevel, &infoLevel),
				diagnosticsRule(&debugLevel, &infoLevel),
			},
			expected: &instrumentationrules.AgentDiagnostics{OdigosLogLevel: &debugLevel},
		},
		{
			name:   "no logger supported by the distro leaves an empty config",
			distro: diagnosticsDistro(false, false),
			irls: []odigosv1.InstrumentationRule{
				diagnosticsRule(&debugLevel, &infoLevel),
				diagnosticsRule(&debugLevel, &infoLevel),
			},
			expected: &instrumentationrules.AgentDiagnostics{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			irls := tt.irls
			require.Equal(t, tt.expected, CalculateAgentDiagnostics(&irls, tt.distro))
		})
	}
}
