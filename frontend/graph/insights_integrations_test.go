package graph

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestInsightsIntegrationGraphQLContract(t *testing.T) {
	integration := `{"id":"slack-1","name":"Security Slack","type":"slack","enabled":true,"config":{"webhook_url":"••••••••"},"filters":{},"health":{},"created_at":"2026-10-14T10:00:00Z","updated_at":"2026-10-14T11:00:00Z"}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == "/api/v1/integrations/catalog":
			_, _ = w.Write([]byte(`{"items":[{"type":"slack","name":"Slack","fields":[{"key":"webhook_url","label":"Incoming webhook URL","type":"secret","required":true}],"capabilities":{"max_batch":1}}]}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"items":[` + integration + `]}`))
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/integrations":
			var input insights.IntegrationWrite
			require.NoError(t, json.NewDecoder(r.Body).Decode(&input))
			assert.False(t, input.Enabled)
			assert.JSONEq(t, `{"webhook_url":"example-only"}`, string(input.Config))
			_, _ = w.Write([]byte(integration))
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer upstream.Close()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	config := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: env.GetCurrentNamespace()}, Data: map[string]string{consts.OdigosConfigurationFileName: "insights:\n  enabled: true\n"}}
	client, err := insights.NewClient(upstream.URL)
	require.NoError(t, err)
	resolver := &Resolver{K8sCacheClient: fake.NewClientBuilder().WithScheme(scheme).WithObjects(config).Build(), InsightsClient: client}
	server := handler.NewDefaultServer(NewExecutableSchema(Config{Resolvers: resolver}))
	tests := []string{
		`{ insights { integrationCatalog { type fields { key type required options default } capabilities { maxBatch supportsResolve } } integrations { id config filters { minSeverity kinds notifyOnResolve } health { pending failed dead lastError } } } }`,
		`mutation { createInsightsIntegration(integration: {name: "Slack", type: "slack", enabled: false, config: "{\"webhook_url\":\"example-only\"}", filters: {kinds: [], notifyOnResolve: false}}) { id config } }`,
		`mutation { testInsightsIntegration(id: "slack-1") deleteInsightsIntegration(id: "slack-1") retryInsightsIntegrationDeliveries(id: "slack-1", findingKeys: []) }`,
	}
	for _, query := range tests {
		body, err := json.Marshal(map[string]string{"query": query})
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		assert.Equal(t, http.StatusOK, response.Code)
		var result map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
		assert.NotContains(t, result, "errors", response.Body.String())
		assert.Contains(t, result, "data")
	}
}
