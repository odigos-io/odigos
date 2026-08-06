package insights

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientEscapesPathAndQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/prefix/api/v1/anomalies/42/sig/with spaces", r.URL.Path)
		assert.Equal(t, "/prefix/api/v1/anomalies/42/sig%2Fwith%20spaces", r.URL.EscapedPath())
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"transaction_id": 42,
			"signature": "sig/with spaces",
			"service": "checkout",
			"namespace": "prod",
			"triggered_classes": [],
			"evidence": {},
			"occurrences": 1,
			"max_score": 10,
			"severity": "low",
			"status": "open"
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL + "/prefix")
	require.NoError(t, err)

	result, err := client.GetAnomaly(context.Background(), 42, "sig/with spaces")
	require.NoError(t, err)
	assert.Equal(t, "sig/with spaces", result.Signature)
}

func TestClientEncodesQueryParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "team/a + b", r.URL.Query().Get("namespace"))
		assert.Equal(t, "checkout/api", r.URL.Query().Get("service"))
		assert.Equal(t, "HTTP", r.URL.Query().Get("kind"))
		_, _ = w.Write([]byte(`{"count":0,"items":[]}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)
	namespace := "team/a + b"
	service := "checkout/api"
	kind := TransactionKind("HTTP")

	items, err := client.ListTransactions(context.Background(), ListTransactionsParams{
		Namespace: &namespace,
		Service:   &service,
		Kind:      &kind,
	})
	require.NoError(t, err)
	assert.Empty(t, items)
}

func TestClientDecodesCollectionEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/v1/services", r.URL.Path)
		_, _ = w.Write([]byte(`{
			"count": 1,
			"items": [{
				"namespace": "prod",
				"service": "checkout",
				"transaction_count": 3,
				"volume": 12,
				"last_seen": "2026-08-05T06:00:00Z"
			}]
		}`))
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)

	items, err := client.ListServices(context.Background())
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, int64(12), items[0].Volume)
}

func TestClientMapsTypedAPIErrors(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		code       ErrorCode
		target     error
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, code: ErrorCodeBadRequest, target: ErrBadRequest},
		{name: "not found", statusCode: http.StatusNotFound, code: ErrorCodeNotFound, target: ErrNotFound},
		{name: "conflict", statusCode: http.StatusConflict, code: ErrorCodeConflict, target: ErrConflict},
		{name: "validation", statusCode: http.StatusUnprocessableEntity, code: ErrorCodeUnprocessableEntity, target: ErrUnprocessableEntity},
		{name: "unavailable", statusCode: http.StatusServiceUnavailable, code: ErrorCodeUnavailable, target: ErrUnavailable},
		{name: "internal", statusCode: http.StatusInternalServerError, code: ErrorCodeInternal, target: ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]any{
						"code":    tt.code,
						"message": "request failed",
						"details": map[string]any{"field": "scope_key"},
					},
				})
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)
			_, err = client.GetTransaction(context.Background(), 1)
			require.Error(t, err)
			assert.ErrorIs(t, err, tt.target)

			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
			assert.Equal(t, tt.code, apiErr.Code)
			assert.JSONEq(t, `{"field":"scope_key"}`, string(apiErr.Details))
		})
	}
}

func TestClientMapsEngineStarting(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "error envelope",
			body: `{"error":{"code":"unavailable","message":"engine starting; retry shortly"}}`,
		},
		{
			name: "plain body",
			body: "Insights engine is starting",
		},
		{
			name: "server warm-up message",
			body: `{"error":{"code":"unavailable","message":"engine not ready; try again shortly"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)
			err = client.ResetTransactionBaselines(context.Background(), 42)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrEngineStarting)
			assert.ErrorIs(t, err, ErrUnavailable)

			var startingErr *EngineStartingError
			require.ErrorAs(t, err, &startingErr)
			assert.Equal(t, http.StatusServiceUnavailable, startingErr.StatusCode)
		})
	}
}

func TestClientSendsJSONAndAcceptsNoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		var request ViolationActionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
		assert.Equal(t, "prod/checkout", request.ScopeKey)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := NewClient(server.URL)
	require.NoError(t, err)
	err = client.AllowGuardrailViolation(context.Background(), ViolationActionRequest{
		ScopeKey:  "prod/checkout",
		RuleKey:   "allowed_egress",
		Offending: "db:5432",
	})
	require.NoError(t, err)
}

func TestNewClientValidatesBaseURL(t *testing.T) {
	for _, baseURL := range []string{"", "insights:8080", "://bad"} {
		t.Run(fmt.Sprintf("%q", baseURL), func(t *testing.T) {
			_, err := NewClient(baseURL)
			require.Error(t, err)
		})
	}
}
