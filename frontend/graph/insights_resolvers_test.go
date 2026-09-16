package graph

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The resolver defaults an omitted scope with a bare "service" string that has
// no compile-time link to the enum the UI sends, so the two have to be checked
// against each other. Deleting under the wrong scope leaves the guardrail in
// place and the UI reporting success.
func TestDeleteInsightsGuardrailTreatsAnOmittedScopeLikeTheServiceScope(t *testing.T) {
	stub := newInsightsEngineStub(t).accept(http.MethodDelete, "/api/v1/guardrails")
	resolver := newInsightsResolver(t, stub)
	serviceScope := model.InsightsPolicyScopeService

	omitted, err := resolver.Mutation().DeleteInsightsGuardrail(context.Background(), "prod/checkout", nil)
	require.NoError(t, err)
	assert.True(t, omitted)

	explicit, err := resolver.Mutation().DeleteInsightsGuardrail(context.Background(), "prod/checkout", &serviceScope)
	require.NoError(t, err)
	assert.True(t, explicit)

	calls := stub.recorded()
	require.Len(t, calls, 2)
	assert.Equal(t, calls[1].query, calls[0].query,
		"an omitted scope has to produce the same request as an explicit service scope")
	// The engine reads scope as a query parameter, so the wire value is a
	// contract and not an internal detail.
	assert.Equal(t, url.Values{"scope": {"service"}, "scope_key": {"prod/checkout"}}, calls[0].query)
}

// A transaction guardrail's scope key is a numeric transaction id rather than
// namespace/service, so the scope has to travel with it.
func TestDeleteInsightsGuardrailForwardsATransactionScope(t *testing.T) {
	stub := newInsightsEngineStub(t).accept(http.MethodDelete, "/api/v1/guardrails")
	resolver := newInsightsResolver(t, stub)
	transactionScope := model.InsightsPolicyScopeTransaction

	deleted, err := resolver.Mutation().DeleteInsightsGuardrail(context.Background(), "42", &transactionScope)
	require.NoError(t, err)
	assert.True(t, deleted)

	call := stub.only()
	assert.Equal(t, http.MethodDelete, call.method)
	assert.Equal(t, url.Values{"scope": {"transaction"}, "scope_key": {"42"}}, call.query)
}

// The upsert is a write followed by a read-back, and a transaction guardrail
// shares its key space with a service guardrail ("42" is both a transaction id
// and a plausible scope key). The read-back therefore has to match on scope as
// well, or the mutation answers with a different guardrail than it just wrote.
func TestUpsertInsightsGuardrailReadsBackTheGuardrailMatchingBothScopeAndKey(t *testing.T) {
	stub := newInsightsEngineStub(t).
		accept(http.MethodPut, "/api/v1/guardrails").
		respond(http.MethodGet, "/api/v1/guardrails", `{"items":[
			{"scope":"service","scope_key":"42","rules":[{"key":"allowed_egress","label":"Egress","mode":"enforce","allowlist":["db:5432"]}]},
			{"scope":"transaction","scope_key":"42","rules":[{"key":"attribute_correlation","label":"Attribute correlation","mode":"enforce","correlations":[
				{"name":"principal-matches-account","left":{"service":"edge-gateway","span":"resolvePrincipal","attr":"return.value","extract":"(\\d+)"},"right":{"service":"account-service","span":"getAccount","attr":"arg.0"},"relation":"equals","severity":"critical","why":"principal must match account"}
			]}]}
		]}`)
	resolver := newInsightsResolver(t, stub)

	extract := `(\d+)`
	stored, err := resolver.Mutation().UpsertInsightsGuardrail(context.Background(), model.InsightsGuardrailInput{
		Scope:    model.InsightsPolicyScopeTransaction,
		ScopeKey: "42",
		Rules: []*model.InsightsGuardrailRuleInput{{
			Key:   "attribute_correlation",
			Label: "Attribute correlation",
			Mode:  model.InsightsRuleModeEnforce,
			Correlations: []*model.InsightsCorrelationSpecInput{{
				Name: "principal-matches-account",
				Left: &model.InsightsCorrelationSelectorInput{
					Service: "edge-gateway",
					Span:    "resolvePrincipal",
					Attr:    "return.value",
					Extract: &extract,
				},
				Right: &model.InsightsCorrelationSelectorInput{
					Service: "account-service",
					Span:    "getAccount",
					Attr:    "arg.0",
				},
				Relation: model.InsightsCorrelationRelationEquals,
			}},
		}},
	})
	require.NoError(t, err)

	require.NotNil(t, stored)
	assert.Equal(t, model.InsightsPolicyScopeTransaction, stored.Scope)
	require.Len(t, stored.Rules, 1)
	require.Len(t, stored.Rules[0].Correlations, 1)
	correlation := stored.Rules[0].Correlations[0]
	assert.Equal(t, "principal-matches-account", correlation.Name)
	assert.Equal(t, model.InsightsCorrelationRelationEquals, correlation.Relation)
	require.NotNil(t, correlation.Left)
	assert.Equal(t, "resolvePrincipal", correlation.Left.Span)
	require.NotNil(t, correlation.Left.Extract)
	assert.Equal(t, extract, *correlation.Left.Extract)
	require.NotNil(t, correlation.Right)
	assert.Equal(t, "getAccount", correlation.Right.Span)

	calls := stub.recorded()
	require.Len(t, calls, 2)
	assert.Equal(t, http.MethodPut, calls[0].method)
	assert.JSONEq(t, `{"scope":"transaction","scope_key":"42","rules":[{"key":"attribute_correlation","label":"Attribute correlation","mode":"enforce","correlations":[
		{"name":"principal-matches-account","left":{"service":"edge-gateway","span":"resolvePrincipal","attr":"return.value","extract":"(\\d+)"},"right":{"service":"account-service","span":"getAccount","attr":"arg.0"},"relation":"equals","severity":"critical"}
	]}]}`, calls[0].body)
}

// insightsLookup is one single-entity resolver, driven through a stub that
// answers its endpoint with a chosen status.
type insightsLookup struct {
	name   string
	method string
	path   string
	// extraRoutes are endpoints the client fans out to on success; they have to
	// be routed or the stub reports an unexpected request.
	extraRoutes []insightsEngineCall
	body        string
	call        func(*Resolver) (any, error)
}

func insightsNullableLookups() []insightsLookup {
	obj := &model.Insights{}
	ctx := context.Background()
	return []insightsLookup{
		{
			name:   "transaction",
			method: http.MethodGet,
			path:   "/api/v1/transactions/42",
			body:   `{"id":42,"service":"checkout","namespace":"prod"}`,
			call: func(r *Resolver) (any, error) {
				return r.Insights().Transaction(ctx, obj, "42")
			},
		},
		{
			name:   "observation",
			method: http.MethodGet,
			path:   "/api/v1/transactions/42/observations/trace-1",
			body:   `{"trace_id":"trace-1","transaction_id":42}`,
			call: func(r *Resolver) (any, error) {
				return r.Insights().Observation(ctx, obj, "42", "trace-1")
			},
		},
		{
			name:        "anomaly",
			method:      http.MethodGet,
			path:        "/api/v1/anomalies/42/sig-1",
			extraRoutes: []insightsEngineCall{{method: http.MethodGet, path: "/api/v1/transactions/42/observations"}},
			body:        `{"transaction_id":42,"signature":"sig-1","status":"open"}`,
			call: func(r *Resolver) (any, error) {
				return r.Insights().Anomaly(ctx, obj, "42", "sig-1")
			},
		},
		{
			name:   "guardrail violation",
			method: http.MethodGet,
			path:   "/api/v1/guardrails/violations/evidence",
			body:   `{"scope_key":"prod/checkout","rule_key":"allowed_egress"}`,
			call: func(r *Resolver) (any, error) {
				return r.Insights().GuardrailViolation(ctx, obj, "prod/checkout", "allowed_egress", "db:5432")
			},
		},
	}
}

// These four fields are nullable in the schema precisely so the UI can render
// "this no longer exists" instead of an error banner, which matters because
// insights entities are evicted as transactions age out.
func TestInsightsNullableLookupsReturnNullForAnUnknownEntity(t *testing.T) {
	for _, lookup := range insightsNullableLookups() {
		t.Run(lookup.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).fail(lookup.method, lookup.path, http.StatusNotFound, insights.ErrorCodeNotFound)

			got, err := lookup.call(newInsightsResolver(t, stub))
			require.NoError(t, err)
			assert.Nil(t, got)
		})
	}
}

// The null-for-missing behaviour must not widen into swallowing every failure,
// or an insights outage looks like an empty product area.
func TestInsightsNullableLookupsStillReportANonMissingFailure(t *testing.T) {
	for _, lookup := range insightsNullableLookups() {
		t.Run(lookup.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).fail(lookup.method, lookup.path, http.StatusInternalServerError, insights.ErrorCodeInternal)

			_, err := lookup.call(newInsightsResolver(t, stub))
			require.Error(t, err)
			requireInsightsErrorCode(t, err, insights.CodeInternal)
		})
	}
}

func TestInsightsNullableLookupsReturnTheEntityWhenItExists(t *testing.T) {
	for _, lookup := range insightsNullableLookups() {
		t.Run(lookup.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).respond(lookup.method, lookup.path, lookup.body)
			for _, extra := range lookup.extraRoutes {
				stub.respond(extra.method, extra.path, `{"items":[]}`)
			}

			got, err := lookup.call(newInsightsResolver(t, stub))
			require.NoError(t, err)
			assert.NotNil(t, got)
		})
	}
}

// Everything that is not one of the four nullable lookups has to surface a
// missing resource as an error carrying INSIGHTS_NOT_FOUND. Returning an empty
// value instead would render, for example, a storage-health panel full of
// zeroes as if it had been measured.
func TestInsightsNonNullableReadsReportAMissingResourceAsAnError(t *testing.T) {
	obj := &model.Insights{}
	ctx := context.Background()

	tests := []insightsLookup{
		{name: "services", method: http.MethodGet, path: "/api/v1/services",
			call: func(r *Resolver) (any, error) { return r.Insights().Services(ctx, obj) }},
		{name: "service names", method: http.MethodGet, path: "/api/v1/service-names",
			call: func(r *Resolver) (any, error) { return r.Insights().ServiceNames(ctx, obj) }},
		{name: "service profile", method: http.MethodGet, path: "/api/v1/services/profile",
			call: func(r *Resolver) (any, error) { return r.Insights().ServiceProfile(ctx, obj, nil, "checkout") }},
		{name: "blast radius", method: http.MethodGet, path: "/api/v1/services/blast-radius",
			call: func(r *Resolver) (any, error) { return r.Insights().BlastRadius(ctx, obj, nil, "checkout", nil) }},
		{name: "transactions", method: http.MethodGet, path: "/api/v1/transactions",
			call: func(r *Resolver) (any, error) { return r.Insights().Transactions(ctx, obj, nil, nil, nil, nil) }},
		{name: "baseline", method: http.MethodGet, path: "/api/v1/transactions/42/baseline",
			call: func(r *Resolver) (any, error) { return r.Insights().Baseline(ctx, obj, "42") }},
		{name: "observations", method: http.MethodGet, path: "/api/v1/transactions/42/observations",
			call: func(r *Resolver) (any, error) { return r.Insights().Observations(ctx, obj, "42", nil) }},
		{name: "findings", method: http.MethodGet, path: "/api/v1/findings",
			call: func(r *Resolver) (any, error) { return r.Insights().Findings(ctx, obj, nil, nil, nil, nil, nil) }},
		{name: "anomalies", method: http.MethodGet, path: "/api/v1/anomalies",
			call: func(r *Resolver) (any, error) { return r.Insights().Anomalies(ctx, obj, nil, nil, nil, nil) }},
		{name: "policies", method: http.MethodGet, path: "/api/v1/policies",
			call: func(r *Resolver) (any, error) { return r.Insights().Policies(ctx, obj) }},
		{name: "learning policies", method: http.MethodGet, path: "/api/v1/learning-policies",
			call: func(r *Resolver) (any, error) { return r.Insights().LearningPolicies(ctx, obj) }},
		{name: "guardrails", method: http.MethodGet, path: "/api/v1/guardrails",
			call: func(r *Resolver) (any, error) { return r.Insights().Guardrails(ctx, obj) }},
		{name: "guardrail violations", method: http.MethodGet, path: "/api/v1/guardrails/violations",
			call: func(r *Resolver) (any, error) { return r.Insights().GuardrailViolations(ctx, obj, nil, nil, nil, nil) }},
		{name: "catalog", method: http.MethodGet, path: "/api/v1/catalog",
			call: func(r *Resolver) (any, error) { return r.Insights().Catalog(ctx, obj) }},
		{name: "system settings", method: http.MethodGet, path: "/api/v1/system-settings",
			call: func(r *Resolver) (any, error) { return r.Insights().SystemSettings(ctx, obj) }},
		{name: "storage health", method: http.MethodGet, path: "/api/v1/storage-health",
			call: func(r *Resolver) (any, error) { return r.Insights().StorageHealth(ctx, obj) }},
	}
	require.Len(t, tests, 16, "every non-nullable insights read belongs in this table")

	for _, lookup := range tests {
		t.Run(lookup.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).fail(lookup.method, lookup.path, http.StatusNotFound, insights.ErrorCodeNotFound)

			_, err := lookup.call(newInsightsResolver(t, stub))
			require.Error(t, err)
			requireInsightsErrorCode(t, err, insights.CodeNotFound)
		})
	}
}

// Transaction ids arrive as GraphQL strings and are parsed before use. A bad id
// has to be rejected locally: reporting it as INSIGHTS_NOT_FOUND (or worse,
// as the null an unknown transaction produces) hides a caller bug, and issuing
// the request anyway would aim a mutation at an unintended URL.
func TestInsightsResolversRejectANonNumericTransactionIDWithoutCallingTheEngine(t *testing.T) {
	obj := &model.Insights{}
	ctx := context.Background()
	const badID = "not-a-number"

	tests := []struct {
		name string
		call func(*Resolver) error
	}{
		{"transaction", func(r *Resolver) error {
			_, err := r.Insights().Transaction(ctx, obj, badID)
			return err
		}},
		{"baseline", func(r *Resolver) error {
			_, err := r.Insights().Baseline(ctx, obj, badID)
			return err
		}},
		{"observations", func(r *Resolver) error {
			_, err := r.Insights().Observations(ctx, obj, badID, nil)
			return err
		}},
		{"observation", func(r *Resolver) error {
			_, err := r.Insights().Observation(ctx, obj, badID, "trace-1")
			return err
		}},
		{"anomaly", func(r *Resolver) error {
			_, err := r.Insights().Anomaly(ctx, obj, badID, "sig-1")
			return err
		}},
		{"promote baseline class", func(r *Resolver) error {
			_, err := r.Mutation().PromoteInsightsBaselineClass(ctx, badID, model.InsightsDeviationClass("D2_egress"))
			return err
		}},
		{"reset baseline class", func(r *Resolver) error {
			_, err := r.Mutation().ResetInsightsBaselineClass(ctx, badID, model.InsightsDeviationClass("D2_egress"))
			return err
		}},
		{"reset transaction baselines", func(r *Resolver) error {
			_, err := r.Mutation().ResetInsightsTransactionBaselines(ctx, badID)
			return err
		}},
		{"promote transaction baselines", func(r *Resolver) error {
			_, err := r.Mutation().PromoteInsightsTransactionBaselines(ctx, badID)
			return err
		}},
		{"delete transaction", func(r *Resolver) error {
			_, err := r.Mutation().DeleteInsightsTransaction(ctx, badID)
			return err
		}},
		{"resolve anomaly", func(r *Resolver) error {
			_, err := r.Mutation().ResolveInsightsAnomaly(ctx, badID, "sig-1", model.InsightsAnomalyResolution("accepted"))
			return err
		}},
		{"bulk promote transactions", func(r *Resolver) error {
			_, err := r.Mutation().BulkPromoteInsightsTransactions(ctx, []string{"1", badID})
			return err
		}},
		{"bulk delete transactions", func(r *Resolver) error {
			_, err := r.Mutation().BulkDeleteInsightsTransactions(ctx, []string{"1", badID})
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t)

			err := tt.call(newInsightsResolver(t, stub))
			require.Error(t, err)
			require.ErrorIs(t, err, insights.ErrBadRequest)
			requireInsightsErrorCode(t, err, insights.CodeBadRequest)
			assert.Empty(t, stub.recorded(), "a malformed id must be rejected before any request")
		})
	}
}

// The bulk mutations parse the whole batch up front. Skipping the bad ids and
// sending the rest would half-apply a destructive action while telling the user
// it succeeded.
func TestBulkInsightsTransactionMutationsSendEveryParsedID(t *testing.T) {
	tests := []struct {
		name string
		path string
		body string
		call func(*Resolver) error
	}{
		{
			name: "bulk promote",
			path: "/api/v1/transactions/promote",
			body: `{"promoted":2}`,
			call: func(r *Resolver) error {
				_, err := r.Mutation().BulkPromoteInsightsTransactions(context.Background(), []string{"7", "42"})
				return err
			},
		},
		{
			name: "bulk delete",
			path: "/api/v1/transactions",
			body: `{"deleted":2}`,
			call: func(r *Resolver) error {
				_, err := r.Mutation().BulkDeleteInsightsTransactions(context.Background(), []string{"7", "42"})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).respond(http.MethodPost, tt.path, tt.body)

			require.NoError(t, tt.call(newInsightsResolver(t, stub)))
			assert.JSONEq(t, `{"transaction_ids":[7,42]}`, stub.only().body)
		})
	}
}

// The GraphQL system settings input has no findings field at all, so the
// resolver re-reads the stored settings and carries the findings block over.
// Dropping that would silently reset every user's finding window on the next
// settings save.
func TestUpdateInsightsSystemSettingsPreservesTheStoredFindingsSettings(t *testing.T) {
	stub := newInsightsEngineStub(t).
		respond(http.MethodGet, "/api/v1/system-settings", `{
			"sampling":{"examples_per_transaction":1,"example_sample_interval_seconds":2},
			"retention":{"observation_retention_days":3},
			"findings":{"default_window_hours":24,"max_window_hours":168},
			"capacity":{"max_resident_transactions":4,"max_baseline_set_members":5},
			"writeback":{"flush_interval_seconds":6},
			"detection":{"auto_transaction_guardrail":true},
			"identity":{"transaction_identity_dimensions":[]}
		}`).
		accept(http.MethodPut, "/api/v1/system-settings")
	resolver := newInsightsResolver(t, stub)

	_, err := resolver.Mutation().UpdateInsightsSystemSettings(context.Background(), model.InsightsSystemSettingsInput{
		Sampling:  &model.InsightsSystemSamplingSettingsInput{ExamplesPerTransaction: 9, ExampleSampleIntervalSeconds: 8},
		Retention: &model.InsightsSystemRetentionSettingsInput{ObservationRetentionDays: 7},
		Capacity:  &model.InsightsSystemCapacitySettingsInput{MaxResidentTransactions: 6, MaxBaselineSetMembers: 5},
		Writeback: &model.InsightsSystemWritebackSettingsInput{FlushIntervalSeconds: 4},
		Detection: &model.InsightsSystemDetectionSettingsInput{AutoTransactionGuardrail: false},
		Identity:  &model.InsightsSystemIdentitySettingsInput{},
	})
	// The stubbed write answers 204, so the read-back sees the same stored
	// settings; only the request the resolver sent is under test here.
	require.NoError(t, err)

	calls := stub.recorded()
	require.Len(t, calls, 3)
	written := calls[1]
	require.Equal(t, http.MethodPut, written.method)
	assert.JSONEq(t, `{
		"sampling":{"examples_per_transaction":9,"example_sample_interval_seconds":8},
		"retention":{"observation_retention_days":7},
		"findings":{"default_window_hours":24,"max_window_hours":168},
		"capacity":{"max_resident_transactions":6,"max_baseline_set_members":5},
		"writeback":{"flush_interval_seconds":4},
		"detection":{"auto_transaction_guardrail":false},
		"identity":{"transaction_identity_dimensions":[]}
	}`, written.body)
}

// Writing settings assembled from a failed read would persist a zeroed findings
// block, so the read failure has to abort the mutation.
func TestUpdateInsightsSystemSettingsDoesNotWriteWhenTheStoredSettingsCannotBeRead(t *testing.T) {
	stub := newInsightsEngineStub(t).
		fail(http.MethodGet, "/api/v1/system-settings", http.StatusServiceUnavailable, insights.ErrorCodeUnavailable)
	resolver := newInsightsResolver(t, stub)

	_, err := resolver.Mutation().UpdateInsightsSystemSettings(context.Background(), model.InsightsSystemSettingsInput{
		Sampling:  &model.InsightsSystemSamplingSettingsInput{},
		Retention: &model.InsightsSystemRetentionSettingsInput{},
		Capacity:  &model.InsightsSystemCapacitySettingsInput{},
		Writeback: &model.InsightsSystemWritebackSettingsInput{},
		Detection: &model.InsightsSystemDetectionSettingsInput{},
		Identity:  &model.InsightsSystemIdentitySettingsInput{},
	})
	require.Error(t, err)
	requireInsightsErrorCode(t, err, insights.CodeUnavailable)

	for _, call := range stub.recorded() {
		assert.Equal(t, http.MethodGet, call.method, "no write may be issued after the read failed")
	}
}

// An incomplete settings input is the caller's fault and has to be rejected
// before anything is written, but the resolver reads the current settings
// first, so the rejection order is worth pinning.
func TestUpdateInsightsSystemSettingsRejectsAnIncompleteInputWithoutWriting(t *testing.T) {
	stub := newInsightsEngineStub(t).
		respond(http.MethodGet, "/api/v1/system-settings", `{"findings":{"default_window_hours":24,"max_window_hours":168}}`)
	resolver := newInsightsResolver(t, stub)

	_, err := resolver.Mutation().UpdateInsightsSystemSettings(context.Background(), model.InsightsSystemSettingsInput{
		Sampling: &model.InsightsSystemSamplingSettingsInput{},
	})
	require.Error(t, err)
	requireInsightsErrorCode(t, err, insights.CodeBadRequest)

	for _, call := range stub.recorded() {
		assert.Equal(t, http.MethodGet, call.method, "an invalid input must not reach the write endpoint")
	}
}

// serviceNames is a non-null list of non-null strings in the schema. gqlgen
// serializes a nil slice as null, which the UI rejects as a schema violation,
// so the resolver normalizes it.
func TestInsightsServiceNamesIsAnEmptyListRatherThanNull(t *testing.T) {
	stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/service-names", `{"items":null}`)

	names, err := newInsightsResolver(t, stub).Insights().ServiceNames(context.Background(), &model.Insights{})
	require.NoError(t, err)
	require.NotNil(t, names)
	assert.Empty(t, names)
}

// An absent namespace means "any namespace" and has to be left out of the
// query rather than sent as an empty value, which the engine would read as a
// filter matching nothing.
func TestInsightsServiceLookupsOmitAnAbsentNamespaceAndSendAPresentOne(t *testing.T) {
	obj := &model.Insights{}
	ctx := context.Background()

	tests := []struct {
		name string
		path string
		body string
		call func(*Resolver, *string) error
	}{
		{
			name: "service profile",
			path: "/api/v1/services/profile",
			body: `{"service":"checkout"}`,
			call: func(r *Resolver, namespace *string) error {
				_, err := r.Insights().ServiceProfile(ctx, obj, namespace, "checkout")
				return err
			},
		},
		{
			name: "blast radius",
			path: "/api/v1/services/blast-radius",
			body: `{"service":"checkout"}`,
			call: func(r *Resolver, namespace *string) error {
				_, err := r.Insights().BlastRadius(ctx, obj, namespace, "checkout", nil)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("absent", func(t *testing.T) {
				stub := newInsightsEngineStub(t).respond(http.MethodGet, tt.path, tt.body)
				require.NoError(t, tt.call(newInsightsResolver(t, stub), nil))
				assert.Equal(t, url.Values{"service": {"checkout"}}, stub.only().query)
			})

			t.Run("present", func(t *testing.T) {
				stub := newInsightsEngineStub(t).respond(http.MethodGet, tt.path, tt.body)
				require.NoError(t, tt.call(newInsightsResolver(t, stub), insightsStrPtr("prod")))
				assert.Equal(t, url.Values{"namespace": {"prod"}, "service": {"checkout"}}, stub.only().query)
			})
		})
	}
}

// Every list resolver copies its GraphQL arguments into a params struct field
// by field. With several same-typed neighbours a fully populated call cannot
// tell a crosswired assignment from a correct one, so each filter is set alone
// and has to arrive alone, under its own query key.
func TestInsightsListResolversForwardEachFilterOnItsOwn(t *testing.T) {
	obj := &model.Insights{}
	ctx := context.Background()
	kind := model.InsightsTransactionKindConsumer
	findingKind := model.InsightsFindingKindViolation

	tests := []struct {
		name      string
		path      string
		call      func(*Resolver) error
		wantQuery url.Values
	}{
		{
			name: "transactions without filters",
			path: "/api/v1/transactions",
			call: func(r *Resolver) error {
				_, err := r.Insights().Transactions(ctx, obj, nil, nil, nil, nil)
				return err
			},
			wantQuery: url.Values{},
		},
		{
			name: "transactions namespace only",
			path: "/api/v1/transactions",
			call: func(r *Resolver) error {
				_, err := r.Insights().Transactions(ctx, obj, insightsStrPtr("prod"), nil, nil, nil)
				return err
			},
			wantQuery: url.Values{"namespace": {"prod"}},
		},
		{
			name: "transactions service only",
			path: "/api/v1/transactions",
			call: func(r *Resolver) error {
				_, err := r.Insights().Transactions(ctx, obj, nil, insightsStrPtr("checkout"), nil, nil)
				return err
			},
			wantQuery: url.Values{"service": {"checkout"}},
		},
		{
			name: "transactions kind only",
			path: "/api/v1/transactions",
			call: func(r *Resolver) error {
				_, err := r.Insights().Transactions(ctx, obj, nil, nil, &kind, nil)
				return err
			},
			wantQuery: url.Values{"kind": {"CONSUMER"}},
		},
		{
			name: "transactions window only",
			path: "/api/v1/transactions",
			call: func(r *Resolver) error {
				_, err := r.Insights().Transactions(ctx, obj, nil, nil, nil, insightsIntPtr(3))
				return err
			},
			wantQuery: url.Values{"window_hours": {"3"}},
		},
		{
			name: "findings window only",
			path: "/api/v1/findings",
			call: func(r *Resolver) error {
				_, err := r.Insights().Findings(ctx, obj, insightsIntPtr(3), nil, nil, nil, nil)
				return err
			},
			wantQuery: url.Values{"window_hours": {"3"}},
		},
		{
			name: "findings service only",
			path: "/api/v1/findings",
			call: func(r *Resolver) error {
				_, err := r.Insights().Findings(ctx, obj, nil, insightsStrPtr("checkout"), nil, nil, nil)
				return err
			},
			wantQuery: url.Values{"service": {"checkout"}},
		},
		{
			name: "findings namespace only",
			path: "/api/v1/findings",
			call: func(r *Resolver) error {
				_, err := r.Insights().Findings(ctx, obj, nil, nil, insightsStrPtr("prod"), nil, nil)
				return err
			},
			wantQuery: url.Values{"namespace": {"prod"}},
		},
		{
			name: "findings status only",
			path: "/api/v1/findings",
			call: func(r *Resolver) error {
				_, err := r.Insights().Findings(ctx, obj, nil, nil, nil, insightsStrPtr("open"), nil)
				return err
			},
			wantQuery: url.Values{"status": {"open"}},
		},
		{
			name: "findings kind only",
			path: "/api/v1/findings",
			call: func(r *Resolver) error {
				_, err := r.Insights().Findings(ctx, obj, nil, nil, nil, nil, &findingKind)
				return err
			},
			wantQuery: url.Values{"kind": {"violation"}},
		},
		{
			name: "anomalies window only",
			path: "/api/v1/anomalies",
			call: func(r *Resolver) error {
				_, err := r.Insights().Anomalies(ctx, obj, insightsIntPtr(3), nil, nil, nil)
				return err
			},
			wantQuery: url.Values{"window_hours": {"3"}},
		},
		{
			name: "anomalies service only",
			path: "/api/v1/anomalies",
			call: func(r *Resolver) error {
				_, err := r.Insights().Anomalies(ctx, obj, nil, insightsStrPtr("checkout"), nil, nil)
				return err
			},
			wantQuery: url.Values{"service": {"checkout"}},
		},
		{
			name: "anomalies namespace only",
			path: "/api/v1/anomalies",
			call: func(r *Resolver) error {
				_, err := r.Insights().Anomalies(ctx, obj, nil, nil, insightsStrPtr("prod"), nil)
				return err
			},
			wantQuery: url.Values{"namespace": {"prod"}},
		},
		{
			name: "anomalies status only",
			path: "/api/v1/anomalies",
			call: func(r *Resolver) error {
				_, err := r.Insights().Anomalies(ctx, obj, nil, nil, nil, insightsStrPtr("open"))
				return err
			},
			wantQuery: url.Values{"status": {"open"}},
		},
		{
			name: "guardrail violations window only",
			path: "/api/v1/guardrails/violations",
			call: func(r *Resolver) error {
				_, err := r.Insights().GuardrailViolations(ctx, obj, insightsIntPtr(3), nil, nil, nil)
				return err
			},
			wantQuery: url.Values{"window_hours": {"3"}},
		},
		{
			name: "guardrail violations service only",
			path: "/api/v1/guardrails/violations",
			call: func(r *Resolver) error {
				_, err := r.Insights().GuardrailViolations(ctx, obj, nil, insightsStrPtr("checkout"), nil, nil)
				return err
			},
			wantQuery: url.Values{"service": {"checkout"}},
		},
		{
			name: "guardrail violations namespace only",
			path: "/api/v1/guardrails/violations",
			call: func(r *Resolver) error {
				_, err := r.Insights().GuardrailViolations(ctx, obj, nil, nil, insightsStrPtr("prod"), nil)
				return err
			},
			wantQuery: url.Values{"namespace": {"prod"}},
		},
		{
			name: "guardrail violations status only",
			path: "/api/v1/guardrails/violations",
			call: func(r *Resolver) error {
				_, err := r.Insights().GuardrailViolations(ctx, obj, nil, nil, nil, insightsStrPtr("open"))
				return err
			},
			wantQuery: url.Values{"status": {"open"}},
		},
		{
			name: "observations sample reason",
			path: "/api/v1/transactions/42/observations",
			call: func(r *Resolver) error {
				reason := model.InsightsSampleReasonExample
				_, err := r.Insights().Observations(ctx, obj, "42", &reason)
				return err
			},
			wantQuery: url.Values{"sample_reason": {"example"}},
		},
		{
			name: "observations without a sample reason",
			path: "/api/v1/transactions/42/observations",
			call: func(r *Resolver) error {
				_, err := r.Insights().Observations(ctx, obj, "42", nil)
				return err
			},
			wantQuery: url.Values{},
		},
		{
			name: "blast radius depth",
			path: "/api/v1/services/blast-radius",
			call: func(r *Resolver) error {
				_, err := r.Insights().BlastRadius(ctx, obj, nil, "checkout", insightsIntPtr(2))
				return err
			},
			wantQuery: url.Values{"service": {"checkout"}, "depth": {"2"}},
		},
		{
			// Three same-typed arguments in a row, so a swapped pair looks up
			// evidence for a rule that did not fire.
			name: "guardrail violation evidence",
			path: "/api/v1/guardrails/violations/evidence",
			call: func(r *Resolver) error {
				_, err := r.Insights().GuardrailViolation(ctx, obj, "prod/checkout", "allowed_egress", "db:5432")
				return err
			},
			wantQuery: url.Values{
				"scope_key": {"prod/checkout"},
				"rule_key":  {"allowed_egress"},
				"offending": {"db:5432"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).respond(http.MethodGet, tt.path, `{"items":[]}`)

			require.NoError(t, tt.call(newInsightsResolver(t, stub)))
			assert.Equal(t, tt.wantQuery, stub.only().query)
		})
	}
}

// Each read resolver hands the engine payload to a converter and returns the
// result. An error path produces an empty list or a zeroed struct, which for a
// dashboard is indistinguishable from "nothing to report", so the populated
// case has to be driven too.
func TestInsightsReadsReturnTheConvertedEnginePayload(t *testing.T) {
	obj := &model.Insights{}
	ctx := context.Background()

	t.Run("services", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/services",
			`{"items":[{"service":"checkout","namespace":"prod","risk_score":42}]}`)

		stats, err := newInsightsResolver(t, stub).Insights().Services(ctx, obj)
		require.NoError(t, err)
		require.Len(t, stats, 1)
		assert.Equal(t, "checkout", stats[0].Service)
		assert.Equal(t, "prod", stats[0].Namespace)
	})

	t.Run("service names", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/service-names",
			`{"items":["checkout","payments"]}`)

		names, err := newInsightsResolver(t, stub).Insights().ServiceNames(ctx, obj)
		require.NoError(t, err)
		assert.Equal(t, []string{"checkout", "payments"}, names)
	})

	t.Run("baseline", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/transactions/42/baseline",
			`{"items":[{"transaction_id":42,"class":"D2_egress","class_label":"Egress","data":{"peers":["db:5432"]}}]}`)

		baseline, err := newInsightsResolver(t, stub).Insights().Baseline(ctx, obj, "42")
		require.NoError(t, err)
		require.Len(t, baseline, 1)
		assert.Equal(t, "42", baseline[0].TransactionID)
		assert.Equal(t, model.InsightsDeviationClassD2Egress, baseline[0].Class)
		require.NotNil(t, baseline[0].Data)
		assert.JSONEq(t, `{"peers":["db:5432"]}`, *baseline[0].Data)
	})

	t.Run("findings carry the rendered summary", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/findings",
			`{"items":[{"kind":"anomaly","service":"checkout","namespace":"prod","title":"Unexpected egress","summary":"first call to db:5432"}]}`)

		findings, err := newInsightsResolver(t, stub).Insights().Findings(ctx, obj, nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.Len(t, findings, 1)
		assert.Equal(t, "Unexpected egress", findings[0].Title)
		assert.Equal(t, "first call to db:5432", findings[0].Summary)
	})

	t.Run("policies", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/policies",
			`{"items":[{"id":5,"name":"checkout","enabled":true,"fire_at_score":70,"signal_weights":{"egress":3},"scope":"service","scope_key":"prod/checkout"}]}`)

		policies, err := newInsightsResolver(t, stub).Insights().Policies(ctx, obj)
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, "5", policies[0].ID)
		require.NotNil(t, policies[0].SignalWeights)
		assert.JSONEq(t, `{"egress":3}`, *policies[0].SignalWeights)
	})

	t.Run("learning policies", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/learning-policies",
			`{"items":[{"class":"D2_egress","mode":"all","scope":"service","scope_key":"prod/checkout"}]}`)

		policies, err := newInsightsResolver(t, stub).Insights().LearningPolicies(ctx, obj)
		require.NoError(t, err)
		require.Len(t, policies, 1)
		assert.Equal(t, model.InsightsLearningModeAll, policies[0].Mode)
	})

	t.Run("guardrails", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/guardrails",
			`{"items":[{"scope":"service","scope_key":"prod/checkout","rules":[{"key":"allowed_egress","label":"Egress","mode":"enforce"}]}]}`)

		guardrails, err := newInsightsResolver(t, stub).Insights().Guardrails(ctx, obj)
		require.NoError(t, err)
		require.Len(t, guardrails, 1)
		require.Len(t, guardrails[0].Rules, 1)
		assert.Equal(t, "allowed_egress", guardrails[0].Rules[0].Key)
	})

	t.Run("catalog", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/catalog",
			`{"correlation_bonus":5,"tags":{"D2_egress":"network"}}`)

		catalog, err := newInsightsResolver(t, stub).Insights().Catalog(ctx, obj)
		require.NoError(t, err)
		require.NotNil(t, catalog)
		assert.Equal(t, 5, catalog.CorrelationBonus)
		assert.JSONEq(t, `{"D2_egress":"network"}`, catalog.Tags)
	})

	t.Run("system settings", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/system-settings", insightsStoredSystemSettings)

		settings, err := newInsightsResolver(t, stub).Insights().SystemSettings(ctx, obj)
		require.NoError(t, err)
		require.NotNil(t, settings)
		require.NotNil(t, settings.Sampling)
		assert.Equal(t, 1, settings.Sampling.ExamplesPerTransaction)
	})

	t.Run("storage health", func(t *testing.T) {
		stub := newInsightsEngineStub(t).respond(http.MethodGet, "/api/v1/storage-health",
			`{"checked_at":"2026-09-16T10:00:00Z","status":"ok","disk":{"used_percent":12.5,"status":"ok"}}`)

		health, err := newInsightsResolver(t, stub).Insights().StorageHealth(ctx, obj)
		require.NoError(t, err)
		require.NotNil(t, health)
		assert.Equal(t, "2026-09-16T10:00:00Z", health.CheckedAt)
	})
}

// These mutations decode a JSON-encoded GraphQL string argument before doing
// anything. A malformed one is the caller's fault and has to be rejected
// locally; sending the partially decoded request would apply a guardrail seed
// or a bulk resolution the user never asked for.
func TestInsightsMutationsRejectAMalformedArgumentWithoutCallingTheEngine(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string
		call func(*Resolver) error
	}{
		{
			name: "guardrail seed items are not JSON",
			call: func(r *Resolver) error {
				_, err := r.Mutation().SeedInsightsGuardrail(ctx, model.InsightsGuardrailSeedInput{
					ScopeKey: "prod/checkout",
					Items:    "not json",
				})
				return err
			},
		},
		{
			name: "policy signal weights are not JSON",
			call: func(r *Resolver) error {
				_, err := r.Mutation().UpsertInsightsPolicy(ctx, model.InsightsPolicyInput{
					Name:          "checkout",
					Scope:         model.InsightsPolicyScopeService,
					ScopeKey:      "prod/checkout",
					SignalWeights: insightsStrPtr("not json"),
				})
				return err
			},
		},
		{
			name: "a bulk anomaly item has a non-numeric transaction id",
			call: func(r *Resolver) error {
				_, err := r.Mutation().BulkResolveInsightsAnomalies(ctx, model.InsightsBulkResolutionDismiss,
					[]*model.InsightsAnomalyRefInput{{TransactionID: "not-a-number", Signature: "sig-1"}})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t)

			err := tt.call(newInsightsResolver(t, stub))
			require.Error(t, err)
			require.ErrorIs(t, err, insights.ErrBadRequest)
			requireInsightsErrorCode(t, err, insights.CodeBadRequest)
			assert.Empty(t, stub.recorded())
		})
	}
}

// The engine answers requests with 503 while it warms up, which the UI retries
// instead of showing a failure. It has to stay distinguishable from a genuine
// unavailability at the resolver boundary.
func TestInsightsResolversTagTheEngineWarmUpDistinctlyFromAnOutage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		wantCode string
	}{
		{name: "warming up", message: "engine is starting", wantCode: insights.CodeEngineStarting},
		{name: "unavailable", message: "no backend available", wantCode: insights.CodeUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := newInsightsEngineStub(t).reply(http.MethodGet, "/api/v1/services", insightsEngineReply{
				status: http.StatusServiceUnavailable,
				body:   `{"error":{"code":"unavailable","message":"` + tt.message + `"}}`,
			})

			_, err := newInsightsResolver(t, stub).Insights().Services(context.Background(), &model.Insights{})
			require.Error(t, err)
			requireInsightsErrorCode(t, err, tt.wantCode)
		})
	}
}

// The root field gates once so a disabled feature produces one error rather
// than one per selected child field.
func TestInsightsRootFieldResolvesToAnEmptyContainerWhenEnabled(t *testing.T) {
	stub := newInsightsEngineStub(t)

	root, err := newInsightsResolver(t, stub).Query().Insights(context.Background())
	require.NoError(t, err)
	assert.Equal(t, &model.Insights{}, root)
	assert.Empty(t, stub.recorded(), "the root field must not query the engine on its own")
}
