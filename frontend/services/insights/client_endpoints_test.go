package insights

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturedRequest is what the fake insights engine saw for a single call.
type capturedRequest struct {
	method string
	path   string
	query  url.Values
	body   string
}

// newCapturingClient serves responseBody (or 204 when it is empty) for every
// request and records the last one, so a test can assert the wire contract of a
// client method: verb, path, query parameters and request body.
func newCapturingClient(t *testing.T, responseBody string) (*Client, *capturedRequest) {
	t.Helper()

	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		*captured = capturedRequest{
			method: r.Method,
			path:   r.URL.Path,
			query:  r.URL.Query(),
			body:   strings.TrimSpace(string(body)),
		}
		if responseBody == "" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(server.URL)
	require.NoError(t, err)
	return client, captured
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

// TestClientEndpointContract pins the verb, path, query and body of every
// single-request client method. The insights engine is a separate service, so a
// wrong verb or path silently makes the UI action a no-op (or hits the wrong
// resource) with nothing in this repository to catch it. The confusable
// families — baseline promote/reset, transaction-guardrail
// enable/disable/force-promote and violation accept/dismiss/reopen — are the
// reason this is exhaustive rather than a sample.
func TestClientEndpointContract(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		call       func(*Client) (any, error)
		wantMethod string
		wantPath   string
		wantQuery  url.Values
		wantBody   string
		response   string
		want       any
	}{
		{
			name:       "list services",
			call:       func(c *Client) (any, error) { return c.ListServices(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/services",
			wantQuery:  url.Values{},
			response:   `{"items":[{"namespace":"prod","service":"checkout","transaction_count":2,"volume":9,"last_seen":"2026-09-01T00:00:00Z"}]}`,
			want: []ServiceStat{{
				Namespace:        "prod",
				Service:          "checkout",
				TransactionCount: 2,
				Volume:           9,
				LastSeen:         "2026-09-01T00:00:00Z",
			}},
		},
		{
			name:       "list service names",
			call:       func(c *Client) (any, error) { return c.ListServiceNames(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/service-names",
			wantQuery:  url.Values{},
			response:   `{"items":["checkout","payments"]}`,
			want:       []string{"checkout", "payments"},
		},
		{
			name:       "get service profile",
			call:       func(c *Client) (any, error) { return c.GetServiceProfile(ctx, "prod", "checkout") },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/services/profile",
			wantQuery:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
			response:   `{"namespace":"prod","service":"checkout","callers":["frontend"]}`,
			want:       &ServiceProfile{Namespace: "prod", Service: "checkout", Callers: []string{"frontend"}},
		},
		{
			name:       "get service profile omits a blank namespace",
			call:       func(c *Client) (any, error) { return c.GetServiceProfile(ctx, "   ", "checkout") },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/services/profile",
			wantQuery:  url.Values{"service": {"checkout"}},
			response:   `{"service":"checkout"}`,
			want:       &ServiceProfile{Service: "checkout"},
		},
		{
			name:       "get blast radius with depth",
			call:       func(c *Client) (any, error) { return c.GetBlastRadius(ctx, "prod", "checkout", intPtr(3)) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/services/blast-radius",
			wantQuery:  url.Values{"namespace": {"prod"}, "service": {"checkout"}, "depth": {"3"}},
			response:   `{"root":{"namespace":"prod","service":"checkout","is_virtual":false}}`,
			want:       &BlastRadiusSubgraph{Root: BlastRadiusNode{Namespace: "prod", Service: "checkout"}},
		},
		{
			name:       "get blast radius without depth",
			call:       func(c *Client) (any, error) { return c.GetBlastRadius(ctx, "", "checkout", nil) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/services/blast-radius",
			wantQuery:  url.Values{"service": {"checkout"}},
			response:   `{"root":{"namespace":"","service":"checkout","is_virtual":true}}`,
			want:       &BlastRadiusSubgraph{Root: BlastRadiusNode{Service: "checkout", IsVirtual: true}},
		},
		{
			name:       "list transactions without filters",
			call:       func(c *Client) (any, error) { return c.ListTransactions(ctx, ListTransactionsParams{}) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions",
			wantQuery:  url.Values{},
			response:   `{"items":[]}`,
			want:       []TransactionStat{},
		},
		{
			name:       "get transaction",
			call:       func(c *Client) (any, error) { return c.GetTransaction(ctx, 7) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions/7",
			wantQuery:  url.Values{},
			response:   `{"id":7,"service":"checkout","kind":"HTTP"}`,
			want:       &Transaction{ID: 7, Service: "checkout", Kind: "HTTP"},
		},
		{
			name:       "delete transaction",
			call:       func(c *Client) (any, error) { return nil, c.DeleteTransaction(ctx, 7) },
			wantMethod: http.MethodDelete,
			wantPath:   "/api/v1/transactions/7",
			wantQuery:  url.Values{},
		},
		{
			name:       "bulk delete transactions",
			call:       func(c *Client) (any, error) { return c.BulkDeleteTransactions(ctx, []int64{7, 8}) },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions",
			wantQuery:  url.Values{},
			wantBody:   `{"transaction_ids":[7,8]}`,
			response:   `{"deleted":2}`,
			want:       &BulkDeleteResult{Deleted: 2},
		},
		{
			name:       "get transaction baseline",
			call:       func(c *Client) (any, error) { return c.GetTransactionBaseline(ctx, 7) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions/7/baseline",
			wantQuery:  url.Values{},
			response:   `{"items":[{"transaction_id":7,"class":"D2_egress","promoted":true}]}`,
			want:       []BaselineClass{{TransactionID: 7, Class: "D2_egress", Promoted: true}},
		},
		{
			name:       "promote a single baseline class",
			call:       func(c *Client) (any, error) { return c.PromoteBaselineClass(ctx, 7, "D2_egress") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions/7/baseline/D2_egress/promote",
			wantQuery:  url.Values{},
			response:   `{"transaction_id":7,"class":"D2_egress","promoted":true}`,
			want:       &PromoteResult{TransactionID: 7, Class: "D2_egress", Promoted: true},
		},
		{
			name:       "reset a single baseline class",
			call:       func(c *Client) (any, error) { return nil, c.ResetBaselineClass(ctx, 7, "D2_egress") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions/7/baseline/D2_egress/reset",
			wantQuery:  url.Values{},
		},
		{
			name:       "promote every baseline class of one transaction",
			call:       func(c *Client) (any, error) { return nil, c.PromoteTransactionBaselines(ctx, 7) },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions/7/baselines/promote",
			wantQuery:  url.Values{},
		},
		{
			name:       "reset every baseline class of one transaction",
			call:       func(c *Client) (any, error) { return nil, c.ResetTransactionBaselines(ctx, 7) },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions/7/baselines/reset",
			wantQuery:  url.Values{},
		},
		{
			name:       "bulk promote transactions",
			call:       func(c *Client) (any, error) { return c.BulkPromoteTransactions(ctx, []int64{7}) },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/transactions/promote",
			wantQuery:  url.Values{},
			wantBody:   `{"transaction_ids":[7]}`,
			response:   `{"promoted":3}`,
			want:       &BulkPromoteResult{Promoted: 3},
		},
		{
			name:       "force promote a service",
			call:       func(c *Client) (any, error) { return nil, c.ForcePromoteService(ctx, "prod", "checkout") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/services/transaction-guardrail/force-promote",
			wantQuery:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
		},
		{
			name:       "enable the transaction guardrail",
			call:       func(c *Client) (any, error) { return nil, c.EnableTransactionGuardrail(ctx, "prod", "checkout") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/services/transaction-guardrail/enable",
			wantQuery:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
		},
		{
			name:       "disable the transaction guardrail",
			call:       func(c *Client) (any, error) { return nil, c.DisableTransactionGuardrail(ctx, "prod", "checkout") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/services/transaction-guardrail/disable",
			wantQuery:  url.Values{"namespace": {"prod"}, "service": {"checkout"}},
		},
		{
			name: "list observations filtered by sample reason",
			call: func(c *Client) (any, error) {
				reason := SampleReasonExample
				return c.ListObservations(ctx, 7, ListObservationsParams{SampleReason: &reason})
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions/7/observations",
			wantQuery:  url.Values{"sample_reason": {"example"}},
			response:   `{"items":[{"trace_id":"trace-1","transaction_id":7,"duration_ms":5}]}`,
			want:       []ObservationSummary{{TraceID: "trace-1", TransactionID: 7, DurationMs: 5}},
		},
		{
			name:       "list observations without a sample reason",
			call:       func(c *Client) (any, error) { return c.ListObservations(ctx, 7, ListObservationsParams{}) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions/7/observations",
			wantQuery:  url.Values{},
			response:   `{"items":[]}`,
			want:       []ObservationSummary{},
		},
		{
			name:       "get one observation",
			call:       func(c *Client) (any, error) { return c.GetObservation(ctx, 7, "trace-1") },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/transactions/7/observations/trace-1",
			wantQuery:  url.Values{},
			response:   `{"trace_id":"trace-1","transaction_id":7,"raw_otlp":"YWJj"}`,
			want:       &Observation{TraceID: "trace-1", TransactionID: 7, RawOTLP: "YWJj"},
		},
		{
			name:       "list policies",
			call:       func(c *Client) (any, error) { return c.ListPolicies(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/policies",
			wantQuery:  url.Values{},
			response:   `{"items":[{"id":3,"name":"default","scope":"service","scope_key":"prod/checkout"}]}`,
			want:       []Policy{{ID: 3, Name: "default", Scope: "service", ScopeKey: "prod/checkout"}},
		},
		{
			name: "upsert a policy",
			call: func(c *Client) (any, error) {
				return nil, c.UpsertPolicy(ctx, Policy{
					ID:          3,
					Name:        "default",
					Enabled:     true,
					FireAtScore: 40,
					Scope:       "service",
					ScopeKey:    "prod/checkout",
				})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/v1/policies",
			wantQuery:  url.Values{},
			wantBody:   `{"id":3,"name":"default","enabled":true,"fire_at_score":40,"scope":"service","scope_key":"prod/checkout"}`,
		},
		{
			name:       "delete a policy",
			call:       func(c *Client) (any, error) { return nil, c.DeletePolicy(ctx, "service", "prod/checkout") },
			wantMethod: http.MethodDelete,
			wantPath:   "/api/v1/policies",
			wantQuery:  url.Values{"scope": {"service"}, "scope_key": {"prod/checkout"}},
		},
		{
			name:       "list learning policies",
			call:       func(c *Client) (any, error) { return c.ListLearningPolicies(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/learning-policies",
			wantQuery:  url.Values{},
			response:   `{"items":[{"class":"D2_egress","mode":"auto","scope":"global","scope_key":""}]}`,
			want:       []LearningPolicy{{Class: "D2_egress", Mode: "auto", Scope: "global"}},
		},
		{
			name: "upsert a learning policy",
			call: func(c *Client) (any, error) {
				return nil, c.UpsertLearningPolicy(ctx, LearningPolicy{
					Class:    "D2_egress",
					Mode:     "auto",
					Scope:    "service",
					ScopeKey: "prod/checkout",
				})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/v1/learning-policies",
			wantQuery:  url.Values{},
			wantBody:   `{"class":"D2_egress","mode":"auto","scope":"service","scope_key":"prod/checkout"}`,
		},
		{
			name: "delete a learning policy",
			call: func(c *Client) (any, error) {
				return nil, c.DeleteLearningPolicy(ctx, "D2_egress", "service", "prod/checkout")
			},
			wantMethod: http.MethodDelete,
			wantPath:   "/api/v1/learning-policies",
			wantQuery: url.Values{
				"class":     {"D2_egress"},
				"scope":     {"service"},
				"scope_key": {"prod/checkout"},
			},
		},
		{
			name: "list findings with every filter",
			call: func(c *Client) (any, error) {
				kind := FindingKind("anomaly")
				return c.ListFindings(ctx, ListFindingsParams{
					WindowHours: intPtr(24),
					Service:     strPtr("checkout"),
					Namespace:   strPtr("prod"),
					Status:      strPtr("open"),
					Kind:        &kind,
				})
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/findings",
			wantQuery: url.Values{
				"window_hours": {"24"},
				"service":      {"checkout"},
				"namespace":    {"prod"},
				"status":       {"open"},
				"kind":         {"anomaly"},
			},
			response: `{"items":[{"kind":"anomaly","service":"checkout","severity":"high"}]}`,
			want:     []Finding{{Kind: "anomaly", Service: "checkout", Severity: "high"}},
		},
		{
			name:       "list findings without filters",
			call:       func(c *Client) (any, error) { return c.ListFindings(ctx, ListFindingsParams{}) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/findings",
			wantQuery:  url.Values{},
			response:   `{"items":[]}`,
			want:       []Finding{},
		},
		{
			name: "list anomalies with every filter",
			call: func(c *Client) (any, error) {
				return c.ListAnomalies(ctx, ListAnomaliesParams{
					WindowHours: intPtr(72),
					Service:     strPtr("checkout"),
					Namespace:   strPtr("prod"),
					Status:      strPtr("open"),
				})
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/anomalies",
			wantQuery: url.Values{
				"window_hours": {"72"},
				"service":      {"checkout"},
				"namespace":    {"prod"},
				"status":       {"open"},
			},
			response: `{"items":[{"transaction_id":7,"signature":"sig-1","status":"open"}]}`,
			want:     []AnomalySummary{{TransactionID: 7, Signature: "sig-1", Status: "open"}},
		},
		{
			name:       "resolve one anomaly",
			call:       func(c *Client) (any, error) { return nil, c.ResolveAnomaly(ctx, 7, "sig-1", "accepted") },
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/anomalies/7/sig-1",
			wantQuery:  url.Values{},
			wantBody:   `{"resolution":"accepted"}`,
		},
		{
			name: "bulk resolve anomalies",
			call: func(c *Client) (any, error) {
				return c.BulkResolveAnomalies(ctx, BulkAnomalyRequest{
					Resolution: "dismiss",
					Items:      []AnomalyRef{{TransactionID: 7, Signature: "sig-1"}},
				})
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/anomalies",
			wantQuery:  url.Values{},
			wantBody:   `{"resolution":"dismiss","items":[{"transaction_id":7,"signature":"sig-1"}]}`,
			response:   `{"resolution":"dismiss","resolved":1}`,
			want:       &BulkResolveResult{Resolution: "dismiss", Resolved: 1},
		},
		{
			name:       "list guardrails",
			call:       func(c *Client) (any, error) { return c.ListGuardrails(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/guardrails",
			wantQuery:  url.Values{},
			response:   `{"items":[{"scope":"service","scope_key":"prod/checkout","rules":[]}]}`,
			want:       []Guardrail{{Scope: "service", ScopeKey: "prod/checkout", Rules: []GuardrailRule{}}},
		},
		{
			name: "upsert a guardrail",
			call: func(c *Client) (any, error) {
				return nil, c.UpsertGuardrail(ctx, Guardrail{
					Scope:    "service",
					ScopeKey: "prod/checkout",
					Rules:    []GuardrailRule{{Key: "allowed_egress", Label: "Egress", Mode: "enforce"}},
				})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/v1/guardrails",
			wantQuery:  url.Values{},
			wantBody:   `{"scope":"service","scope_key":"prod/checkout","rules":[{"key":"allowed_egress","label":"Egress","mode":"enforce"}]}`,
		},
		{
			name:       "delete a guardrail",
			call:       func(c *Client) (any, error) { return nil, c.DeleteGuardrail(ctx, "prod/checkout") },
			wantMethod: http.MethodDelete,
			wantPath:   "/api/v1/guardrails",
			wantQuery:  url.Values{"scope_key": {"prod/checkout"}},
		},
		{
			name: "seed a guardrail",
			call: func(c *Client) (any, error) {
				mode := RuleMode("enforce")
				return nil, c.SeedGuardrail(ctx, GuardrailSeedRequest{
					ScopeKey: "prod/checkout",
					Mode:     &mode,
					Items:    map[string][]string{"allowed_egress": {"db:5432"}},
				})
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/guardrails/seed",
			wantQuery:  url.Values{},
			wantBody:   `{"scope_key":"prod/checkout","mode":"enforce","items":{"allowed_egress":["db:5432"]}}`,
		},
		{
			name: "list guardrail violations",
			call: func(c *Client) (any, error) {
				return c.ListGuardrailViolations(ctx, ListGuardrailViolationsParams{
					WindowHours: intPtr(24),
					Service:     strPtr("checkout"),
					Namespace:   strPtr("prod"),
					Status:      strPtr("open"),
				})
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/guardrails/violations",
			wantQuery: url.Values{
				"window_hours": {"24"},
				"service":      {"checkout"},
				"namespace":    {"prod"},
				"status":       {"open"},
			},
			response: `{"items":[{"scope_key":"prod/checkout","rule_key":"allowed_egress","status":"open"}]}`,
			want:     []GuardrailViolation{{ScopeKey: "prod/checkout", RuleKey: "allowed_egress", Status: "open"}},
		},
		{
			name: "get one guardrail violation",
			call: func(c *Client) (any, error) {
				return c.GetGuardrailViolation(ctx, "prod/checkout", "allowed_egress", "db:5432")
			},
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/guardrails/violations/evidence",
			wantQuery: url.Values{
				"scope_key": {"prod/checkout"},
				"rule_key":  {"allowed_egress"},
				"offending": {"db:5432"},
			},
			response: `{"scope_key":"prod/checkout","rule_key":"allowed_egress","offending":"db:5432"}`,
			want: &GuardrailViolationDetail{
				ScopeKey:  "prod/checkout",
				RuleKey:   "allowed_egress",
				Offending: "db:5432",
			},
		},
		{
			name: "accept a guardrail violation",
			call: func(c *Client) (any, error) {
				return nil, c.AcceptGuardrailViolation(ctx, ViolationActionRequest{ScopeKey: "prod/checkout", RuleKey: "allowed_egress", Offending: "db:5432"})
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/guardrails/violations/accept",
			wantQuery:  url.Values{},
			wantBody:   `{"scope_key":"prod/checkout","rule_key":"allowed_egress","offending":"db:5432"}`,
		},
		{
			name: "dismiss a guardrail violation",
			call: func(c *Client) (any, error) {
				return nil, c.DismissGuardrailViolation(ctx, ViolationActionRequest{ScopeKey: "prod/checkout", RuleKey: "allowed_egress", Offending: "db:5432"})
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/guardrails/violations/dismiss",
			wantQuery:  url.Values{},
			wantBody:   `{"scope_key":"prod/checkout","rule_key":"allowed_egress","offending":"db:5432"}`,
		},
		{
			name: "reopen a guardrail violation",
			call: func(c *Client) (any, error) {
				return nil, c.ReopenGuardrailViolation(ctx, ViolationActionRequest{ScopeKey: "prod/checkout", RuleKey: "allowed_egress", Offending: "db:5432"})
			},
			wantMethod: http.MethodPost,
			wantPath:   "/api/v1/guardrails/violations/reopen",
			wantQuery:  url.Values{},
			wantBody:   `{"scope_key":"prod/checkout","rule_key":"allowed_egress","offending":"db:5432"}`,
		},
		{
			name:       "get the catalog",
			call:       func(c *Client) (any, error) { return c.GetCatalog(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/catalog",
			wantQuery:  url.Values{},
			response:   `{"correlation_bonus":5,"tags":{"source":"builtin"}}`,
			want:       &Catalog{CorrelationBonus: 5, Tags: map[string]string{"source": "builtin"}},
		},
		{
			name:       "get system settings",
			call:       func(c *Client) (any, error) { return c.GetSystemSettings(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/system-settings",
			wantQuery:  url.Values{},
			response:   `{"retention":{"observation_retention_days":14}}`,
			want:       &SystemSettings{Retention: SystemRetentionSettings{ObservationRetentionDays: 14}},
		},
		{
			name: "update system settings",
			call: func(c *Client) (any, error) {
				return nil, c.UpdateSystemSettings(ctx, SystemSettings{
					Sampling:  SystemSamplingSettings{ExamplesPerTransaction: 3},
					Retention: SystemRetentionSettings{ObservationRetentionDays: 14},
					Findings:  SystemFindingsSettings{DefaultWindowHours: 24, MaxWindowHours: 168},
				})
			},
			wantMethod: http.MethodPut,
			wantPath:   "/api/v1/system-settings",
			wantQuery:  url.Values{},
			wantBody: `{
				"sampling":{"examples_per_transaction":3,"example_sample_interval_seconds":0},
				"retention":{"observation_retention_days":14},
				"findings":{"default_window_hours":24,"max_window_hours":168},
				"capacity":{"max_resident_transactions":0,"max_baseline_set_members":0},
				"writeback":{"flush_interval_seconds":0},
				"detection":{"auto_transaction_guardrail":false},
				"identity":{"transaction_identity_dimensions":null}
			}`,
		},
		{
			name:       "get storage health",
			call:       func(c *Client) (any, error) { return c.GetStorageHealth(ctx) },
			wantMethod: http.MethodGet,
			wantPath:   "/api/v1/storage-health",
			wantQuery:  url.Values{},
			response:   `{"status":"degraded","disk":{"used_percent":91.5}}`,
			want:       &StorageHealth{Status: "degraded", Disk: StorageDisk{UsedPercent: 91.5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, captured := newCapturingClient(t, tt.response)

			got, err := tt.call(client)
			require.NoError(t, err)

			assert.Equal(t, tt.wantMethod, captured.method)
			assert.Equal(t, tt.wantPath, captured.path)
			assert.Equal(t, tt.wantQuery, captured.query)
			if tt.wantBody == "" {
				assert.Empty(t, captured.body)
			} else {
				assert.JSONEq(t, tt.wantBody, captured.body)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClientRequestsAcceptJSONAndOmitContentTypeWithoutABody(t *testing.T) {
	var headers http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	require.NoError(t, client.DeleteTransaction(context.Background(), 7))
	assert.Equal(t, "application/json", headers.Get("Accept"))
	assert.Empty(t, headers.Get("Content-Type"))
}

func TestNewClientKeepsTheBaseURLPathPrefix(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		wantPath string
	}{
		{name: "no prefix", baseURL: "", wantPath: "/api/v1/services"},
		{name: "prefix", baseURL: "/insights", wantPath: "/insights/api/v1/services"},
		{name: "trailing slash", baseURL: "/insights/", wantPath: "/insights/api/v1/services"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var path string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				path = r.URL.Path
				_, _ = w.Write([]byte(`{"items":[]}`))
			}))
			defer server.Close()

			client, err := NewClient(server.URL + tt.baseURL)
			require.NoError(t, err)
			_, err = client.ListServices(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, path)
		})
	}
}

func TestNewClientWithHTTPClientFallsBackToTheDefaultClient(t *testing.T) {
	client, err := NewClientWithHTTPClient("http://insights:8080", nil)
	require.NoError(t, err)
	assert.Same(t, http.DefaultClient, client.httpClient)
}
