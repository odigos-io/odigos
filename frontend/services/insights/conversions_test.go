package insights

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLearningConditionUsesCamelCaseOnTheWire(t *testing.T) {
	minObservations := int64(100)
	stableMinutes := int64(1440)
	condition := LearningCondition{
		Type:            "min_observations",
		MinObservations: &minObservations,
		StableMinutes:   &stableMinutes,
	}

	encoded, err := json.Marshal(condition)
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"type": "min_observations",
		"minObservations": 100,
		"stableMinutes": 1440
	}`, string(encoded))
	assert.NotContains(t, string(encoded), `"stable_minutes"`)
	assert.NotContains(t, string(encoded), `"min_observations":`)
	assert.Contains(t, string(encoded), `"minObservations":`)

	var decoded LearningCondition
	require.NoError(t, json.Unmarshal([]byte(`{
		"type": "stable_duration",
		"stableMinutes": 60,
		"minObservations": 10
	}`), &decoded))
	require.NotNil(t, decoded.StableMinutes)
	require.NotNil(t, decoded.MinObservations)
	assert.Equal(t, LearningConditionType("stable_duration"), decoded.Type)
	assert.Equal(t, int64(60), *decoded.StableMinutes)
	assert.Equal(t, int64(10), *decoded.MinObservations)
}

func TestLearningPolicyRoundTripPreservesCamelCaseConditions(t *testing.T) {
	minObservations := int64(50)
	minMatches := 1
	rest := LearningPolicy{
		Class: "D2_egress",
		Mode:  "any",
		Conditions: []LearningCondition{{
			Type:            "min_observations",
			MinObservations: &minObservations,
		}},
		MinMatches: &minMatches,
		Scope:      "global",
		ScopeKey:   "",
	}

	encoded, err := json.Marshal(rest)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"min_matches"`)
	assert.Contains(t, string(encoded), `"scope_key"`)
	assert.Contains(t, string(encoded), `"minObservations"`)

	gql := LearningPolicyToModel(rest)
	require.Len(t, gql.Conditions, 1)
	require.NotNil(t, gql.Conditions[0].MinObservations)
	assert.Equal(t, 50, *gql.Conditions[0].MinObservations)

	back := LearningPolicyFromInput(model.InsightsLearningPolicyInput{
		Class: gql.Class,
		Mode:  gql.Mode,
		Conditions: []*model.InsightsLearningConditionInput{{
			Type:            gql.Conditions[0].Type,
			MinObservations: gql.Conditions[0].MinObservations,
		}},
		MinMatches: gql.MinMatches,
		Scope:      gql.Scope,
		ScopeKey:   gql.ScopeKey,
	})
	require.Len(t, back.Conditions, 1)
	require.NotNil(t, back.Conditions[0].MinObservations)
	assert.Equal(t, int64(50), *back.Conditions[0].MinObservations)
	assert.Equal(t, rest.Class, back.Class)
	assert.Equal(t, rest.Mode, back.Mode)
}

func TestTransactionIDRoundTrip(t *testing.T) {
	assert.Equal(t, "4211", FormatID(4211))
	id, err := ParseID("4211")
	require.NoError(t, err)
	assert.Equal(t, int64(4211), id)

	_, err = ParseID("not-a-number")
	require.Error(t, err)

	stat := TransactionStat{
		ID:            4211,
		Service:       "checkout",
		Namespace:     "prod",
		Operation:     "POST /pay [http.response.status_code=200]",
		OperationName: "POST /pay",
		IdentityDimensions: []TransactionIdentityValue{
			{Key: "http.response.status_code", Value: "200"},
		},
		Kind:     "HTTP",
		Volume:   12,
		LastSeen: "2026-08-05T06:00:00Z",
	}
	converted := TransactionStatToModel(stat)
	assert.Equal(t, "4211", converted.ID)
	assert.Equal(t, model.InsightsTransactionKindHTTP, converted.Kind)
	assert.Equal(t, "POST /pay", converted.OperationName)
	require.Len(t, converted.IdentityDimensions, 1)
	assert.Equal(t, "http.response.status_code", converted.IdentityDimensions[0].Key)
	assert.Equal(t, "200", converted.IdentityDimensions[0].Value)
}

func TestPolicyRoundTripEncodesMapsAsJSONStrings(t *testing.T) {
	rest := Policy{
		ID:          7,
		Name:        "default",
		Enabled:     true,
		FireAtScore: 60,
		SignalWeights: map[string]int{
			"D7_db_access": 30,
		},
		EnricherLists: map[string][]string{
			"sensitive_tables": {"users", "secrets"},
		},
		Scope:    "global",
		ScopeKey: "",
	}

	gql, err := PolicyToModel(rest)
	require.NoError(t, err)
	assert.Equal(t, "7", gql.ID)
	require.NotNil(t, gql.SignalWeights)
	require.NotNil(t, gql.EnricherLists)
	assert.JSONEq(t, `{"D7_db_access":30}`, *gql.SignalWeights)
	assert.JSONEq(t, `{"sensitive_tables":["users","secrets"]}`, *gql.EnricherLists)

	back, err := PolicyFromInput(model.InsightsPolicyInput{
		ID:            &gql.ID,
		Name:          gql.Name,
		Enabled:       gql.Enabled,
		FireAtScore:   gql.FireAtScore,
		SignalWeights: gql.SignalWeights,
		EnricherLists: gql.EnricherLists,
		Scope:         gql.Scope,
		ScopeKey:      gql.ScopeKey,
	})
	require.NoError(t, err)
	assert.Equal(t, rest, back)
}

func TestBaselineAndAnomalyEvidenceStayJSONStrings(t *testing.T) {
	baseline, err := BaselineClassToModel(BaselineClass{
		TransactionID:    9,
		Class:            "D3_latency",
		ClassLabel:       "Request latency",
		ClassDescription: "Learns typical request latency for this transaction.",
		Data:             json.RawMessage(`{"p99_ms":120}`),
		ObservationCount: 3,
		Promoted:         true,
		Learning: BaselineLearning{
			Phase:                       BaselineLearningPhasePromoted,
			ObservationsSinceLastChange: 2,
			QuietMinutes:                60,
			Mode:                        "any",
			ReadyToPromote:              false,
			Stability: BaselineLearningStability{
				Observations: BaselineStabilityProgress{Current: 2, Target: 50, DrivesPromotion: true, Met: false},
				Duration:     BaselineStabilityProgress{Current: 60, Target: 0, DrivesPromotion: false, Met: false},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, baseline.Data)
	assert.JSONEq(t, `{"p99_ms":120}`, *baseline.Data)
	assert.Equal(t, "9", baseline.TransactionID)
	assert.Equal(t, "Request latency", baseline.ClassLabel)
	assert.Equal(t, "Learns typical request latency for this transaction.", baseline.ClassDescription)
	require.NotNil(t, baseline.Learning)
	assert.Equal(t, model.InsightsBaselineLearningPhasePromoted, baseline.Learning.Phase)
	assert.Equal(t, 2, baseline.Learning.ObservationsSinceLastChange)
	assert.Equal(t, 60, baseline.Learning.QuietMinutes)
	assert.Equal(t, model.InsightsLearningModeAny, baseline.Learning.Mode)
	require.NotNil(t, baseline.Learning.Stability)
	require.NotNil(t, baseline.Learning.Stability.Observations)
	assert.Equal(t, 50, baseline.Learning.Stability.Observations.Target)

	histogramBaseline, err := BaselineClassToModel(BaselineClass{
		TransactionID:    9,
		Class:            "D3_latency",
		ClassLabel:       "Request latency",
		ClassDescription: "Learns typical request latency for this transaction.",
		Data:             json.RawMessage(`{"latency":{"buckets":[0,0,0,0,0,0,0,140,12,3,0]}}`),
		ObservationCount: 155,
		Promoted:         false,
		Learning: BaselineLearning{
			Phase: BaselineLearningPhaseLearning,
			Stability: BaselineLearningStability{
				Observations: BaselineStabilityProgress{Current: 10, Target: 50, DrivesPromotion: true, Met: false},
				Duration:     BaselineStabilityProgress{Current: 30, Target: 60, DrivesPromotion: true, Met: false},
			},
		},
		Histogram: &BaselineHistogram{
			Unit:              BaselineHistogramUnitUs,
			LayoutFingerprint: "8f3e2a1b4c5d6e7f",
			Series: []BaselineHistogramSeries{
				{
					Name:  "latency",
					Label: "Entry latency",
					Bars: []BaselineHistogramBar{
						{Lo: 0, Hi: 0, Count: 0, Label: "0µs"},
						{Lo: 64, Hi: 128, Count: 140, Label: "64–127µs"},
						{Lo: 128, Hi: 256, Count: 12, Label: "128–255µs"},
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, histogramBaseline.Histogram)
	assert.Equal(t, model.InsightsBaselineHistogramUnitUs, histogramBaseline.Histogram.Unit)
	require.Len(t, histogramBaseline.Histogram.Series, 1)
	assert.Equal(t, "latency", histogramBaseline.Histogram.Series[0].Name)
	require.Len(t, histogramBaseline.Histogram.Series[0].Bars, 3)
	assert.Equal(t, 140, histogramBaseline.Histogram.Series[0].Bars[1].Count)

	anomaly, err := AnomalyIssueToModel(AnomalyIssue{
		TransactionID:    9,
		Signature:        "sig",
		Service:          "checkout",
		Namespace:        "prod",
		TriggeredClasses: []DeviationClass{"D2_egress"},
		Evidence:         json.RawMessage(`{"D2_egress":{"new_peers":["1.2.3.4:443"]}}`),
		Occurrences:      2,
		MaxScore:         40,
		Severity:         "medium",
		Status:           "open",
		ClassFindings: []AnomalyClassFinding{
			{
				Class:   "D2_egress",
				Kind:    "structural",
				Title:   "Unexpected egress",
				Summary: "New peer 1.2.3.4:443 not in the learned baseline.",
				SpanHighlights: []AnomalySpanHighlight{
					{SpanID: "aabb", Severity: "high", Reason: "new egress peer"},
				},
				RawEvidence: json.RawMessage(`{"new_peers":["1.2.3.4:443"]}`),
			},
		},
		AnomalyTrace: &Observation{
			TraceID:       "trace-anomaly",
			TransactionID: 9,
			ObservedAt:    "2026-07-20T14:03:22Z",
			DurationMs:    12,
			SampleReason:  "anomaly:sig",
			RawOTLP:       "YmFzZTY0",
		},
		BaselineTrace: &Observation{
			TraceID:       "trace-baseline",
			TransactionID: 9,
			ObservedAt:    "2026-07-20T09:15:01Z",
			DurationMs:    4,
			SampleReason:  "example",
			RawOTLP:       "YmFzZTY0",
		},
	})
	require.NoError(t, err)
	assert.JSONEq(t, `{"D2_egress":{"new_peers":["1.2.3.4:443"]}}`, anomaly.Evidence)
	assert.Equal(t, "9", anomaly.TransactionID)
	require.Len(t, anomaly.ClassFindings, 1)
	assert.Equal(t, model.InsightsAnomalyClassFindingKindStructural, anomaly.ClassFindings[0].Kind)
	assert.Equal(t, "Unexpected egress", anomaly.ClassFindings[0].Title)
	require.NotNil(t, anomaly.ClassFindings[0].RawEvidence)
	assert.JSONEq(t, `{"new_peers":["1.2.3.4:443"]}`, *anomaly.ClassFindings[0].RawEvidence)
	require.NotNil(t, anomaly.AnomalyTrace)
	assert.Equal(t, "trace-anomaly", anomaly.AnomalyTrace.TraceID)
	assert.Equal(t, model.InsightsSampleReasonAnomalyEvidence, anomaly.AnomalyTrace.SampleReason)
	require.NotNil(t, anomaly.BaselineTrace)
	assert.Equal(t, model.InsightsSampleReasonExample, anomaly.BaselineTrace.SampleReason)
}

func TestSystemSettingsAndGuardrailSeedRoundTrip(t *testing.T) {
	settings := SystemSettings{
		Sampling:  SystemSamplingSettings{ExamplesPerTransaction: 3, ExampleSampleIntervalSeconds: 60},
		Retention: SystemRetentionSettings{ObservationRetentionDays: 14},
		Capacity:  SystemCapacitySettings{MaxResidentTransactions: 1000, MaxBaselineSetMembers: 500},
		Writeback: SystemWritebackSettings{FlushIntervalSeconds: 30},
		Detection: SystemDetectionSettings{AutoTransactionGuardrail: true},
		Identity: SystemIdentitySettings{
			TransactionIdentityDimensions: []SystemTransactionIdentityDimension{
				{Key: "http.response.status_code", Enabled: true},
				{Key: "rpc.grpc.status_code", Enabled: true},
			},
		},
	}
	gql := SystemSettingsToModel(settings)
	back, err := SystemSettingsFromInput(model.InsightsSystemSettingsInput{
		Sampling:  &model.InsightsSystemSamplingSettingsInput{ExamplesPerTransaction: gql.Sampling.ExamplesPerTransaction, ExampleSampleIntervalSeconds: gql.Sampling.ExampleSampleIntervalSeconds},
		Retention: &model.InsightsSystemRetentionSettingsInput{ObservationRetentionDays: gql.Retention.ObservationRetentionDays},
		Capacity:  &model.InsightsSystemCapacitySettingsInput{MaxResidentTransactions: gql.Capacity.MaxResidentTransactions, MaxBaselineSetMembers: gql.Capacity.MaxBaselineSetMembers},
		Writeback: &model.InsightsSystemWritebackSettingsInput{FlushIntervalSeconds: gql.Writeback.FlushIntervalSeconds},
		Detection: &model.InsightsSystemDetectionSettingsInput{AutoTransactionGuardrail: gql.Detection.AutoTransactionGuardrail},
		Identity: &model.InsightsSystemIdentitySettingsInput{
			TransactionIdentityDimensions: []*model.InsightsSystemTransactionIdentityDimensionInput{
				{Key: "http.response.status_code", Enabled: true},
				{Key: "rpc.grpc.status_code", Enabled: true},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, settings, back)

	items := `{"allowed_egress":["db:5432"]}`
	seed, err := GuardrailSeedFromInput(model.InsightsGuardrailSeedInput{
		ScopeKey: "prod/checkout",
		Items:    items,
	})
	require.NoError(t, err)
	assert.Equal(t, "prod/checkout", seed.ScopeKey)
	assert.Equal(t, []string{"db:5432"}, seed.Items["allowed_egress"])
}

func TestBulkAnomalyRequestConvertsTransactionIDs(t *testing.T) {
	request, err := BulkAnomalyRequestFromInput(model.InsightsBulkResolutionDismiss, []*model.InsightsAnomalyRefInput{
		{TransactionID: "42", Signature: "abc"},
		{TransactionID: "7", Signature: "def"},
	})
	require.NoError(t, err)
	assert.Equal(t, BulkResolution("dismiss"), request.Resolution)
	require.Len(t, request.Items, 2)
	assert.Equal(t, int64(42), request.Items[0].TransactionID)
	assert.Equal(t, "abc", request.Items[0].Signature)
}

func TestLearningConditionRejectsAMalformedWireValue(t *testing.T) {
	var condition LearningCondition
	require.Error(t, condition.UnmarshalJSON([]byte(`{"minObservations":"many"}`)))
}
