package metrics

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/stretchr/testify/require"
)

// network metrics are presence based: any matching rule that sets networkMetrics turns
// them on for the container, and a container with no such rule must not collect them.
func TestCalculateNetworkMetricsConfig(t *testing.T) {
	t.Parallel()

	networkMetrics := &instrumentationrules.NetworkMetricsConfig{}

	tests := []struct {
		name     string
		irls     *[]odigosv1.InstrumentationRule
		expected *instrumentationrules.NetworkMetricsConfig
	}{
		{
			name:     "nil rules",
			irls:     nil,
			expected: nil,
		},
		{
			name:     "no rules",
			irls:     &[]odigosv1.InstrumentationRule{},
			expected: nil,
		},
		{
			name: "rules without network metrics",
			irls: &[]odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{}},
				{Spec: odigosv1.InstrumentationRuleSpec{}},
			},
			expected: nil,
		},
		{
			name: "a single rule sets network metrics",
			irls: &[]odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{NetworkMetrics: networkMetrics}},
			},
			expected: networkMetrics,
		},
		{
			name: "a rule without network metrics does not clear an earlier one",
			irls: &[]odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{NetworkMetrics: networkMetrics}},
				{Spec: odigosv1.InstrumentationRuleSpec{}},
			},
			expected: networkMetrics,
		},
		{
			name: "a later rule sets network metrics",
			irls: &[]odigosv1.InstrumentationRule{
				{Spec: odigosv1.InstrumentationRuleSpec{}},
				{Spec: odigosv1.InstrumentationRuleSpec{NetworkMetrics: networkMetrics}},
			},
			expected: networkMetrics,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.expected, CalculateNetworkMetricsConfig(tt.irls))
		})
	}
}
