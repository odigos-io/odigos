package signals

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/stretchr/testify/require"
)

func nodeCollectorsGroupWithSignals(receiverSignals ...common.ObservabilitySignal) *odigosv1.CollectorsGroup {
	return &odigosv1.CollectorsGroup{
		Status: odigosv1.CollectorsGroupStatus{
			ReceiverSignals: receiverSignals,
		},
	}
}

func tracesDisablingRule() odigosv1.InstrumentationRule {
	disabled := true
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			TraceConfig: &instrumentationrules.TraceConfig{
				Disabled: &disabled,
			},
		},
	}
}

// a container with no enabled signal must not be instrumented at all,
// and the caller relies on the returned disabled info to report why.
func TestGetEnabledSignalsForContainer_NoSignalsDisablesTheAgent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		nodeCollectorsGroup *odigosv1.CollectorsGroup
		irls                []odigosv1.InstrumentationRule
		expectedReason      string
		expectedMessage     string
	}{
		{
			name:                "node collectors group not created yet",
			nodeCollectorsGroup: nil,
			expectedReason:      "WaitingForNodeCollector",
			expectedMessage:     "waiting for OpenTelemetry Collector to be created",
		},
		{
			name:                "collectors group receives no signals",
			nodeCollectorsGroup: nodeCollectorsGroupWithSignals(),
			expectedReason:      "NoCollectedSignals",
			expectedMessage:     "all signals are disabled, no agent will be injected",
		},
		{
			name:                "traces is the only collected signal and a rule disables it",
			nodeCollectorsGroup: nodeCollectorsGroupWithSignals(common.TracesObservabilitySignal),
			irls:                []odigosv1.InstrumentationRule{tracesDisablingRule()},
			expectedReason:      "NoCollectedSignals",
			expectedMessage:     "all signals are disabled, no agent will be injected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			irls := tt.irls
			enabledSignals, disabledInfo := GetEnabledSignalsForContainer(tt.nodeCollectorsGroup, &irls)

			require.Equal(t, EnabledSignals{}, enabledSignals)
			require.NotNil(t, disabledInfo)
			require.Equal(t, tt.expectedReason, string(disabledInfo.AgentEnabledReason))
			require.Equal(t, tt.expectedMessage, disabledInfo.AgentEnabledMessage)
		})
	}
}

// the collectors group receiver signals are the source of truth for which signals
// are collected cluster wide, and each signal must be gated independently.
func TestGetEnabledSignalsForContainer_FollowsCollectorsGroupReceiverSignals(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		receiverSignals []common.ObservabilitySignal
		expected        EnabledSignals
	}{
		{
			name:            "traces only",
			receiverSignals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			expected:        EnabledSignals{TracesEnabled: true},
		},
		{
			name:            "metrics only",
			receiverSignals: []common.ObservabilitySignal{common.MetricsObservabilitySignal},
			expected:        EnabledSignals{MetricsEnabled: true},
		},
		{
			name:            "logs only",
			receiverSignals: []common.ObservabilitySignal{common.LogsObservabilitySignal},
			expected:        EnabledSignals{LogsEnabled: true},
		},
		{
			name: "all signals",
			receiverSignals: []common.ObservabilitySignal{
				common.TracesObservabilitySignal,
				common.MetricsObservabilitySignal,
				common.LogsObservabilitySignal,
			},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			irls := []odigosv1.InstrumentationRule{}
			enabledSignals, disabledInfo := GetEnabledSignalsForContainer(nodeCollectorsGroupWithSignals(tt.receiverSignals...), &irls)

			require.Nil(t, disabledInfo)
			require.Equal(t, tt.expected, enabledSignals)
		})
	}
}

// only workload level rules can turn traces off, and they must never affect metrics or logs.
func TestGetEnabledSignalsForContainer_TraceConfigRules(t *testing.T) {
	t.Parallel()

	notDisabled := false
	disabled := true
	allSignals := []common.ObservabilitySignal{
		common.TracesObservabilitySignal,
		common.MetricsObservabilitySignal,
		common.LogsObservabilitySignal,
	}

	tests := []struct {
		name     string
		irls     []odigosv1.InstrumentationRule
		expected EnabledSignals
	}{
		{
			name:     "no rules",
			irls:     []odigosv1.InstrumentationRule{},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name: "rule without a trace config",
			irls: []odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{}},
			},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name: "trace config without an explicit disabled value",
			irls: []odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{TraceConfig: &instrumentationrules.TraceConfig{}}},
			},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name: "trace config explicitly not disabled",
			irls: []odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{TraceConfig: &instrumentationrules.TraceConfig{Disabled: &notDisabled}}},
			},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name:     "trace config disabled turns traces off but keeps metrics and logs",
			irls:     []odigosv1.InstrumentationRule{tracesDisablingRule()},
			expected: EnabledSignals{MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name: "a library scoped rule is not a workload level rule and is ignored",
			irls: []odigosv1.InstrumentationRule{
				{
					Spec: odigosv1.InstrumentationRuleSpec{
						InstrumentationLibraries: &[]odigosv1.InstrumentationLibraryGlobalId{},
						TraceConfig:              &instrumentationrules.TraceConfig{Disabled: &disabled},
					},
				},
			},
			expected: EnabledSignals{TracesEnabled: true, MetricsEnabled: true, LogsEnabled: true},
		},
		{
			name: "a single disabling rule wins over the other rules",
			irls: []odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{TraceConfig: &instrumentationrules.TraceConfig{Disabled: &notDisabled}}},
				tracesDisablingRule(),
				{Spec: odigosv1.InstrumentationRuleSpec{TraceConfig: &instrumentationrules.TraceConfig{Disabled: &notDisabled}}},
			},
			expected: EnabledSignals{MetricsEnabled: true, LogsEnabled: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			irls := tt.irls
			enabledSignals, disabledInfo := GetEnabledSignalsForContainer(nodeCollectorsGroupWithSignals(allSignals...), &irls)

			require.Nil(t, disabledInfo)
			require.Equal(t, tt.expected, enabledSignals)
		})
	}
}
