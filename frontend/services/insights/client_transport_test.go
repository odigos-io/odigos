package insights

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errorBodyTransport struct{}

func (errorBodyTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(iotest.ErrReader(errors.New("connection reset"))),
	}, nil
}

// Every transport failure has to arrive at the resolver as an error rather than
// as an empty-but-successful response, which the UI would render as "no data".
func TestClientReportsTransportFailures(t *testing.T) {
	ctx := context.Background()

	t.Run("request body cannot be encoded", func(t *testing.T) {
		client, err := NewClient("http://insights:8080")
		require.NoError(t, err)

		err = client.do(ctx, http.MethodPost, client.apiEndpoint("policies"), make(chan int), nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "encode insights request")
	})

	t.Run("request cannot be built", func(t *testing.T) {
		client, err := NewClient("http://insights:8080")
		require.NoError(t, err)

		err = client.do(ctx, "GET WITH A SPACE", client.apiEndpoint("services"), nil, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "create insights request")
	})

	t.Run("the engine is unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		baseURL := server.URL
		server.Close()

		client, err := NewClient(baseURL)
		require.NoError(t, err)

		_, err = client.ListServices(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "send insights request")
	})

	t.Run("the response body cannot be read", func(t *testing.T) {
		client, err := NewClientWithHTTPClient("http://insights:8080", &http.Client{Transport: errorBodyTransport{}})
		require.NoError(t, err)

		_, err = client.ListServices(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "read insights response")
	})

	t.Run("the response body is not JSON", func(t *testing.T) {
		client, _ := newCapturingClient(t, "<html>gateway</html>")

		_, err := client.GetCatalog(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "decode insights response")
	})
}

// 204 is the documented answer to the upsert endpoints, so a caller that also
// passes a result must not fail trying to decode an empty body.
func TestClientAcceptsNoContentForAReadRequest(t *testing.T) {
	client, _ := newCapturingClient(t, "")

	result := SystemSettings{Retention: SystemRetentionSettings{ObservationRetentionDays: 3}}
	require.NoError(t, client.do(context.Background(), http.MethodGet, client.apiEndpoint("system-settings"), nil, &result))
	assert.Equal(t, 3, result.Retention.ObservationRetentionDays, "an empty body must leave the result untouched")
}

func TestClientErrorMessageFallbacks(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		body        string
		wantMessage string
		wantCode    ErrorCode
	}{
		{
			name:        "empty body falls back to the status text",
			statusCode:  http.StatusBadGateway,
			body:        "",
			wantMessage: "Bad Gateway (invalid error response: unexpected end of JSON input)",
		},
		{
			name:        "plain text body is kept as the message",
			statusCode:  http.StatusInternalServerError,
			body:        "  boom  ",
			wantMessage: "boom (invalid error response: invalid character 'b' looking for beginning of value)",
		},
		{
			name:        "an envelope without a message falls back to the raw body",
			statusCode:  http.StatusNotFound,
			body:        `{"error":{"code":"not_found","message":"   "}}`,
			wantMessage: `{"error":{"code":"not_found","message":"   "}}`,
			wantCode:    ErrorCodeNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			_, err = client.GetCatalog(context.Background())
			require.Error(t, err)

			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr)
			assert.Equal(t, tt.statusCode, apiErr.StatusCode)
			assert.Equal(t, tt.wantCode, apiErr.Code)
			assert.Equal(t, tt.wantMessage, apiErr.Message)
		})
	}
}

// A response that carries no error code still has to land in the right
// category, because the resolver picks the GraphQL extensions code from it.
func TestAPIErrorCategoryFallsBackToTheStatusCode(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       error
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, want: ErrBadRequest},
		{name: "not found", statusCode: http.StatusNotFound, want: ErrNotFound},
		{name: "conflict", statusCode: http.StatusConflict, want: ErrConflict},
		{name: "validation", statusCode: http.StatusUnprocessableEntity, want: ErrUnprocessableEntity},
		{name: "unavailable", statusCode: http.StatusServiceUnavailable, want: ErrUnavailable},
		{name: "bad gateway", statusCode: http.StatusBadGateway, want: ErrInternal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode, Message: "no code"}
			assert.ErrorIs(t, err, tt.want)
			assert.Equal(t, errorCodeFor(tt.want), errorCode(err))
		})
	}

	// 4xx statuses the engine does not use are not silently promoted to a
	// category the UI would act on.
	teapot := &APIError{StatusCode: http.StatusTeapot, Message: "no code"}
	for _, category := range []error{ErrBadRequest, ErrNotFound, ErrConflict, ErrUnprocessableEntity, ErrUnavailable, ErrInternal} {
		assert.NotErrorIs(t, teapot, category)
	}
	assert.Equal(t, CodeInternal, errorCode(teapot))
}

func errorCodeFor(category error) string {
	switch category {
	case ErrBadRequest:
		return CodeBadRequest
	case ErrNotFound:
		return CodeNotFound
	case ErrConflict:
		return CodeConflict
	case ErrUnprocessableEntity:
		return CodeValidation
	case ErrUnavailable:
		return CodeUnavailable
	default:
		return CodeInternal
	}
}

// Every read method has to surface an engine failure instead of returning a
// zero value that the UI would render as "nothing found".
func TestClientPropagatesEngineErrors(t *testing.T) {
	ctx := context.Background()

	calls := map[string]func(*Client) error{
		"ListServices":            func(c *Client) error { _, err := c.ListServices(ctx); return err },
		"ListServiceNames":        func(c *Client) error { _, err := c.ListServiceNames(ctx); return err },
		"GetServiceProfile":       func(c *Client) error { _, err := c.GetServiceProfile(ctx, "prod", "checkout"); return err },
		"GetBlastRadius":          func(c *Client) error { _, err := c.GetBlastRadius(ctx, "prod", "checkout", nil); return err },
		"ListTransactions":        func(c *Client) error { _, err := c.ListTransactions(ctx, ListTransactionsParams{}); return err },
		"GetTransaction":          func(c *Client) error { _, err := c.GetTransaction(ctx, 7); return err },
		"DeleteTransaction":       func(c *Client) error { return c.DeleteTransaction(ctx, 7) },
		"BulkDeleteTransactions":  func(c *Client) error { _, err := c.BulkDeleteTransactions(ctx, []int64{7}); return err },
		"GetTransactionBaseline":  func(c *Client) error { _, err := c.GetTransactionBaseline(ctx, 7); return err },
		"PromoteBaselineClass":    func(c *Client) error { _, err := c.PromoteBaselineClass(ctx, 7, "D2_egress"); return err },
		"BulkPromoteTransactions": func(c *Client) error { _, err := c.BulkPromoteTransactions(ctx, []int64{7}); return err },
		"ListObservations":        func(c *Client) error { _, err := c.ListObservations(ctx, 7, ListObservationsParams{}); return err },
		"GetObservation":          func(c *Client) error { _, err := c.GetObservation(ctx, 7, "trace-1"); return err },
		"ListPolicies":            func(c *Client) error { _, err := c.ListPolicies(ctx); return err },
		"ListLearningPolicies":    func(c *Client) error { _, err := c.ListLearningPolicies(ctx); return err },
		"ListFindings":            func(c *Client) error { _, err := c.ListFindings(ctx, ListFindingsParams{}); return err },
		"ListAnomalies":           func(c *Client) error { _, err := c.ListAnomalies(ctx, ListAnomaliesParams{}); return err },
		"GetAnomaly":              func(c *Client) error { _, err := c.GetAnomaly(ctx, 7, "sig-1"); return err },
		"BulkResolveAnomalies":    func(c *Client) error { _, err := c.BulkResolveAnomalies(ctx, BulkAnomalyRequest{}); return err },
		"ListGuardrails":          func(c *Client) error { _, err := c.ListGuardrails(ctx); return err },
		"ListGuardrailViolations": func(c *Client) error {
			_, err := c.ListGuardrailViolations(ctx, ListGuardrailViolationsParams{})
			return err
		},
		"GetGuardrailViolation": func(c *Client) error {
			_, err := c.GetGuardrailViolation(ctx, "prod/checkout", "allowed_egress", "db")
			return err
		},
		"GetCatalog":               func(c *Client) error { _, err := c.GetCatalog(ctx); return err },
		"GetSystemSettings":        func(c *Client) error { _, err := c.GetSystemSettings(ctx); return err },
		"UpdateSystemSettings":     func(c *Client) error { return c.UpdateSystemSettings(ctx, SystemSettings{}) },
		"GetStorageHealth":         func(c *Client) error { _, err := c.GetStorageHealth(ctx); return err },
		"UpsertPolicyAndRead":      func(c *Client) error { _, err := c.UpsertPolicyAndRead(ctx, Policy{}); return err },
		"UpsertLearningPolicyRead": func(c *Client) error { _, err := c.UpsertLearningPolicyAndRead(ctx, LearningPolicy{}); return err },
		"UpsertGuardrailAndRead":   func(c *Client) error { _, err := c.UpsertGuardrailAndRead(ctx, Guardrail{}); return err },
		"UpdateSystemSettingsRead": func(c *Client) error { _, err := c.UpdateSystemSettingsAndRead(ctx, SystemSettings{}); return err },
	}

	for name, call := range calls {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"internal_error","message":"engine failed"}}`))
			}))
			defer server.Close()

			client, err := NewClient(server.URL)
			require.NoError(t, err)

			err = call(client)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrInternal)
		})
	}
}

func TestAPIErrorMessageIncludesTheCodeWhenPresent(t *testing.T) {
	withCode := &APIError{StatusCode: http.StatusNotFound, Code: ErrorCodeNotFound, Message: "no such policy"}
	assert.Equal(t, "insights API returned not_found (status 404): no such policy", withCode.Error())

	withoutCode := &APIError{StatusCode: http.StatusBadGateway, Message: "boom"}
	assert.Equal(t, "insights API returned status 502: boom", withoutCode.Error())
}
