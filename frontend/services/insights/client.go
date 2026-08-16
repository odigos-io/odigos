package insights

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	return NewClientWithHTTPClient(baseURL, http.DefaultClient)
}

func NewClientWithHTTPClient(baseURL string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, fmt.Errorf("insights base URL is empty")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse insights base URL: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("insights base URL must include scheme and host")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{baseURL: parsed, httpClient: httpClient}, nil
}

func (c *Client) ListServices(ctx context.Context) ([]ServiceStat, error) {
	return doList[ServiceStat](ctx, c, http.MethodGet, c.apiEndpoint("services"), nil)
}

func (c *Client) GetServiceProfile(ctx context.Context, namespace, service string) (*ServiceProfile, error) {
	endpoint := c.apiEndpoint("services", "profile")
	query := endpoint.Query()
	if strings.TrimSpace(namespace) != "" {
		query.Set("namespace", namespace)
	}
	query.Set("service", service)
	endpoint.RawQuery = query.Encode()
	var result ServiceProfile
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBlastRadius(ctx context.Context, namespace, service string, depth *int) (*BlastRadiusSubgraph, error) {
	endpoint := c.apiEndpoint("services", "blast-radius")
	query := endpoint.Query()
	if strings.TrimSpace(namespace) != "" {
		query.Set("namespace", namespace)
	}
	query.Set("service", service)
	if depth != nil {
		query.Set("depth", strconv.Itoa(*depth))
	}
	endpoint.RawQuery = query.Encode()
	var result BlastRadiusSubgraph
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListTransactions(ctx context.Context, params ListTransactionsParams) ([]TransactionStat, error) {
	endpoint := c.apiEndpoint("transactions")
	query := endpoint.Query()
	addOptionalString(query, "namespace", params.Namespace)
	addOptionalString(query, "service", params.Service)
	if params.Kind != nil {
		query.Set("kind", string(*params.Kind))
	}
	endpoint.RawQuery = query.Encode()
	return doList[TransactionStat](ctx, c, http.MethodGet, endpoint, nil)
}

func (c *Client) GetTransaction(ctx context.Context, transactionID int64) (*Transaction, error) {
	var result Transaction
	if err := c.do(ctx, http.MethodGet, c.transactionEndpoint(transactionID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteTransaction(ctx context.Context, transactionID int64) error {
	return c.do(ctx, http.MethodDelete, c.transactionEndpoint(transactionID), nil, nil)
}

func (c *Client) GetTransactionBaseline(ctx context.Context, transactionID int64) ([]BaselineClass, error) {
	return doList[BaselineClass](ctx, c, http.MethodGet, c.transactionEndpoint(transactionID, "baseline"), nil)
}

func (c *Client) PromoteBaselineClass(ctx context.Context, transactionID int64, class DeviationClass) (*PromoteResult, error) {
	var result PromoteResult
	if err := c.do(ctx, http.MethodPost, c.transactionEndpoint(transactionID, "baseline", string(class), "promote"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ResetBaselineClass(ctx context.Context, transactionID int64, class DeviationClass) error {
	return c.do(ctx, http.MethodPost, c.transactionEndpoint(transactionID, "baseline", string(class), "reset"), nil, nil)
}

func (c *Client) ResetTransactionBaselines(ctx context.Context, transactionID int64) error {
	return c.do(ctx, http.MethodPost, c.transactionEndpoint(transactionID, "baselines", "reset"), nil, nil)
}

func (c *Client) ListObservations(ctx context.Context, transactionID int64, params ListObservationsParams) ([]ObservationSummary, error) {
	endpoint := c.transactionEndpoint(transactionID, "observations")
	query := endpoint.Query()
	if params.SampleReason != nil {
		query.Set("sample_reason", string(*params.SampleReason))
	}
	endpoint.RawQuery = query.Encode()
	return doList[ObservationSummary](ctx, c, http.MethodGet, endpoint, nil)
}

func (c *Client) GetObservation(ctx context.Context, transactionID int64, traceID string) (*Observation, error) {
	var result Observation
	if err := c.do(ctx, http.MethodGet, c.transactionEndpoint(transactionID, "observations", traceID), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListPolicies(ctx context.Context) ([]Policy, error) {
	return doList[Policy](ctx, c, http.MethodGet, c.apiEndpoint("policies"), nil)
}

func (c *Client) UpsertPolicy(ctx context.Context, policy Policy) error {
	return c.do(ctx, http.MethodPut, c.apiEndpoint("policies"), policy, nil)
}

func (c *Client) DeletePolicy(ctx context.Context, scope PolicyScope, scopeKey string) error {
	endpoint := c.apiEndpoint("policies")
	query := endpoint.Query()
	query.Set("scope", string(scope))
	query.Set("scope_key", scopeKey)
	endpoint.RawQuery = query.Encode()
	return c.do(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (c *Client) ListLearningPolicies(ctx context.Context) ([]LearningPolicy, error) {
	return doList[LearningPolicy](ctx, c, http.MethodGet, c.apiEndpoint("learning-policies"), nil)
}

func (c *Client) UpsertLearningPolicy(ctx context.Context, policy LearningPolicy) error {
	return c.do(ctx, http.MethodPut, c.apiEndpoint("learning-policies"), policy, nil)
}

func (c *Client) DeleteLearningPolicy(ctx context.Context, class DeviationClass, scope PolicyScope, scopeKey string) error {
	endpoint := c.apiEndpoint("learning-policies")
	query := endpoint.Query()
	query.Set("class", string(class))
	query.Set("scope", string(scope))
	query.Set("scope_key", scopeKey)
	endpoint.RawQuery = query.Encode()
	return c.do(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (c *Client) ListFindings(ctx context.Context, params ListFindingsParams) ([]Finding, error) {
	endpoint := c.apiEndpoint("findings")
	addFindingFilters(endpoint, params.WindowHours, params.Service, params.Namespace, params.Status)
	query := endpoint.Query()
	if params.Kind != nil {
		query.Set("kind", string(*params.Kind))
	}
	endpoint.RawQuery = query.Encode()
	return doList[Finding](ctx, c, http.MethodGet, endpoint, nil)
}

func (c *Client) ListAnomalies(ctx context.Context, params ListAnomaliesParams) ([]AnomalySummary, error) {
	endpoint := c.apiEndpoint("anomalies")
	addFindingFilters(endpoint, params.WindowHours, params.Service, params.Namespace, params.Status)
	return doList[AnomalySummary](ctx, c, http.MethodGet, endpoint, nil)
}

func (c *Client) GetAnomaly(ctx context.Context, transactionID int64, signature string) (*AnomalyIssue, error) {
	var result AnomalyIssue
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("anomalies", strconv.FormatInt(transactionID, 10), signature), nil, &result); err != nil {
		return nil, err
	}
	// Insights detail should embed compare traces; older builds may omit them even
	// when retained samples exist. Fill from the observations API when missing.
	c.enrichAnomalyCompareTraces(ctx, &result)
	return &result, nil
}

// enrichAnomalyCompareTraces loads anomaly/baseline OTLP samples when the anomaly
// detail payload omitted anomaly_trace / baseline_trace.
func (c *Client) enrichAnomalyCompareTraces(ctx context.Context, issue *AnomalyIssue) {
	if issue == nil {
		return
	}
	if issue.AnomalyTrace == nil {
		reason := AnomalyEvidenceSampleReason(issue.Signature)
		if obs := c.firstObservation(ctx, issue.TransactionID, &reason); obs != nil {
			issue.AnomalyTrace = obs
		} else if issue.LastTraceID != nil && *issue.LastTraceID != "" {
			if obs, err := c.GetObservation(ctx, issue.TransactionID, *issue.LastTraceID); err == nil {
				issue.AnomalyTrace = obs
			}
		}
	}
	if issue.BaselineTrace == nil {
		reason := SampleReasonExample
		if obs := c.firstObservation(ctx, issue.TransactionID, &reason); obs != nil {
			issue.BaselineTrace = obs
		}
	}
}

func (c *Client) firstObservation(ctx context.Context, transactionID int64, reason *SampleReason) *Observation {
	samples, err := c.ListObservations(ctx, transactionID, ListObservationsParams{SampleReason: reason})
	if err != nil || len(samples) == 0 {
		return nil
	}
	obs, err := c.GetObservation(ctx, transactionID, samples[0].TraceID)
	if err != nil {
		return nil
	}
	return obs
}

func (c *Client) ResolveAnomaly(ctx context.Context, transactionID int64, signature string, resolution AnomalyResolution) error {
	body := struct {
		Resolution AnomalyResolution `json:"resolution"`
	}{Resolution: resolution}
	return c.do(ctx, http.MethodPost, c.apiEndpoint("anomalies", strconv.FormatInt(transactionID, 10), signature), body, nil)
}

func (c *Client) BulkResolveAnomalies(ctx context.Context, request BulkAnomalyRequest) (*BulkResolveResult, error) {
	var result BulkResolveResult
	if err := c.do(ctx, http.MethodPost, c.apiEndpoint("anomalies"), request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) ListGuardrails(ctx context.Context) ([]Guardrail, error) {
	return doList[Guardrail](ctx, c, http.MethodGet, c.apiEndpoint("guardrails"), nil)
}

func (c *Client) UpsertGuardrail(ctx context.Context, guardrail Guardrail) error {
	return c.do(ctx, http.MethodPut, c.apiEndpoint("guardrails"), guardrail, nil)
}

func (c *Client) DeleteGuardrail(ctx context.Context, scopeKey string) error {
	endpoint := c.apiEndpoint("guardrails")
	query := endpoint.Query()
	query.Set("scope_key", scopeKey)
	endpoint.RawQuery = query.Encode()
	return c.do(ctx, http.MethodDelete, endpoint, nil, nil)
}

func (c *Client) SeedGuardrail(ctx context.Context, request GuardrailSeedRequest) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("guardrails", "seed"), request, nil)
}

func (c *Client) ListGuardrailViolations(ctx context.Context, params ListGuardrailViolationsParams) ([]GuardrailViolation, error) {
	endpoint := c.apiEndpoint("guardrails", "violations")
	addFindingFilters(endpoint, params.WindowHours, params.Service, params.Namespace, params.Status)
	return doList[GuardrailViolation](ctx, c, http.MethodGet, endpoint, nil)
}

func (c *Client) GetGuardrailViolation(ctx context.Context, scopeKey, ruleKey, offending string) (*GuardrailViolationDetail, error) {
	endpoint := c.apiEndpoint("guardrails", "violations", "evidence")
	query := endpoint.Query()
	query.Set("scope_key", scopeKey)
	query.Set("rule_key", ruleKey)
	query.Set("offending", offending)
	endpoint.RawQuery = query.Encode()
	var result GuardrailViolationDetail
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) AllowGuardrailViolation(ctx context.Context, request ViolationActionRequest) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("guardrails", "violations", "allow"), request, nil)
}

func (c *Client) DismissGuardrailViolation(ctx context.Context, request ViolationActionRequest) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("guardrails", "violations", "dismiss"), request, nil)
}

func (c *Client) ReopenGuardrailViolation(ctx context.Context, request ViolationActionRequest) error {
	return c.do(ctx, http.MethodPost, c.apiEndpoint("guardrails", "violations", "reopen"), request, nil)
}

func (c *Client) GetCatalog(ctx context.Context) (*Catalog, error) {
	var result Catalog
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("catalog"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetSystemSettings(ctx context.Context) (*SystemSettings, error) {
	var result SystemSettings
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("system-settings"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateSystemSettings(ctx context.Context, settings SystemSettings) error {
	return c.do(ctx, http.MethodPut, c.apiEndpoint("system-settings"), settings, nil)
}

func (c *Client) GetStorageHealth(ctx context.Context) (*StorageHealth, error) {
	var result StorageHealth
	if err := c.do(ctx, http.MethodGet, c.apiEndpoint("storage-health"), nil, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) do(ctx context.Context, method string, endpoint *url.URL, body any, result any) error {
	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode insights request: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, endpoint.String(), requestBody)
	if err != nil {
		return fmt.Errorf("create insights request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send insights request: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("read insights response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return decodeAPIError(response.StatusCode, responseBody)
	}
	if result == nil || response.StatusCode == http.StatusNoContent {
		return nil
	}
	if err := json.Unmarshal(responseBody, result); err != nil {
		return fmt.Errorf("decode insights response: %w", err)
	}
	return nil
}

func (c *Client) endpoint(segments ...string) *url.URL {
	endpoint := *c.baseURL
	path := strings.TrimRight(endpoint.Path, "/")
	rawPath := strings.TrimRight(endpoint.EscapedPath(), "/")
	for _, segment := range segments {
		path += "/" + segment
		rawPath += "/" + url.PathEscape(segment)
	}
	endpoint.Path = path
	endpoint.RawPath = rawPath
	return &endpoint
}

func (c *Client) apiEndpoint(segments ...string) *url.URL {
	apiSegments := make([]string, 0, len(segments)+2)
	apiSegments = append(apiSegments, "api", "v1")
	apiSegments = append(apiSegments, segments...)
	return c.endpoint(apiSegments...)
}

func (c *Client) transactionEndpoint(transactionID int64, segments ...string) *url.URL {
	allSegments := []string{"transactions", strconv.FormatInt(transactionID, 10)}
	allSegments = append(allSegments, segments...)
	return c.apiEndpoint(allSegments...)
}

func doList[T any](ctx context.Context, client *Client, method string, endpoint *url.URL, body any) ([]T, error) {
	var envelope struct {
		Items []T `json:"items"`
	}
	if err := client.do(ctx, method, endpoint, body, &envelope); err != nil {
		return nil, err
	}
	return envelope.Items, nil
}

func decodeAPIError(statusCode int, body []byte) error {
	var envelope struct {
		Error struct {
			Code    ErrorCode       `json:"code"`
			Message string          `json:"message"`
			Details json.RawMessage `json:"details"`
		} `json:"error"`
	}
	decodeErr := json.Unmarshal(body, &envelope)
	message := strings.TrimSpace(envelope.Error.Message)
	if message == "" {
		message = strings.TrimSpace(string(body))
	}
	if message == "" {
		message = http.StatusText(statusCode)
	}
	apiErr := &APIError{
		StatusCode: statusCode,
		Code:       envelope.Error.Code,
		Message:    message,
		Details:    envelope.Error.Details,
	}
	if decodeErr != nil {
		apiErr.Message = fmt.Sprintf("%s (invalid error response: %v)", message, decodeErr)
	}
	if isEngineStarting(statusCode, body, apiErr) {
		return &EngineStartingError{APIError: apiErr}
	}
	return apiErr
}

func addOptionalString(query url.Values, key string, value *string) {
	if value != nil {
		query.Set(key, *value)
	}
}

func addFindingFilters(endpoint *url.URL, windowHours *int, service, namespace, status *string) {
	query := endpoint.Query()
	if windowHours != nil {
		query.Set("window_hours", strconv.Itoa(*windowHours))
	}
	addOptionalString(query, "service", service)
	addOptionalString(query, "namespace", namespace)
	addOptionalString(query, "status", status)
	endpoint.RawQuery = query.Encode()
}
