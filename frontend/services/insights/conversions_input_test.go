package insights

import (
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The input converters are the only validation between a GraphQL mutation
// argument and a write to the insights engine, and they all report failures as
// ErrBadRequest so the resolver answers 400 instead of 500.

func TestPolicyFromInputRejectsMalformedArguments(t *testing.T) {
	tests := []struct {
		name  string
		input model.InsightsPolicyInput
	}{
		{
			name:  "id is not a number",
			input: model.InsightsPolicyInput{ID: strPtr("seven")},
		},
		{
			name:  "signal weights are not JSON",
			input: model.InsightsPolicyInput{SignalWeights: strPtr("{")},
		},
		{
			name:  "signal weights are not a map of numbers",
			input: model.InsightsPolicyInput{SignalWeights: strPtr(`{"D2_egress":"high"}`)},
		},
		{
			name:  "enricher lists are not JSON",
			input: model.InsightsPolicyInput{EnricherLists: strPtr(`["token"]`)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := PolicyFromInput(tt.input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBadRequest)
		})
	}
}

// A create has no id yet, and the policy key is scope + scopeKey.
func TestPolicyFromInputWithoutAnID(t *testing.T) {
	got, err := PolicyFromInput(model.InsightsPolicyInput{
		Name:     "default",
		Scope:    model.InsightsPolicyScope("global"),
		ScopeKey: "",
	})
	require.NoError(t, err)
	assert.Equal(t, Policy{Name: "default", Scope: "global"}, got)
}

func TestLearningPolicyFromInputMapsConditions(t *testing.T) {
	got := LearningPolicyFromInput(model.InsightsLearningPolicyInput{
		Class: model.InsightsDeviationClass("D3_latency"),
		Mode:  model.InsightsLearningMode("all"),
		Conditions: []*model.InsightsLearningConditionInput{
			nil,
			{
				Type:               model.InsightsLearningConditionType("stability"),
				MinObservations:    intPtr(11),
				MinDurationMinutes: intPtr(12),
				StableObservations: intPtr(13),
				StableMinutes:      intPtr(14),
			},
		},
		MinMatches: intPtr(15),
		Scope:      model.InsightsPolicyScope("service"),
		ScopeKey:   "prod/checkout",
	})

	minObservations, minDuration := int64(11), int64(12)
	stableObservations, stableMinutes := int64(13), int64(14)
	assert.Equal(t, LearningPolicy{
		Class: "D3_latency",
		Mode:  "all",
		Conditions: []LearningCondition{{
			Type:               "stability",
			MinObservations:    &minObservations,
			MinDurationMinutes: &minDuration,
			StableObservations: &stableObservations,
			StableMinutes:      &stableMinutes,
		}},
		MinMatches: intPtr(15),
		Scope:      "service",
		ScopeKey:   "prod/checkout",
	}, got)
}

func TestLearningPolicyFromInputWithoutConditions(t *testing.T) {
	got := LearningPolicyFromInput(model.InsightsLearningPolicyInput{Class: model.InsightsDeviationClass("D2_egress")})
	assert.Nil(t, got.Conditions)
}

func TestGuardrailFromInputMapsEveryField(t *testing.T) {
	got := GuardrailFromInput(model.InsightsGuardrailInput{
		Scope:    model.InsightsPolicyScope("service"),
		ScopeKey: "prod/checkout",
		Rules: []*model.InsightsGuardrailRuleInput{
			nil,
			{
				Key:       "allowed_egress",
				Label:     "Allowed egress",
				Mode:      model.InsightsRuleMode("enforce"),
				Allowlist: []string{"db:5432"},
				Origin:    strPtr("auto_transaction_guardrail"),
			},
			{
				Key:   "allowed_callers",
				Label: "Allowed callers",
				Mode:  model.InsightsRuleMode("off"),
			},
		},
	})

	assert.Equal(t, Guardrail{
		Scope:    "service",
		ScopeKey: "prod/checkout",
		Rules: []GuardrailRule{
			{
				Key:       "allowed_egress",
				Label:     "Allowed egress",
				Mode:      "enforce",
				Allowlist: []string{"db:5432"},
				Origin:    "auto_transaction_guardrail",
			},
			{
				Key:   "allowed_callers",
				Label: "Allowed callers",
				Mode:  "off",
			},
		},
	}, got)
}

func TestViolationActionFromInputMapsEveryField(t *testing.T) {
	got := ViolationActionFromInput(model.InsightsViolationActionInput{
		ScopeKey:  "prod/checkout",
		RuleKey:   "allowed_egress",
		Offending: "db:5432",
	})

	assert.Equal(t, ViolationActionRequest{
		ScopeKey:  "prod/checkout",
		RuleKey:   "allowed_egress",
		Offending: "db:5432",
	}, got)
}

func TestGuardrailSeedFromInputRejectsMalformedItems(t *testing.T) {
	_, err := GuardrailSeedFromInput(model.InsightsGuardrailSeedInput{
		ScopeKey: "prod/checkout",
		Items:    "not json",
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBadRequest)
}

func TestGuardrailSeedFromInputWithoutAMode(t *testing.T) {
	got, err := GuardrailSeedFromInput(model.InsightsGuardrailSeedInput{
		ScopeKey: "prod/checkout",
		Items:    `{"allowed_callers":["frontend"]}`,
	})
	require.NoError(t, err)
	assert.Nil(t, got.Mode)
	assert.Equal(t, map[string][]string{"allowed_callers": {"frontend"}}, got.Items)
}

func TestBulkAnomalyRequestFromInputRejectsMalformedItems(t *testing.T) {
	tests := []struct {
		name  string
		items []*model.InsightsAnomalyRefInput
	}{
		{
			name:  "nil item",
			items: []*model.InsightsAnomalyRefInput{nil},
		},
		{
			name:  "transaction id is not a number",
			items: []*model.InsightsAnomalyRefInput{{TransactionID: "abc", Signature: "sig-1"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BulkAnomalyRequestFromInput(model.InsightsBulkResolutionDismiss, tt.items)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBadRequest)
		})
	}
}

func TestBulkAnomalyRequestFromInputWithoutItems(t *testing.T) {
	got, err := BulkAnomalyRequestFromInput(model.InsightsBulkResolutionDismiss, nil)
	require.NoError(t, err)
	assert.Equal(t, BulkAnomalyRequest{Resolution: "dismiss"}, got)
}

func TestSystemSettingsFromInputRequiresEverySection(t *testing.T) {
	complete := func() model.InsightsSystemSettingsInput {
		return model.InsightsSystemSettingsInput{
			Sampling:  &model.InsightsSystemSamplingSettingsInput{},
			Retention: &model.InsightsSystemRetentionSettingsInput{},
			Capacity:  &model.InsightsSystemCapacitySettingsInput{},
			Writeback: &model.InsightsSystemWritebackSettingsInput{},
			Detection: &model.InsightsSystemDetectionSettingsInput{},
			Identity:  &model.InsightsSystemIdentitySettingsInput{},
		}
	}

	tests := []struct {
		name  string
		clear func(*model.InsightsSystemSettingsInput)
	}{
		{name: "sampling", clear: func(i *model.InsightsSystemSettingsInput) { i.Sampling = nil }},
		{name: "retention", clear: func(i *model.InsightsSystemSettingsInput) { i.Retention = nil }},
		{name: "capacity", clear: func(i *model.InsightsSystemSettingsInput) { i.Capacity = nil }},
		{name: "writeback", clear: func(i *model.InsightsSystemSettingsInput) { i.Writeback = nil }},
		{name: "detection", clear: func(i *model.InsightsSystemSettingsInput) { i.Detection = nil }},
		{name: "identity", clear: func(i *model.InsightsSystemSettingsInput) { i.Identity = nil }},
	}

	for _, tt := range tests {
		t.Run("missing "+tt.name, func(t *testing.T) {
			input := complete()
			tt.clear(&input)

			_, err := SystemSettingsFromInput(input)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrBadRequest)
		})
	}

	t.Run("complete", func(t *testing.T) {
		_, err := SystemSettingsFromInput(complete())
		require.NoError(t, err)
	})
}

func TestSystemSettingsFromInputCarriesResetTransactions(t *testing.T) {
	reset := true
	got, err := SystemSettingsFromInput(model.InsightsSystemSettingsInput{
		Sampling:  &model.InsightsSystemSamplingSettingsInput{},
		Retention: &model.InsightsSystemRetentionSettingsInput{},
		Capacity:  &model.InsightsSystemCapacitySettingsInput{},
		Writeback: &model.InsightsSystemWritebackSettingsInput{},
		Detection: &model.InsightsSystemDetectionSettingsInput{},
		Identity: &model.InsightsSystemIdentitySettingsInput{
			TransactionIdentityDimensions: []*model.InsightsSystemTransactionIdentityDimensionInput{
				nil,
				{Key: "tenant", Enabled: true},
			},
		},
		ResetTransactions: &reset,
	})
	require.NoError(t, err)

	assert.Equal(t, &reset, got.ResetTransactions)
	assert.Equal(t, []SystemTransactionIdentityDimension{{Key: "tenant", Enabled: true}}, got.Identity.TransactionIdentityDimensions)
}

func TestSystemIdentitySettingsFromInputWithoutASection(t *testing.T) {
	got := SystemIdentitySettingsFromInput(nil)
	assert.Equal(t, SystemIdentitySettings{TransactionIdentityDimensions: []SystemTransactionIdentityDimension{}}, got)
}
