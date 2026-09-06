package insights

import (
	"encoding/json"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The converters in conversions.go copy the REST payloads field by field into
// the generated GraphQL models. A dropped or swapped field is silent: the UI
// renders a plausible but wrong value and nothing errors. Every fixture below
// therefore gives each sibling field a distinct value and asserts the whole
// converted struct at once.

func TestServiceConvertersMapEveryField(t *testing.T) {
	t.Run("service stats", func(t *testing.T) {
		got := ServiceStatsToModel([]ServiceStat{{
			Namespace:        "prod",
			Service:          "checkout",
			TransactionCount: 11,
			Volume:           22,
			LastSeen:         "2026-09-01T10:00:00Z",
		}})

		assert.Equal(t, []*model.InsightsServiceStat{{
			Namespace:        "prod",
			Service:          "checkout",
			TransactionCount: 11,
			Volume:           22,
			LastSeen:         "2026-09-01T10:00:00Z",
		}}, got)
	})

	t.Run("service profile", func(t *testing.T) {
		got := ServiceProfileToModel(ServiceProfile{
			Namespace:    "prod",
			Service:      "checkout",
			Callers:      []string{"frontend"},
			Callees:      []string{"payments"},
			Egress:       []string{"db:5432"},
			Transactions: []string{"HTTP\tGET /cart"},
		})

		assert.Equal(t, &model.InsightsServiceProfile{
			Namespace:    "prod",
			Service:      "checkout",
			Callers:      []string{"frontend"},
			Callees:      []string{"payments"},
			Egress:       []string{"db:5432"},
			Transactions: []string{"HTTP\tGET /cart"},
		}, got)
	})

	// The GraphQL schema declares these lists non-nullable, so a missing REST
	// list has to become an empty list rather than null.
	t.Run("service profile lists are never null", func(t *testing.T) {
		got := ServiceProfileToModel(ServiceProfile{Service: "checkout"})

		assert.Equal(t, &model.InsightsServiceProfile{
			Service:      "checkout",
			Callers:      []string{},
			Callees:      []string{},
			Egress:       []string{},
			Transactions: []string{},
		}, got)
	})

	t.Run("blast radius subgraph", func(t *testing.T) {
		got := BlastRadiusSubgraphToModel(BlastRadiusSubgraph{
			Root: BlastRadiusNode{Namespace: "root-ns", Service: "root-svc"},
			Nodes: []BlastRadiusNode{
				{Namespace: "node-ns", Service: "node-svc", IsVirtual: true},
			},
			Edges: []BlastRadiusEdge{{
				ClientNamespace: "client-ns",
				ClientService:   "client-svc",
				ServerNamespace: "server-ns",
				ServerService:   "server-svc",
				ConnectionType:  "http",
				RequestCount:    31,
				FailedCount:     32,
				LastSeen:        "2026-09-01T11:00:00Z",
			}},
		})

		assert.Equal(t, &model.InsightsBlastRadiusSubgraph{
			Root: &model.InsightsBlastRadiusNode{Namespace: "root-ns", Service: "root-svc"},
			Nodes: []*model.InsightsBlastRadiusNode{
				{Namespace: "node-ns", Service: "node-svc", IsVirtual: true},
			},
			Edges: []*model.InsightsBlastRadiusEdge{{
				ClientNamespace: "client-ns",
				ClientService:   "client-svc",
				ServerNamespace: "server-ns",
				ServerService:   "server-svc",
				ConnectionType:  "http",
				RequestCount:    31,
				FailedCount:     32,
				LastSeen:        "2026-09-01T11:00:00Z",
			}},
		}, got)
	})
}

func TestTransactionConvertersMapEveryField(t *testing.T) {
	hasBaseline := true
	promoted := false
	identity := []TransactionIdentityValue{{Key: "tenant", Value: "acme"}}
	modelIdentity := []*model.InsightsTransactionIdentityValue{{Key: "tenant", Value: "acme"}}

	t.Run("transaction stat", func(t *testing.T) {
		got := TransactionStatsToModel([]TransactionStat{{
			ID:                 41,
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "GET /cart[tenant=acme]",
			OperationName:      "GET /cart",
			IdentityDimensions: identity,
			Kind:               "HTTP",
			Volume:             42,
			LastSeen:           "2026-09-01T12:00:00Z",
			HasBaseline:        &hasBaseline,
			Promoted:           &promoted,
		}})

		assert.Equal(t, []*model.InsightsTransactionStat{{
			ID:                 "41",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "GET /cart[tenant=acme]",
			OperationName:      "GET /cart",
			IdentityDimensions: modelIdentity,
			Kind:               model.InsightsTransactionKindHTTP,
			Volume:             42,
			LastSeen:           "2026-09-01T12:00:00Z",
			HasBaseline:        &hasBaseline,
			Promoted:           &promoted,
		}}, got)
	})

	t.Run("transaction", func(t *testing.T) {
		got := TransactionToModel(Transaction{
			ID:                 43,
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "GET /cart[tenant=acme]",
			OperationName:      "GET /cart",
			IdentityDimensions: identity,
			Kind:               "GRPC",
		})

		assert.Equal(t, &model.InsightsTransaction{
			ID:                 "43",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "GET /cart[tenant=acme]",
			OperationName:      "GET /cart",
			IdentityDimensions: modelIdentity,
			Kind:               model.InsightsTransactionKindGrpc,
		}, got)
	})

	t.Run("identity dimensions are never null", func(t *testing.T) {
		got := TransactionToModel(Transaction{ID: 44})
		assert.Equal(t, []*model.InsightsTransactionIdentityValue{}, got.IdentityDimensions)
	})
}

func TestBaselineConvertersMapEveryField(t *testing.T) {
	schemaVersion := 2
	countAtLastChange := int64(51)
	learningStartedAt := "2026-09-01T00:00:00Z"
	lastChangedAt := "2026-09-01T01:00:00Z"

	got, err := BaselineClassesToModel([]BaselineClass{{
		TransactionID:    52,
		Class:            "D3_latency",
		ClassLabel:       "Request latency",
		ClassDescription: "How long the transaction usually takes",
		Data:             json.RawMessage(`{"p95":120}`),
		Histogram: &BaselineHistogram{
			Unit:              BaselineHistogramUnitUs,
			LayoutFingerprint: "fp-1",
			Series: []BaselineHistogramSeries{{
				Name:  "baseline",
				Label: "Baseline",
				Bars:  []BaselineHistogramBar{{Lo: 53, Hi: 54, Count: 55, Label: "1-2ms"}},
			}},
		},
		DataSchemaVersion:            &schemaVersion,
		ObservationCount:             56,
		Promoted:                     true,
		LearningStartedAt:            &learningStartedAt,
		LastChangedAt:                &lastChangedAt,
		ObservationCountAtLastChange: &countAtLastChange,
		Learning: BaselineLearning{
			Phase:                       BaselineLearningPhaseLearning,
			ObservationsSinceLastChange: 57,
			QuietMinutes:                58,
			Mode:                        "auto",
			ReadyToPromote:              true,
			Stability: BaselineLearningStability{
				Observations: BaselineStabilityProgress{Current: 59, Target: 60, DrivesPromotion: true, Met: false},
				Duration:     BaselineStabilityProgress{Current: 61, Target: 62, DrivesPromotion: false, Met: true},
			},
		},
	}})
	require.NoError(t, err)

	assert.Equal(t, []*model.InsightsBaselineClass{{
		TransactionID:    "52",
		Class:            "D3_latency",
		ClassLabel:       "Request latency",
		ClassDescription: "How long the transaction usually takes",
		Data:             strPtr(`{"p95":120}`),
		Histogram: &model.InsightsBaselineHistogram{
			Unit:              model.InsightsBaselineHistogramUnit("us"),
			LayoutFingerprint: strPtr("fp-1"),
			Series: []*model.InsightsBaselineHistogramSeries{{
				Name:  "baseline",
				Label: "Baseline",
				Bars:  []*model.InsightsBaselineHistogramBar{{Lo: 53, Hi: 54, Count: 55, Label: "1-2ms"}},
			}},
		},
		DataSchemaVersion:            &schemaVersion,
		ObservationCount:             56,
		Promoted:                     true,
		LearningStartedAt:            &learningStartedAt,
		LastChangedAt:                &lastChangedAt,
		ObservationCountAtLastChange: intPtr(51),
		Learning: &model.InsightsBaselineLearning{
			Phase:                       model.InsightsBaselineLearningPhase("learning"),
			ObservationsSinceLastChange: 57,
			QuietMinutes:                58,
			Mode:                        model.InsightsLearningMode("auto"),
			ReadyToPromote:              true,
			Stability: &model.InsightsBaselineLearningStability{
				Observations: &model.InsightsBaselineStabilityProgress{Current: 59, Target: 60, DrivesPromotion: true},
				Duration:     &model.InsightsBaselineStabilityProgress{Current: 61, Target: 62, Met: true},
			},
		},
	}}, got)
}

func TestBaselineOptionalFieldsAreOmittedWhenAbsent(t *testing.T) {
	got, err := BaselineClassToModel(BaselineClass{
		TransactionID: 63,
		Class:         "D2_egress",
	})
	require.NoError(t, err)

	assert.Nil(t, got.Data, "an absent data blob must not become the string \"null\"")
	assert.Nil(t, got.Histogram)

	histogram, err := BaselineClassToModel(BaselineClass{
		Histogram: &BaselineHistogram{Unit: BaselineHistogramUnitBytes},
	})
	require.NoError(t, err)
	require.NotNil(t, histogram.Histogram)
	assert.Nil(t, histogram.Histogram.LayoutFingerprint, "an empty layout fingerprint must stay unset")
	assert.Empty(t, histogram.Histogram.Series)
}

func TestObservationConvertersMapEveryField(t *testing.T) {
	t.Run("summary", func(t *testing.T) {
		got := ObservationSummariesToModel([]ObservationSummary{{
			TraceID:       "trace-1",
			TransactionID: 71,
			ObservedAt:    "2026-09-01T13:00:00Z",
			DurationMs:    72,
			SampleReason:  SampleReasonExample,
		}})

		assert.Equal(t, []*model.InsightsObservationSummary{{
			TraceID:       "trace-1",
			TransactionID: "71",
			ObservedAt:    "2026-09-01T13:00:00Z",
			DurationMs:    72,
			SampleReason:  model.InsightsSampleReasonExample,
		}}, got)
	})

	t.Run("full observation", func(t *testing.T) {
		got := ObservationToModel(Observation{
			TraceID:       "trace-2",
			TransactionID: 73,
			ObservedAt:    "2026-09-01T14:00:00Z",
			DurationMs:    74,
			SampleReason:  AnomalyEvidenceSampleReason("sig-1"),
			RawOTLP:       "YWJj",
		})

		assert.Equal(t, &model.InsightsObservation{
			TraceID:       "trace-2",
			TransactionID: "73",
			ObservedAt:    "2026-09-01T14:00:00Z",
			DurationMs:    74,
			SampleReason:  model.InsightsSampleReasonAnomalyEvidence,
			RawOtlp:       "YWJj",
		}, got)
	})
}

func TestPolicyConvertersMapEveryField(t *testing.T) {
	t.Run("policy", func(t *testing.T) {
		got, err := PoliciesToModel([]Policy{{
			ID:            81,
			Name:          "strict",
			Enabled:       true,
			FireAtScore:   82,
			SignalWeights: map[string]int{"D2_egress": 83},
			EnricherLists: map[string][]string{"secrets": {"token"}},
			Scope:         "service",
			ScopeKey:      "prod/checkout",
		}})
		require.NoError(t, err)

		assert.Equal(t, []*model.InsightsPolicy{{
			ID:            "81",
			Name:          "strict",
			Enabled:       true,
			FireAtScore:   82,
			SignalWeights: strPtr(`{"D2_egress":83}`),
			EnricherLists: strPtr(`{"secrets":["token"]}`),
			Scope:         model.InsightsPolicyScope("service"),
			ScopeKey:      "prod/checkout",
		}}, got)
	})

	// Absent maps must stay absent: encoding them as "null" would make the UI
	// show a policy that overrides every weight with nothing.
	t.Run("policy without maps", func(t *testing.T) {
		got, err := PolicyToModel(Policy{ID: 84, Scope: "global"})
		require.NoError(t, err)
		assert.Nil(t, got.SignalWeights)
		assert.Nil(t, got.EnricherLists)
	})

	t.Run("learning policy", func(t *testing.T) {
		minMatches := 85
		minObservations := int64(86)
		got := LearningPoliciesToModel([]LearningPolicy{{
			Class: "D3_latency",
			Mode:  "manual",
			Conditions: []LearningCondition{{
				Type:            "observations",
				MinObservations: &minObservations,
			}},
			MinMatches: &minMatches,
			Scope:      "service",
			ScopeKey:   "prod/checkout",
		}})

		assert.Equal(t, []*model.InsightsLearningPolicy{{
			Class: "D3_latency",
			Mode:  model.InsightsLearningMode("manual"),
			Conditions: []*model.InsightsLearningCondition{{
				Type:            model.InsightsLearningConditionType("observations"),
				MinObservations: intPtr(86),
			}},
			MinMatches: &minMatches,
			Scope:      model.InsightsPolicyScope("service"),
			ScopeKey:   "prod/checkout",
		}}, got)
	})
}

func TestFindingConverterMapsEveryField(t *testing.T) {
	transactionID := int64(91)

	got := FindingsToModel([]Finding{{
		Kind:               "anomaly",
		Service:            "checkout",
		Namespace:          "prod",
		Title:              "Unexpected egress",
		Operation:          strPtr("GET /cart[tenant=acme]"),
		OperationName:      strPtr("GET /cart"),
		IdentityDimensions: []TransactionIdentityValue{{Key: "tenant", Value: "acme"}},
		Offending:          strPtr("db:5432"),
		Score:              92,
		Severity:           "high",
		Occurrences:        93,
		FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
		LastSeen:           "2026-09-01T15:00:00Z",
		Status:             "open",
		TransactionID:      &transactionID,
		Signature:          strPtr("sig-1"),
		TriggeredClasses:   []DeviationClass{"D2_egress"},
		ScopeKey:           strPtr("prod/checkout"),
		RuleKey:            strPtr("allowed_egress"),
	}})

	assert.Equal(t, []*model.InsightsFinding{{
		Kind:               model.InsightsFindingKind("anomaly"),
		Service:            "checkout",
		Namespace:          "prod",
		Title:              "Unexpected egress",
		Operation:          strPtr("GET /cart[tenant=acme]"),
		OperationName:      strPtr("GET /cart"),
		IdentityDimensions: []*model.InsightsTransactionIdentityValue{{Key: "tenant", Value: "acme"}},
		Offending:          strPtr("db:5432"),
		Score:              92,
		Severity:           model.InsightsSeverityHigh,
		Occurrences:        93,
		FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
		LastSeen:           "2026-09-01T15:00:00Z",
		Status:             "open",
		TransactionID:      strPtr("91"),
		Signature:          strPtr("sig-1"),
		TriggeredClasses:   []model.InsightsDeviationClass{"D2_egress"},
		ScopeKey:           strPtr("prod/checkout"),
		RuleKey:            strPtr("allowed_egress"),
	}}, got)
}

func TestAnomalyConvertersMapEveryField(t *testing.T) {
	kind := TransactionKind("CONSUMER")
	modelKind := model.InsightsTransactionKindConsumer
	policyID := int64(101)
	identity := []TransactionIdentityValue{{Key: "tenant", Value: "acme"}}
	modelIdentity := []*model.InsightsTransactionIdentityValue{{Key: "tenant", Value: "acme"}}

	t.Run("summary", func(t *testing.T) {
		got := AnomalySummariesToModel([]AnomalySummary{{
			TransactionID:      102,
			Signature:          "sig-1",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "consume orders[tenant=acme]",
			OperationName:      "consume orders",
			IdentityDimensions: identity,
			Kind:               &kind,
			TriggeredClasses:   []DeviationClass{"D2_egress"},
			Offending:          strPtr("db:5432"),
			PolicyID:           &policyID,
			Occurrences:        103,
			MaxScore:           104,
			Severity:           "critical",
			FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
			LastSeen:           strPtr("2026-09-01T16:00:00Z"),
			LastTraceID:        strPtr("trace-9"),
			Status:             "open",
		}})

		assert.Equal(t, []*model.InsightsAnomalySummary{{
			TransactionID:      "102",
			Signature:          "sig-1",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "consume orders[tenant=acme]",
			OperationName:      "consume orders",
			IdentityDimensions: modelIdentity,
			Kind:               &modelKind,
			TriggeredClasses:   []model.InsightsDeviationClass{"D2_egress"},
			Offending:          strPtr("db:5432"),
			PolicyID:           strPtr("101"),
			Occurrences:        103,
			MaxScore:           104,
			Severity:           model.InsightsSeverityCritical,
			FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
			LastSeen:           strPtr("2026-09-01T16:00:00Z"),
			LastTraceID:        strPtr("trace-9"),
			Status:             model.InsightsAnomalyStatus("open"),
		}}, got)
	})

	t.Run("issue detail", func(t *testing.T) {
		got, err := AnomalyIssueToModel(AnomalyIssue{
			TransactionID:      105,
			Signature:          "sig-1",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "consume orders[tenant=acme]",
			OperationName:      "consume orders",
			IdentityDimensions: identity,
			Kind:               &kind,
			TriggeredClasses:   []DeviationClass{"D2_egress"},
			Offending:          strPtr("db:5432"),
			Evidence:           json.RawMessage(`{"D2_egress":{"seen":1}}`),
			PolicyID:           &policyID,
			Occurrences:        106,
			MaxScore:           107,
			Severity:           "medium",
			FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
			LastSeen:           strPtr("2026-09-01T17:00:00Z"),
			LastTraceID:        strPtr("trace-9"),
			Status:             "accepted",
			Risk: &RiskAssessment{
				Score:            108,
				Severity:         "low",
				Correlated:       true,
				CorrelationBonus: 109,
				FireAtScore:      110,
				Categories:       []string{"exfiltration"},
				Signals: []RiskSignal{{
					Source:       "D2_egress",
					Category:     "network",
					Weight:       111,
					Confidence:   0.5,
					Contribution: 112,
					Rationale:    strPtr("new destination"),
					Mitre:        strPtr("T1041"),
					Owasp:        strPtr("A01"),
				}},
			},
			ClassFindings: []AnomalyClassFinding{{
				Class:          "D2_egress",
				Kind:           "new_value",
				Title:          "New egress destination",
				Summary:        "checkout called db:5432 for the first time",
				SpanHighlights: []AnomalySpanHighlight{{SpanID: "span-1", Severity: "high", Reason: "unknown peer"}},
				AttrHighlights: []AnomalyAttrHighlight{{SpanID: "span-2", Key: "net.peer.name", Value: "db"}},
				Metric:         &AnomalyMetricComparison{Observed: 113, Baseline: 114, Unit: "us"},
				RawEvidence:    json.RawMessage(`{"peer":"db"}`),
			}},
			AnomalyTrace:  &Observation{TraceID: "trace-anomaly"},
			BaselineTrace: &Observation{TraceID: "trace-baseline"},
		})
		require.NoError(t, err)

		assert.Equal(t, &model.InsightsAnomalyIssue{
			TransactionID:      "105",
			Signature:          "sig-1",
			Service:            "checkout",
			Namespace:          "prod",
			Operation:          "consume orders[tenant=acme]",
			OperationName:      "consume orders",
			IdentityDimensions: modelIdentity,
			Kind:               &modelKind,
			TriggeredClasses:   []model.InsightsDeviationClass{"D2_egress"},
			Offending:          strPtr("db:5432"),
			PolicyID:           strPtr("101"),
			Occurrences:        106,
			MaxScore:           107,
			Severity:           model.InsightsSeverityMedium,
			FirstSeen:          strPtr("2026-09-01T00:00:00Z"),
			LastSeen:           strPtr("2026-09-01T17:00:00Z"),
			LastTraceID:        strPtr("trace-9"),
			Status:             model.InsightsAnomalyStatus("accepted"),
			Evidence:           `{"D2_egress":{"seen":1}}`,
			Risk: &model.InsightsRiskAssessment{
				Score:            108,
				Severity:         model.InsightsSeverityLow,
				Correlated:       true,
				CorrelationBonus: 109,
				FireAtScore:      110,
				Categories:       []string{"exfiltration"},
				Signals: []*model.InsightsRiskSignal{{
					Source:       "D2_egress",
					Category:     "network",
					Weight:       111,
					Confidence:   0.5,
					Contribution: 112,
					Rationale:    strPtr("new destination"),
					Mitre:        strPtr("T1041"),
					Owasp:        strPtr("A01"),
				}},
			},
			ClassFindings: []*model.InsightsAnomalyClassFinding{{
				Class:          "D2_egress",
				Kind:           model.InsightsAnomalyClassFindingKind("new_value"),
				Title:          "New egress destination",
				Summary:        "checkout called db:5432 for the first time",
				SpanHighlights: []*model.InsightsAnomalySpanHighlight{{SpanID: "span-1", Severity: "high", Reason: "unknown peer"}},
				AttrHighlights: []*model.InsightsAnomalyAttrHighlight{{SpanID: "span-2", Key: "net.peer.name", Value: "db"}},
				Metric:         &model.InsightsAnomalyMetricComparison{Observed: 113, Baseline: 114, Unit: "us"},
				RawEvidence:    strPtr(`{"peer":"db"}`),
			}},
			AnomalyTrace:  &model.InsightsObservation{TraceID: "trace-anomaly", TransactionID: "0"},
			BaselineTrace: &model.InsightsObservation{TraceID: "trace-baseline", TransactionID: "0"},
		}, got)
	})

	// evidence and classFindings are non-nullable in the schema.
	t.Run("issue detail without evidence", func(t *testing.T) {
		got, err := AnomalyIssueToModel(AnomalyIssue{TransactionID: 115})
		require.NoError(t, err)

		assert.Equal(t, "{}", got.Evidence)
		assert.Equal(t, []*model.InsightsAnomalyClassFinding{}, got.ClassFindings)
		assert.Nil(t, got.Risk)
		assert.Nil(t, got.Kind)
		assert.Nil(t, got.AnomalyTrace)
		assert.Nil(t, got.BaselineTrace)
	})

	t.Run("class finding without a metric", func(t *testing.T) {
		got, err := AnomalyClassFindingToModel(AnomalyClassFinding{Class: "D2_egress"})
		require.NoError(t, err)
		assert.Nil(t, got.Metric)
		assert.Nil(t, got.RawEvidence)
	})
}

func TestGuardrailConvertersMapEveryField(t *testing.T) {
	t.Run("guardrail", func(t *testing.T) {
		got := GuardrailsToModel([]Guardrail{{
			Scope:    "service",
			ScopeKey: "prod/checkout",
			Rules: []GuardrailRule{{
				Key:       "allowed_egress",
				Label:     "Allowed egress",
				Mode:      "enforce",
				Allowlist: []string{"db:5432"},
				Origin:    "auto_transaction_guardrail",
			}},
		}})

		assert.Equal(t, []*model.InsightsGuardrail{{
			Scope:    model.InsightsPolicyScope("service"),
			ScopeKey: "prod/checkout",
			Rules: []*model.InsightsGuardrailRule{{
				Key:       "allowed_egress",
				Label:     "Allowed egress",
				Mode:      model.InsightsRuleMode("enforce"),
				Allowlist: []string{"db:5432"},
				Origin:    strPtr("auto_transaction_guardrail"),
			}},
		}}, got)
	})

	// A manually created rule has no origin, and the UI distinguishes it from an
	// auto-created one, so an empty origin must not surface as "".
	t.Run("rule without an origin", func(t *testing.T) {
		got := GuardrailRuleToModel(GuardrailRule{Key: "allowed_callers"})
		assert.Nil(t, got.Origin)
	})

	t.Run("violation", func(t *testing.T) {
		got := GuardrailViolationsToModel([]GuardrailViolation{{
			Service:     "checkout",
			Namespace:   "prod",
			ScopeKey:    "prod/checkout",
			RuleKey:     "allowed_egress",
			RuleLabel:   "Allowed egress",
			Offending:   strPtr("db:5432"),
			Occurrences: 121,
			MaxScore:    122,
			Severity:    "high",
			LastSeen:    strPtr("2026-09-01T18:00:00Z"),
			Status:      "dismissed",
		}})

		assert.Equal(t, []*model.InsightsGuardrailViolation{{
			Service:     "checkout",
			Namespace:   "prod",
			ScopeKey:    "prod/checkout",
			RuleKey:     "allowed_egress",
			RuleLabel:   "Allowed egress",
			Offending:   strPtr("db:5432"),
			Occurrences: 121,
			MaxScore:    122,
			Severity:    model.InsightsSeverityHigh,
			LastSeen:    strPtr("2026-09-01T18:00:00Z"),
			Status:      model.InsightsViolationStatusDismissed,
		}}, got)
	})

	t.Run("violation detail", func(t *testing.T) {
		got := GuardrailViolationDetailToModel(GuardrailViolationDetail{
			Service:        "checkout",
			Namespace:      "prod",
			ScopeKey:       "prod/checkout",
			RuleKey:        "allowed_egress",
			RuleLabel:      "Allowed egress",
			Offending:      "db:5432",
			Occurrences:    123,
			MaxScore:       124,
			Severity:       "low",
			LastSeen:       strPtr("2026-09-01T19:00:00Z"),
			Status:         "accepted",
			Summary:        "checkout reached db:5432",
			SpanHighlights: []AnomalySpanHighlight{{SpanID: "span-1", Severity: "medium", Reason: "forbidden edge"}},
			EvidenceTrace:  &Observation{TraceID: "trace-evidence", RawOTLP: "YWJj"},
		})

		assert.Equal(t, &model.InsightsGuardrailViolationDetail{
			Service:        "checkout",
			Namespace:      "prod",
			ScopeKey:       "prod/checkout",
			RuleKey:        "allowed_egress",
			RuleLabel:      "Allowed egress",
			Offending:      "db:5432",
			Occurrences:    123,
			MaxScore:       124,
			Severity:       model.InsightsSeverityLow,
			LastSeen:       strPtr("2026-09-01T19:00:00Z"),
			Status:         model.InsightsViolationStatusAccepted,
			Summary:        "checkout reached db:5432",
			SpanHighlights: []*model.InsightsAnomalySpanHighlight{{SpanID: "span-1", Severity: "medium", Reason: "forbidden edge"}},
			EvidenceTrace:  &model.InsightsObservation{TraceID: "trace-evidence", TransactionID: "0", RawOtlp: "YWJj"},
		}, got)
	})

	t.Run("violation detail without a retained sample", func(t *testing.T) {
		got := GuardrailViolationDetailToModel(GuardrailViolationDetail{ScopeKey: "prod/checkout"})
		assert.Nil(t, got.EvidenceTrace)
	})
}

func TestCatalogConverterMapsEveryField(t *testing.T) {
	got, err := CatalogToModel(Catalog{
		DeviationClasses: []CatalogClass{{
			ID:                  "D2_egress",
			Label:               "Unexpected egress",
			Description:         strPtr("class description"),
			BaselineLabel:       "Known destinations",
			BaselineDescription: strPtr("baseline description"),
			Category:            "network",
			CategoryLabel:       strPtr("Network"),
			Weight:              131,
			Mitre:               strPtr("T1041"),
			Owasp:               strPtr("A01"),
		}},
		Enrichers: []CatalogEnricher{{
			Source:      "secrets",
			ParentClass: "D2_egress",
			Category:    "data",
			Weight:      132,
			Label:       "Secret keywords",
			Description: strPtr("enricher description"),
			List: &EnricherList{
				Key:     "secrets",
				Label:   "Secrets",
				Hint:    strPtr("one keyword per line"),
				Default: []string{"token"},
			},
		}},
		Categories: []CatalogCategory{{
			ID:    "network",
			Label: "Network",
			Base:  133,
			Mitre: strPtr("TA0010"),
			Owasp: strPtr("A02"),
		}},
		GuardrailRules: []CatalogGuardrailRule{{
			Key:         "allowed_egress",
			Label:       "Allowed egress",
			Description: strPtr("rule description"),
			Category:    "network",
			Weight:      134,
			Hint:        strPtr("host:port"),
		}},
		SeverityBands:    []SeverityBand{{Severity: "high", MinScore: 135}},
		CorrelationBonus: 136,
		Tags:             map[string]string{"source": "builtin"},
	})
	require.NoError(t, err)

	assert.Equal(t, &model.InsightsCatalog{
		DeviationClasses: []*model.InsightsCatalogClass{{
			ID:                  "D2_egress",
			Label:               "Unexpected egress",
			Description:         strPtr("class description"),
			BaselineLabel:       "Known destinations",
			BaselineDescription: strPtr("baseline description"),
			Category:            "network",
			CategoryLabel:       strPtr("Network"),
			Weight:              131,
			Mitre:               strPtr("T1041"),
			Owasp:               strPtr("A01"),
		}},
		Enrichers: []*model.InsightsCatalogEnricher{{
			Source:      "secrets",
			ParentClass: "D2_egress",
			Category:    "data",
			Weight:      132,
			Label:       "Secret keywords",
			Description: strPtr("enricher description"),
			List: &model.InsightsEnricherList{
				Key:     "secrets",
				Label:   "Secrets",
				Hint:    strPtr("one keyword per line"),
				Default: []string{"token"},
			},
		}},
		Categories: []*model.InsightsCatalogCategory{{
			ID:    "network",
			Label: "Network",
			Base:  133,
			Mitre: strPtr("TA0010"),
			Owasp: strPtr("A02"),
		}},
		GuardrailRules: []*model.InsightsCatalogGuardrailRule{{
			Key:         "allowed_egress",
			Label:       "Allowed egress",
			Description: strPtr("rule description"),
			Category:    "network",
			Weight:      134,
			Hint:        strPtr("host:port"),
		}},
		SeverityBands:    []*model.InsightsSeverityBand{{Severity: model.InsightsSeverityHigh, MinScore: 135}},
		CorrelationBonus: 136,
		Tags:             `{"source":"builtin"}`,
	}, got)
}

func TestCatalogTagsAreAlwaysAJSONObject(t *testing.T) {
	got, err := CatalogToModel(Catalog{})
	require.NoError(t, err)
	assert.Equal(t, "{}", got.Tags)
}

func TestCatalogEnricherWithoutAList(t *testing.T) {
	got := CatalogEnricherToModel(CatalogEnricher{Source: "secrets"})
	assert.Nil(t, got.List)
}

func TestSystemSettingsConverterMapsEveryField(t *testing.T) {
	got := SystemSettingsToModel(SystemSettings{
		Sampling:  SystemSamplingSettings{ExamplesPerTransaction: 141, ExampleSampleIntervalSeconds: 142},
		Retention: SystemRetentionSettings{ObservationRetentionDays: 143},
		Findings:  SystemFindingsSettings{DefaultWindowHours: 144, MaxWindowHours: 145},
		Capacity:  SystemCapacitySettings{MaxResidentTransactions: 146, MaxBaselineSetMembers: 147},
		Writeback: SystemWritebackSettings{FlushIntervalSeconds: 148},
		Detection: SystemDetectionSettings{AutoTransactionGuardrail: true},
		Identity: SystemIdentitySettings{TransactionIdentityDimensions: []SystemTransactionIdentityDimension{
			{Key: "tenant", Enabled: true},
			{Key: "region", Enabled: false},
		}},
	})

	assert.Equal(t, &model.InsightsSystemSettings{
		Sampling:  &model.InsightsSystemSamplingSettings{ExamplesPerTransaction: 141, ExampleSampleIntervalSeconds: 142},
		Retention: &model.InsightsSystemRetentionSettings{ObservationRetentionDays: 143},
		Capacity:  &model.InsightsSystemCapacitySettings{MaxResidentTransactions: 146, MaxBaselineSetMembers: 147},
		Writeback: &model.InsightsSystemWritebackSettings{FlushIntervalSeconds: 148},
		Detection: &model.InsightsSystemDetectionSettings{AutoTransactionGuardrail: true},
		Identity: &model.InsightsSystemIdentitySettings{
			TransactionIdentityDimensions: []*model.InsightsSystemTransactionIdentityDimension{
				{Key: "tenant", Enabled: true},
				{Key: "region"},
			},
		},
	}, got)
}

func TestSystemIdentityDimensionsAreNeverNull(t *testing.T) {
	got := SystemSettingsToModel(SystemSettings{})
	require.NotNil(t, got.Identity)
	assert.Equal(t, []*model.InsightsSystemTransactionIdentityDimension{}, got.Identity.TransactionIdentityDimensions)
}

func TestStorageHealthConverterMapsEveryField(t *testing.T) {
	got := StorageHealthToModel(StorageHealth{
		CheckedAt: "2026-09-01T20:00:00Z",
		Status:    "degraded",
		Connection: StorageConnection{
			Reachable:     true,
			PingLatencyMs: 151,
			Version:       "24.3",
			Database:      "insights",
			Error:         strPtr("slow"),
		},
		Disk: StorageDisk{
			InsightsBytes:  152,
			TotalBytes:     153,
			UsedBytes:      154,
			AvailableBytes: 155,
			UsedPercent:    91.5,
			Status:         "warning",
		},
		Memory: StorageMemory{
			ProcessRSSBytes:  156,
			CgroupUsedBytes:  157,
			CgroupTotalBytes: 158,
			UptimeSeconds:    159.5,
		},
		Queries: StorageQueries{
			Active:          160,
			LongestSeconds:  161.5,
			PeakMemoryBytes: 162,
			Items:           []StorageQuery{{ElapsedSeconds: 1.5, User: "reader", MemoryBytes: 163, Query: "SELECT 1"}},
			RecentHeaviest:  []StorageQuery{{Query: "SELECT heaviest", FinishedAt: strPtr("2026-09-01T19:00:00Z")}},
			RecentLatest:    []StorageQuery{{Query: "SELECT latest"}},
		},
		Tables:        []StorageTable{{Name: "observations", Rows: 164, BytesOnDisk: 165, ActiveParts: 166}},
		WritePressure: StorageWritePressure{UnfinishedMutations: 167, ActiveParts: 168},
		Writeback: StorageWriteback{
			FlushIntervalSeconds: 169,
			EverFlushed:          true,
			LastFlushOK:          false,
			LastFlushAt:          strPtr("2026-09-01T18:30:00Z"),
			LastFlushError:       strPtr("timeout"),
		},
	})

	assert.Equal(t, &model.InsightsStorageHealth{
		CheckedAt: "2026-09-01T20:00:00Z",
		Status:    model.InsightsStorageHealthStatus("degraded"),
		Connection: &model.InsightsStorageConnection{
			Reachable:     true,
			PingLatencyMs: 151,
			Version:       "24.3",
			Database:      "insights",
			Error:         strPtr("slow"),
		},
		Disk: &model.InsightsStorageDisk{
			InsightsBytes:  152,
			TotalBytes:     153,
			UsedBytes:      154,
			AvailableBytes: 155,
			UsedPercent:    91.5,
			Status:         model.InsightsStorageDiskStatus("warning"),
		},
		Memory: &model.InsightsStorageMemory{
			ProcessRssBytes:  156,
			CgroupUsedBytes:  157,
			CgroupTotalBytes: 158,
			UptimeSeconds:    159.5,
		},
		Queries: &model.InsightsStorageQueries{
			Active:          160,
			LongestSeconds:  161.5,
			PeakMemoryBytes: 162,
			Items:           []*model.InsightsStorageQuery{{ElapsedSeconds: 1.5, User: "reader", MemoryBytes: 163, Query: "SELECT 1"}},
			RecentHeaviest:  []*model.InsightsStorageQuery{{Query: "SELECT heaviest", FinishedAt: strPtr("2026-09-01T19:00:00Z")}},
			RecentLatest:    []*model.InsightsStorageQuery{{Query: "SELECT latest"}},
		},
		Tables:        []*model.InsightsStorageTable{{Name: "observations", Rows: 164, BytesOnDisk: 165, ActiveParts: 166}},
		WritePressure: &model.InsightsStorageWritePressure{UnfinishedMutations: 167, ActiveParts: 168},
		Writeback: &model.InsightsStorageWriteback{
			FlushIntervalSeconds: 169,
			EverFlushed:          true,
			LastFlushOk:          false,
			LastFlushAt:          strPtr("2026-09-01T18:30:00Z"),
			LastFlushError:       strPtr("timeout"),
		},
	}, got)
}

func TestResultConvertersMapEveryField(t *testing.T) {
	assert.Equal(t, &model.InsightsPromoteResult{
		TransactionID: "171",
		Class:         "D3_latency",
		Promoted:      true,
	}, PromoteResultToModel(PromoteResult{TransactionID: 171, Class: "D3_latency", Promoted: true}))

	assert.Equal(t, &model.InsightsBulkPromoteResult{Promoted: 172}, BulkPromoteResultToModel(BulkPromoteResult{Promoted: 172}))
	assert.Equal(t, &model.InsightsBulkDeleteResult{Deleted: 173}, BulkDeleteResultToModel(BulkDeleteResult{Deleted: 173}))
	assert.Equal(t, &model.InsightsBulkResolveResult{
		Resolution: "dismiss",
		Resolved:   174,
	}, BulkResolveResultToModel(BulkResolveResult{Resolution: "dismiss", Resolved: 174}))
}

// Optional ids and counters are drill-down keys and progress numbers: an
// absent one has to stay absent, not become "" or 0, which the UI would follow
// or display.
func TestOptionalNumbersStayAbsent(t *testing.T) {
	finding := FindingToModel(Finding{Kind: "guardrail_violation"})
	assert.Nil(t, finding.TransactionID)

	anomaly := AnomalySummaryToModel(AnomalySummary{Signature: "sig-1"})
	assert.Nil(t, anomaly.PolicyID)

	baseline, err := BaselineClassToModel(BaselineClass{})
	require.NoError(t, err)
	assert.Nil(t, baseline.DataSchemaVersion)
	assert.Nil(t, baseline.ObservationCountAtLastChange)

	condition := LearningConditionToModel(LearningCondition{Type: "observations"})
	assert.Nil(t, condition.MinObservations)
	assert.Nil(t, condition.MinDurationMinutes)
	assert.Nil(t, condition.StableObservations)
	assert.Nil(t, condition.StableMinutes)

	fromInput := LearningConditionFromInput(model.InsightsLearningConditionInput{})
	assert.Nil(t, fromInput.MinObservations)
	assert.Nil(t, fromInput.MinDurationMinutes)
	assert.Nil(t, fromInput.StableObservations)
	assert.Nil(t, fromInput.StableMinutes)
}

// A REST list that is absent must stay absent rather than becoming an empty
// list, except where the schema declares the field non-nullable.
func TestListConvertersPreserveAbsentLists(t *testing.T) {
	assert.Nil(t, ServiceStatsToModel(nil))
	assert.Nil(t, TransactionStatsToModel(nil))
	assert.Nil(t, ObservationSummariesToModel(nil))
	assert.Nil(t, FindingsToModel(nil))
	assert.Nil(t, AnomalySummariesToModel(nil))
	assert.Nil(t, GuardrailsToModel(nil))
	assert.Nil(t, GuardrailViolationsToModel(nil))
	assert.Nil(t, LearningPoliciesToModel(nil))

	policies, err := PoliciesToModel(nil)
	require.NoError(t, err)
	assert.Nil(t, policies)

	baselines, err := BaselineClassesToModel(nil)
	require.NoError(t, err)
	assert.Nil(t, baselines)
}
