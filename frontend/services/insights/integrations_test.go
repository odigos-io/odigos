package insights

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationEndpointContract(t *testing.T) {
	ctx := context.Background()
	write := IntegrationWrite{Name: "Security Slack", Type: "slack", Enabled: false, Config: json.RawMessage(`{"webhook_url":"","mention":"<!channel>"}`)}
	tests := []struct {
		name, method, path, response string
		call                         func(*Client) error
	}{
		{"catalog", "GET", "/api/v1/integrations/catalog", `{"items":[{"type":"future","fields":[{"key":"region","type":"select","options":["eu","us"]}]}]}`, func(c *Client) error {
			values, err := c.ListIntegrationCatalog(ctx)
			require.Len(t, values, 1)
			assert.Equal(t, []string{"eu", "us"}, values[0].Fields[0].Options)
			return err
		}},
		{"list", "GET", "/api/v1/integrations", `{"items":[{"id":"i-1","config":{"webhook_url":"••••••••"},"health":{"failed":2}}]}`, func(c *Client) error {
			values, err := c.ListIntegrations(ctx)
			require.Len(t, values, 1)
			assert.Equal(t, 2, values[0].Health.Failed)
			return err
		}},
		{"get", "GET", "/api/v1/integrations/i-1", `{"id":"i-1"}`, func(c *Client) error { _, err := c.GetIntegration(ctx, "i-1"); return err }},
		{"create", "POST", "/api/v1/integrations", `{"id":"i-1"}`, func(c *Client) error { _, err := c.CreateIntegration(ctx, write); return err }},
		{"update", "PUT", "/api/v1/integrations/i-1", `{"id":"i-1"}`, func(c *Client) error { _, err := c.UpdateIntegration(ctx, "i-1", write); return err }},
		{"delete", "DELETE", "/api/v1/integrations/i-1", "", func(c *Client) error { return c.DeleteIntegration(ctx, "i-1") }},
		{"test", "POST", "/api/v1/integrations/i-1/test", "", func(c *Client) error { return c.TestIntegration(ctx, "i-1") }},
		{"retry", "POST", "/api/v1/integrations/i-1/deliveries/retry", "", func(c *Client) error { return c.RetryIntegrationDeliveries(ctx, "i-1", []string{"anomaly:42:a&b"}) }},
		{"deliveries", "GET", "/api/v1/integrations/i-1/deliveries", `{"items":[]}`, func(c *Client) error {
			_, err := c.ListIntegrationDeliveries(ctx, "i-1", strPtr("dead"), intPtr(20))
			return err
		}},
		{"attempts", "GET", "/api/v1/integrations/i-1/attempts", `{"items":[]}`, func(c *Client) error { _, err := c.ListIntegrationAttempts(ctx, "i-1", intPtr(20)); return err }},
		{"finding deliveries", "GET", "/api/v1/findings/deliveries", `{"items":[]}`, func(c *Client) error { _, err := c.GetFindingDeliveries(ctx, "anomaly:42:a&b"); return err }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, request := newCapturingClient(t, test.response)
			require.NoError(t, test.call(c))
			assert.Equal(t, test.method, request.method)
			assert.Equal(t, test.path, request.path)
			switch test.name {
			case "create", "update":
				assert.JSONEq(t, `{"name":"Security Slack","type":"slack","enabled":false,"config":{"webhook_url":"","mention":"<!channel>"},"filters":{"kinds":null,"notify_on_resolve":false}}`, request.body)
			case "retry":
				assert.JSONEq(t, `{"finding_keys":["anomaly:42:a&b"]}`, request.body)
			case "deliveries":
				assert.Equal(t, "dead", request.query.Get("state"))
				assert.Equal(t, "20", request.query.Get("limit"))
			case "attempts":
				assert.Equal(t, "20", request.query.Get("limit"))
			case "finding deliveries":
				assert.Equal(t, "anomaly:42:a&b", request.query.Get("finding_key"))
			}
		})
	}
}

func TestIntegrationInputKeepsTypedConfigAndBlankSecrets(t *testing.T) {
	input := model.InsightsIntegrationInput{Name: "Alerts", Type: "future", Enabled: false, Config: `{"password":"","batch":20,"verify_tls":true}`, Filters: &model.InsightsIntegrationFiltersInput{Kinds: []model.InsightsFindingKind{model.InsightsFindingKindAnomaly}, NotifyOnResolve: true}}
	value, err := IntegrationFromInput(input)
	require.NoError(t, err)
	assert.JSONEq(t, input.Config, string(value.Config))
	assert.False(t, value.Enabled)
	assert.Equal(t, []FindingKind{"anomaly"}, value.Filters.Kinds)
	assert.True(t, value.Filters.NotifyOnResolve)
	for _, config := range []string{`null`, `[]`, `"secret"`, `{bad`} {
		input.Config = config
		_, err := IntegrationFromInput(input)
		assert.ErrorIs(t, err, ErrBadRequest)
	}
}

func TestDeliverySettingsSurvivePolicyAndGuardrailRoundTrips(t *testing.T) {
	delivery := &model.InsightsDeliverySettingsInput{Muted: true, ExcludedIntegrationIds: []string{"slack-1", "future-1"}}
	policy, err := PolicyFromInput(model.InsightsPolicyInput{Name: "Alerts", Scope: model.InsightsPolicyScopeGlobal, Delivery: delivery})
	require.NoError(t, err)
	converted, err := PolicyToModel(policy)
	require.NoError(t, err)
	assert.Equal(t, delivery.Muted, converted.Delivery.Muted)
	assert.Equal(t, delivery.ExcludedIntegrationIds, converted.Delivery.ExcludedIntegrationIds)
	guardrail := GuardrailFromInput(model.InsightsGuardrailInput{Scope: model.InsightsPolicyScopeService, ScopeKey: "default/api", Delivery: delivery})
	assert.Equal(t, converted.Delivery, GuardrailToModel(guardrail).Delivery)
	encoded, err := json.Marshal(guardrail)
	require.NoError(t, err)
	assert.Contains(t, string(encoded), `"delivery":{"muted":true,"excluded_integration_ids":["slack-1","future-1"]}`)
	assert.Nil(t, DeliverySettingsFromInput(nil))
}

func TestIntegrationTestPropagatesUpstreamFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":{"code":"unprocessable_entity","message":"Slack returned HTTP 403"}}`))
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	require.NoError(t, err)
	err = client.TestIntegration(context.Background(), "slack-1")
	assert.ErrorIs(t, err, ErrValidation)
	assert.Contains(t, err.Error(), "Slack returned HTTP 403")
}

func TestIntegrationTimestampsMapZeroTimeToEmpty(t *testing.T) {
	const zero = "0001-01-01T00:00:00Z"
	const setAt = "2026-10-11T09:30:15Z"

	health := IntegrationHealthToModel(IntegrationHealth{LastSuccessAt: zero, LastFailureAt: setAt})
	assert.Equal(t, "", health.LastSuccessAt)
	assert.Equal(t, setAt, health.LastFailureAt)

	delivery := DeliveryToModel(Delivery{NextAttemptAt: setAt, SentAt: zero})
	assert.Equal(t, setAt, delivery.NextAttemptAt)
	assert.Equal(t, "", delivery.SentAt)

	assert.Equal(t, "", IntegrationHealthToModel(IntegrationHealth{}).LastSuccessAt)
	assert.Equal(t, "", DeliveryToModel(Delivery{}).SentAt)
}
