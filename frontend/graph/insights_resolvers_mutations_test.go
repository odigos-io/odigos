package graph

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// insightsMutationCase drives one insights mutation against a stubbed engine
// and states the exact requests it is expected to produce.
type insightsMutationCase struct {
	name string
	// setup registers the endpoints the mutation should reach.
	setup func(*insightsEngineStub)
	call  func(*Resolver) (any, error)
	// writeMethod and writePath identify the request that changes state; the
	// failure test below breaks exactly that one.
	writeMethod string
	writePath   string
	wantCalls   []insightsEngineCall
}

func insightsMutationCases() []insightsMutationCase {
	ctx := context.Background()
	noQuery := url.Values{}

	return []insightsMutationCase{
		{
			name: "promote a baseline class",
			setup: func(s *insightsEngineStub) {
				s.respond(http.MethodPost, "/api/v1/transactions/42/baseline/D2_egress/promote",
					`{"transaction_id":42,"class":"D2_egress","promoted":true}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().PromoteInsightsBaselineClass(ctx, "42", model.InsightsDeviationClassD2Egress)
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions/42/baseline/D2_egress/promote",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions/42/baseline/D2_egress/promote", query: noQuery},
			},
		},
		{
			name: "reset a baseline class",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/transactions/42/baseline/D2_egress/reset")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().ResetInsightsBaselineClass(ctx, "42", model.InsightsDeviationClassD2Egress)
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions/42/baseline/D2_egress/reset",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions/42/baseline/D2_egress/reset", query: noQuery},
			},
		},
		{
			name: "reset every baseline of a transaction",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/transactions/42/baselines/reset")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().ResetInsightsTransactionBaselines(ctx, "42")
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions/42/baselines/reset",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions/42/baselines/reset", query: noQuery},
			},
		},
		{
			name: "promote every baseline of a transaction",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/transactions/42/baselines/promote")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().PromoteInsightsTransactionBaselines(ctx, "42")
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions/42/baselines/promote",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions/42/baselines/promote", query: noQuery},
			},
		},
		{
			name: "bulk promote transactions",
			setup: func(s *insightsEngineStub) {
				s.respond(http.MethodPost, "/api/v1/transactions/promote", `{"promoted":2}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().BulkPromoteInsightsTransactions(ctx, []string{"7", "42"})
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions/promote",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions/promote", query: noQuery, body: `{"transaction_ids":[7,42]}`},
			},
		},
		{
			name: "force promote a service",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/services/transaction-guardrail/force-promote")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().ForcePromoteInsightsService(ctx, "prod", "checkout")
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/services/transaction-guardrail/force-promote",
			wantCalls: []insightsEngineCall{{
				method: http.MethodPost,
				path:   "/api/v1/services/transaction-guardrail/force-promote",
				query:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
			}},
		},
		{
			name: "enable a transaction guardrail",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/services/transaction-guardrail/enable")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().EnableInsightsTransactionGuardrail(ctx, "prod", "checkout")
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/services/transaction-guardrail/enable",
			wantCalls: []insightsEngineCall{{
				method: http.MethodPost,
				path:   "/api/v1/services/transaction-guardrail/enable",
				query:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
			}},
		},
		{
			name: "disable a transaction guardrail",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/services/transaction-guardrail/disable")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().DisableInsightsTransactionGuardrail(ctx, "prod", "checkout")
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/services/transaction-guardrail/disable",
			wantCalls: []insightsEngineCall{{
				method: http.MethodPost,
				path:   "/api/v1/services/transaction-guardrail/disable",
				query:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
			}},
		},
		{
			name: "delete a transaction",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodDelete, "/api/v1/transactions/42")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().DeleteInsightsTransaction(ctx, "42")
			},
			writeMethod: http.MethodDelete,
			writePath:   "/api/v1/transactions/42",
			wantCalls: []insightsEngineCall{
				{method: http.MethodDelete, path: "/api/v1/transactions/42", query: noQuery},
			},
		},
		{
			name: "bulk delete transactions",
			setup: func(s *insightsEngineStub) {
				s.respond(http.MethodPost, "/api/v1/transactions", `{"deleted":2}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().BulkDeleteInsightsTransactions(ctx, []string{"7", "42"})
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/transactions",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/transactions", query: noQuery, body: `{"transaction_ids":[7,42]}`},
			},
		},
		{
			name: "upsert a policy",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPut, "/api/v1/policies").
					respond(http.MethodGet, "/api/v1/policies",
						`{"items":[{"id":5,"name":"checkout","enabled":true,"fire_at_score":70,"scope":"service","scope_key":"prod/checkout"}]}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().UpsertInsightsPolicy(ctx, model.InsightsPolicyInput{
					Name:        "checkout",
					Enabled:     true,
					FireAtScore: 70,
					Scope:       model.InsightsPolicyScopeService,
					ScopeKey:    "prod/checkout",
				})
			},
			writeMethod: http.MethodPut,
			writePath:   "/api/v1/policies",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPut, path: "/api/v1/policies", query: noQuery,
					body: `{"id":0,"name":"checkout","enabled":true,"fire_at_score":70,"scope":"service","scope_key":"prod/checkout"}`},
				{method: http.MethodGet, path: "/api/v1/policies", query: noQuery},
			},
		},
		{
			name: "delete a policy",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodDelete, "/api/v1/policies")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().DeleteInsightsPolicy(ctx, model.InsightsPolicyScopeService, "prod/checkout")
			},
			writeMethod: http.MethodDelete,
			writePath:   "/api/v1/policies",
			wantCalls: []insightsEngineCall{{
				method: http.MethodDelete,
				path:   "/api/v1/policies",
				query:  url.Values{"scope": {"service"}, "scope_key": {"prod/checkout"}},
			}},
		},
		{
			name: "upsert a learning policy",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPut, "/api/v1/learning-policies").
					respond(http.MethodGet, "/api/v1/learning-policies",
						`{"items":[{"class":"D2_egress","mode":"all","scope":"service","scope_key":"prod/checkout"}]}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().UpsertInsightsLearningPolicy(ctx, model.InsightsLearningPolicyInput{
					Class:    model.InsightsDeviationClassD2Egress,
					Mode:     model.InsightsLearningModeAll,
					Scope:    model.InsightsPolicyScopeService,
					ScopeKey: "prod/checkout",
				})
			},
			writeMethod: http.MethodPut,
			writePath:   "/api/v1/learning-policies",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPut, path: "/api/v1/learning-policies", query: noQuery,
					body: `{"class":"D2_egress","mode":"all","scope":"service","scope_key":"prod/checkout"}`},
				{method: http.MethodGet, path: "/api/v1/learning-policies", query: noQuery},
			},
		},
		{
			name: "delete a learning policy",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodDelete, "/api/v1/learning-policies")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().DeleteInsightsLearningPolicy(ctx,
					model.InsightsDeviationClassD2Egress, model.InsightsPolicyScopeService, "prod/checkout")
			},
			writeMethod: http.MethodDelete,
			writePath:   "/api/v1/learning-policies",
			wantCalls: []insightsEngineCall{{
				method: http.MethodDelete,
				path:   "/api/v1/learning-policies",
				query:  url.Values{"class": {"D2_egress"}, "scope": {"service"}, "scope_key": {"prod/checkout"}},
			}},
		},
		{
			name: "resolve an anomaly",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/anomalies/42/sig-1")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().ResolveInsightsAnomaly(ctx, "42", "sig-1", model.InsightsAnomalyResolutionAccept)
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/anomalies/42/sig-1",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/anomalies/42/sig-1", query: noQuery, body: `{"resolution":"accept"}`},
			},
		},
		{
			name: "bulk resolve anomalies",
			setup: func(s *insightsEngineStub) {
				s.respond(http.MethodPost, "/api/v1/anomalies", `{"resolution":"dismiss","resolved":1}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().BulkResolveInsightsAnomalies(ctx, model.InsightsBulkResolutionDismiss,
					[]*model.InsightsAnomalyRefInput{{TransactionID: "42", Signature: "sig-1"}})
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/anomalies",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/anomalies", query: noQuery,
					body: `{"resolution":"dismiss","items":[{"transaction_id":42,"signature":"sig-1"}]}`},
			},
		},
		{
			name: "upsert a guardrail",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPut, "/api/v1/guardrails").
					respond(http.MethodGet, "/api/v1/guardrails",
						`{"items":[{"scope":"service","scope_key":"prod/checkout","rules":[{"key":"allowed_egress","label":"Egress","mode":"enforce"}]}]}`)
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().UpsertInsightsGuardrail(ctx, model.InsightsGuardrailInput{
					Scope:    model.InsightsPolicyScopeService,
					ScopeKey: "prod/checkout",
					Rules: []*model.InsightsGuardrailRuleInput{{
						Key:   "allowed_egress",
						Label: "Egress",
						Mode:  model.InsightsRuleModeEnforce,
					}},
				})
			},
			writeMethod: http.MethodPut,
			writePath:   "/api/v1/guardrails",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPut, path: "/api/v1/guardrails", query: noQuery,
					body: `{"scope":"service","scope_key":"prod/checkout","rules":[{"key":"allowed_egress","label":"Egress","mode":"enforce"}]}`},
				{method: http.MethodGet, path: "/api/v1/guardrails", query: noQuery},
			},
		},
		{
			name: "delete a guardrail",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodDelete, "/api/v1/guardrails")
			},
			call: func(r *Resolver) (any, error) {
				scope := model.InsightsPolicyScopeTransaction
				return r.Mutation().DeleteInsightsGuardrail(ctx, "42", &scope)
			},
			writeMethod: http.MethodDelete,
			writePath:   "/api/v1/guardrails",
			wantCalls: []insightsEngineCall{{
				method: http.MethodDelete,
				path:   "/api/v1/guardrails",
				query:  url.Values{"scope": {"transaction"}, "scope_key": {"42"}},
			}},
		},
		{
			name: "seed a guardrail",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/guardrails/seed")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().SeedInsightsGuardrail(ctx, model.InsightsGuardrailSeedInput{
					ScopeKey: "prod/checkout",
					Items:    `{"allowed_callers":["frontend"]}`,
				})
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/guardrails/seed",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/guardrails/seed", query: noQuery,
					body: `{"scope_key":"prod/checkout","items":{"allowed_callers":["frontend"]}}`},
			},
		},
		{
			name: "accept a guardrail violation",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/guardrails/violations/accept")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().AcceptInsightsGuardrailViolation(ctx, insightsViolationAction())
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/guardrails/violations/accept",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/guardrails/violations/accept", query: noQuery,
					body: insightsViolationActionBody},
			},
		},
		{
			name: "dismiss a guardrail violation",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/guardrails/violations/dismiss")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().DismissInsightsGuardrailViolation(ctx, insightsViolationAction())
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/guardrails/violations/dismiss",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/guardrails/violations/dismiss", query: noQuery,
					body: insightsViolationActionBody},
			},
		},
		{
			name: "reopen a guardrail violation",
			setup: func(s *insightsEngineStub) {
				s.accept(http.MethodPost, "/api/v1/guardrails/violations/reopen")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().ReopenInsightsGuardrailViolation(ctx, insightsViolationAction())
			},
			writeMethod: http.MethodPost,
			writePath:   "/api/v1/guardrails/violations/reopen",
			wantCalls: []insightsEngineCall{
				{method: http.MethodPost, path: "/api/v1/guardrails/violations/reopen", query: noQuery,
					body: insightsViolationActionBody},
			},
		},
		{
			name: "update system settings",
			setup: func(s *insightsEngineStub) {
				s.respond(http.MethodGet, "/api/v1/system-settings", insightsStoredSystemSettings).
					accept(http.MethodPut, "/api/v1/system-settings")
			},
			call: func(r *Resolver) (any, error) {
				return r.Mutation().UpdateInsightsSystemSettings(ctx, model.InsightsSystemSettingsInput{
					Sampling:  &model.InsightsSystemSamplingSettingsInput{},
					Retention: &model.InsightsSystemRetentionSettingsInput{},
					Capacity:  &model.InsightsSystemCapacitySettingsInput{},
					Writeback: &model.InsightsSystemWritebackSettingsInput{},
					Detection: &model.InsightsSystemDetectionSettingsInput{},
					Identity:  &model.InsightsSystemIdentitySettingsInput{},
				})
			},
			writeMethod: http.MethodPut,
			writePath:   "/api/v1/system-settings",
			wantCalls: []insightsEngineCall{
				{method: http.MethodGet, path: "/api/v1/system-settings", query: url.Values{}},
				{method: http.MethodPut, path: "/api/v1/system-settings", query: url.Values{}, body: `{
					"sampling":{"examples_per_transaction":0,"example_sample_interval_seconds":0},
					"retention":{"observation_retention_days":0},
					"findings":{"default_window_hours":24,"max_window_hours":168},
					"capacity":{"max_resident_transactions":0,"max_baseline_set_members":0},
					"writeback":{"flush_interval_seconds":0},
					"detection":{"auto_transaction_guardrail":false},
					"identity":{"transaction_identity_dimensions":[]}
				}`},
				{method: http.MethodGet, path: "/api/v1/system-settings", query: url.Values{}},
			},
		},
	}
}

const insightsViolationActionBody = `{"scope_key":"prod/checkout","rule_key":"allowed_egress","offending":"db:5432"}`

const insightsStoredSystemSettings = `{
	"sampling":{"examples_per_transaction":1,"example_sample_interval_seconds":2},
	"retention":{"observation_retention_days":3},
	"findings":{"default_window_hours":24,"max_window_hours":168},
	"capacity":{"max_resident_transactions":4,"max_baseline_set_members":5},
	"writeback":{"flush_interval_seconds":6},
	"detection":{"auto_transaction_guardrail":true},
	"identity":{"transaction_identity_dimensions":[]}
}`

func insightsViolationAction() model.InsightsViolationActionInput {
	return model.InsightsViolationActionInput{
		ScopeKey:  "prod/checkout",
		RuleKey:   "allowed_egress",
		Offending: "db:5432",
	}
}

// Several insights mutations are one path segment apart — accept/dismiss/reopen
// a violation, enable/disable/force-promote a transaction guardrail,
// promote/reset a baseline — and every one of them takes the same arguments.
// Calling the wrong neighbour therefore compiles, returns true and applies the
// opposite action, so each mutation has to be pinned to the request it makes.
func TestEveryInsightsMutationIssuesTheRequestItNames(t *testing.T) {
	cases := insightsMutationCases()
	require.Len(t, cases, insightsMutationCount(t))

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t)
			tt.setup(stub)

			got, err := tt.call(newInsightsResolver(t, stub))
			require.NoError(t, err)
			assert.NotNil(t, got)
			if succeeded, isBool := got.(bool); isBool {
				assert.True(t, succeeded)
			}

			recorded := stub.recorded()
			require.Len(t, recorded, len(tt.wantCalls))
			for i, want := range tt.wantCalls {
				assert.Equal(t, want.method, recorded[i].method)
				assert.Equal(t, want.path, recorded[i].path)
				assert.Equal(t, want.query, recorded[i].query)
				if want.body == "" {
					assert.Empty(t, recorded[i].body)
					continue
				}
				assert.JSONEq(t, want.body, recorded[i].body)
			}
		})
	}
}

// A mutation that reports success after the engine rejected it leaves the UI
// showing state that was never persisted, so every one has to propagate the
// failure with the code the engine returned.
func TestEveryInsightsMutationPropagatesARejectionFromTheEngine(t *testing.T) {
	cases := insightsMutationCases()
	require.Len(t, cases, insightsMutationCount(t))

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t)
			tt.setup(stub)
			stub.fail(tt.writeMethod, tt.writePath, http.StatusConflict, insights.ErrorCodeConflict)

			got, err := tt.call(newInsightsResolver(t, stub))
			require.Error(t, err)
			requireInsightsErrorCode(t, err, insights.CodeConflict)
			if succeeded, isBool := got.(bool); isBool {
				assert.False(t, succeeded)
			}
		})
	}
}

// insightsMutationCount counts the insights mutations the generated schema
// declares. Deriving it keeps the two tables above honest: a mutation added
// later fails them until it has a case, instead of going silently untested.
func insightsMutationCount(t *testing.T) int {
	t.Helper()

	mutations := reflect.TypeOf((*MutationResolver)(nil)).Elem()
	count := 0
	for i := 0; i < mutations.NumMethod(); i++ {
		if strings.Contains(mutations.Method(i).Name, "Insights") {
			count++
		}
	}
	require.Positive(t, count)
	return count
}
