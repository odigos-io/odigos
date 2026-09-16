package graph

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services/insights"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const insightsTestNamespace = "odigos-insights-test"

// insightsEngineCall is one request the insights REST client sent to the stub.
type insightsEngineCall struct {
	method string
	path   string
	query  url.Values
	body   string
}

type insightsEngineReply struct {
	status int
	body   string
}

// insightsEngineStub stands in for the odigos-insights service. It routes on
// method and path instead of answering everything with one canned body, so a
// resolver that reaches the wrong endpoint fails the test rather than quietly
// receiving the right answer.
type insightsEngineStub struct {
	t *testing.T

	mu     sync.Mutex
	routes map[string]insightsEngineReply
	calls  []insightsEngineCall

	baseURL string
}

func newInsightsEngineStub(t *testing.T) *insightsEngineStub {
	t.Helper()

	stub := &insightsEngineStub{t: t, routes: map[string]insightsEngineReply{}}
	server := httptest.NewServer(http.HandlerFunc(stub.serve))
	t.Cleanup(server.Close)
	stub.baseURL = server.URL
	return stub
}

func (s *insightsEngineStub) serve(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	s.mu.Lock()
	s.calls = append(s.calls, insightsEngineCall{
		method: r.Method,
		path:   r.URL.Path,
		query:  r.URL.Query(),
		body:   strings.TrimSpace(string(body)),
	})
	reply, routed := s.routes[r.Method+" "+r.URL.Path]
	s.mu.Unlock()

	if !routed {
		// Errorf, not Fatalf: this runs on the test server's goroutine.
		s.t.Errorf("insights stub has no route for %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(reply.status)
	_, _ = w.Write([]byte(reply.body))
}

// respond registers a 200 JSON reply for one endpoint.
func (s *insightsEngineStub) respond(method, path, body string) *insightsEngineStub {
	return s.reply(method, path, insightsEngineReply{status: http.StatusOK, body: body})
}

// accept registers the 204 No Content the engine answers its writes with.
func (s *insightsEngineStub) accept(method, path string) *insightsEngineStub {
	return s.reply(method, path, insightsEngineReply{status: http.StatusNoContent})
}

// fail registers an error envelope in the shape the insights REST API returns.
func (s *insightsEngineStub) fail(method, path string, status int, code insights.ErrorCode) *insightsEngineStub {
	return s.reply(method, path, insightsEngineReply{
		status: status,
		body:   `{"error":{"code":"` + string(code) + `","message":"stubbed failure"}}`,
	})
}

func (s *insightsEngineStub) reply(method, path string, reply insightsEngineReply) *insightsEngineStub {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[method+" "+path] = reply
	return s
}

func (s *insightsEngineStub) recorded() []insightsEngineCall {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]insightsEngineCall(nil), s.calls...)
}

// only returns the single call the engine received, failing when a resolver
// issued a different number of requests than expected.
func (s *insightsEngineStub) only() insightsEngineCall {
	s.t.Helper()
	calls := s.recorded()
	require.Len(s.t, calls, 1)
	return calls[0]
}

// newInsightsResolver wires a resolver with insights enabled in the effective
// config and its REST client pointed at stub.
func newInsightsResolver(t *testing.T, stub *insightsEngineStub) *Resolver {
	t.Helper()
	return insightsResolverFor(t, "insights:\n  enabled: true\n", stub)
}

// newInsightsDisabledResolver leaves insights off in the effective config. The
// stub is still attached so a test can prove the gate rejected the call before
// any request reached the engine.
func newInsightsDisabledResolver(t *testing.T, stub *insightsEngineStub) *Resolver {
	t.Helper()
	return insightsResolverFor(t, "insights:\n  enabled: false\n", stub)
}

func insightsResolverFor(t *testing.T, configYAML string, stub *insightsEngineStub) *Resolver {
	t.Helper()

	resolver := &Resolver{K8sCacheClient: insightsEffectiveConfigClient(t, configYAML)}
	if stub != nil {
		client, err := insights.NewClient(stub.baseURL)
		require.NoError(t, err)
		resolver.InsightsClient = client
	}
	return resolver
}

func insightsEffectiveConfigClient(t *testing.T, configYAML string) client.Client {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, insightsTestNamespace)
	require.Equal(t, insightsTestNamespace, env.GetCurrentNamespace())

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(&v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: insightsTestNamespace,
		},
		Data: map[string]string{consts.OdigosConfigurationFileName: configYAML},
	}).Build()
}

// requireInsightsErrorCode asserts the GraphQL extensions code a resolver
// returned. The UI switches on that code to tell "feature is off" from "engine
// warming up" from a real failure, so it is the resolver layer's contract —
// unlike the message text.
func requireInsightsErrorCode(t *testing.T, err error, want string) {
	t.Helper()

	var gqlErr *gqlerror.Error
	require.ErrorAs(t, err, &gqlErr)
	require.Equal(t, want, gqlErr.Extensions["code"])
}

func insightsStrPtr(value string) *string {
	return &value
}

func insightsIntPtr(value int) *int {
	return &value
}
