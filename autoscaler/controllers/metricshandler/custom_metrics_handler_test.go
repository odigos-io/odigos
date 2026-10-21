package metricshandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
)

const (
	gatewayTestNamespace = "odigos-test-ns"
	// The gateway exposes its own telemetry on loopback in these tests, so a pod that must fail to
	// be scraped needs an address that can never be dialled regardless of what is bound locally.
	unscrapablePodIP = "not a host"
	// The literal the collector's prometheus endpoint exposes: the memory limiter instrument is
	// created as "odigos_collector_memory_limiter_batch_rejections" (see
	// collector/config/configgrpc/configgrpc.go) and prometheus appends _total to a counter.
	// It is spelled out here on purpose, so that renaming it on either side fails this test
	// instead of silently reporting zero rejections forever.
	collectorRejectionsSeries = "odigos_collector_memory_limiter_batch_rejections_total"
)

func metricsHandlerScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, apiregv1.AddToScheme(scheme))
	return scheme
}

// gatewayStub stands in for the own-telemetry endpoint of a gateway pod.
type gatewayStub struct {
	body     atomic.Pointer[string]
	status   atomic.Int32
	requests atomic.Int32
	lastPath atomic.Pointer[string]
	truncate atomic.Bool
}

func (s *gatewayStub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.requests.Add(1)
	path := r.URL.Path
	s.lastPath.Store(&path)

	if status := int(s.status.Load()); status != 0 {
		w.WriteHeader(status)
		return
	}

	// announce more body than is written and hang up, as a gateway pod that dies mid scrape does
	if s.truncate.Load() {
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = io.WriteString(conn, "HTTP/1.1 200 OK\r\nContent-Length: 4096\r\n\r\n"+*s.body.Load())
		return
	}

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = io.WriteString(w, *s.body.Load())
}

func (s *gatewayStub) serveBody(body string) {
	s.body.Store(&body)
	s.status.Store(0)
}

// serveGatewayOwnTelemetry binds the stub to the fixed own-telemetry port the scraper uses, so a
// pod whose IP is 127.0.0.1 is scraped from it.
func serveGatewayOwnTelemetry(t *testing.T, body string) *gatewayStub {
	t.Helper()

	stub := &gatewayStub{}
	stub.serveBody(body)

	addr := fmt.Sprintf("127.0.0.1:%d", k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault)
	listener, err := net.Listen("tcp", addr)
	require.NoErrorf(t, err, "cannot bind %s to stand in for a gateway pod", addr)

	server := httptest.NewUnstartedServer(stub)
	require.NoError(t, server.Listener.Close())
	server.Listener = listener
	server.Start()
	t.Cleanup(server.Close)

	return stub
}

func rejectionsPayload(values ...float64) string {
	var payload strings.Builder
	payload.WriteString("# HELP " + collectorRejectionsSeries + " batches rejected by the memory limiter\n")
	payload.WriteString("# TYPE " + collectorRejectionsSeries + " counter\n")
	for i, value := range values {
		fmt.Fprintf(&payload, "%s{exporter=\"otlp/%d\"} %g\n", collectorRejectionsSeries, i, value)
	}
	return payload.String()
}

// resetRejectionSamples clears the process-wide previous-scrape cache so tests do not observe each
// other's counters.
func resetRejectionSamples(t *testing.T) {
	t.Helper()

	clear := func() {
		lastSample.Range(func(key, _ any) bool {
			lastSample.Delete(key)
			return true
		})
	}
	clear()
	t.Cleanup(clear)
}

// rememberRejectionSample seeds the previous scrape of a pod, which is what makes the next scrape
// produce a delta.
func rememberRejectionSample(name string, value float64) {
	lastSample.Store(name, value)
}

func rememberedRejectionSample(t *testing.T, name string) float64 {
	t.Helper()

	value, ok := lastSample.Load(name)
	require.Truef(t, ok, "no sample remembered for pod %q", name)
	return value.(float64)
}

// gatewayPod builds a pod that the handler's selector must match. The label and its value are
// spelled out rather than taken from k8sconsts so that a rename that breaks the selector against
// already-running gateway pods fails here.
func gatewayPod(name, podIP string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: gatewayTestNamespace,
			Labels:    map[string]string{"odigos.io/collector-role": "CLUSTER_GATEWAY"},
		},
		Status: corev1.PodStatus{PodIP: podIP},
	}
}

func oomKilledGatewayPod(name string) *corev1.Pod {
	pod := gatewayPod(name, "127.0.0.1")
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name:  k8sconsts.OdigosClusterCollectorContainerName,
		State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
		LastTerminationState: corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"},
		},
	}}
	return pod
}

func servedMetricValue(t *testing.T, k8sClient client.Client) (*httptest.ResponseRecorder, MetricValueList) {
	t.Helper()

	recorder := httptest.NewRecorder()
	MetricHandler(context.Background(), k8sClient, gatewayTestNamespace)(
		recorder, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	var body MetricValueList
	if recorder.Code == http.StatusOK {
		require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &body))
	}
	return recorder, body
}

// ****************
// isPodOOMKilled
// ****************

func TestIsPodOOMKilled(t *testing.T) {
	terminated := func(reason string) corev1.ContainerState {
		return corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: reason}}
	}
	waiting := func(reason string) corev1.ContainerState {
		return corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}
	}

	for _, tc := range []struct {
		name     string
		statuses []corev1.ContainerStatus
		expected bool
	}{
		{
			name:     "no container statuses yet",
			statuses: nil,
			expected: false,
		},
		{
			name: "running container",
			statuses: []corev1.ContainerStatus{{
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			}},
			expected: false,
		},
		{
			name: "crash looping after an OOM kill",
			statuses: []corev1.ContainerStatus{{
				State:                waiting("CrashLoopBackOff"),
				LastTerminationState: terminated("OOMKilled"),
			}},
			expected: true,
		},
		{
			name: "crash looping after a non OOM exit",
			statuses: []corev1.ContainerStatus{{
				State:                waiting("CrashLoopBackOff"),
				LastTerminationState: terminated("Error"),
			}},
			expected: false,
		},
		{
			name: "crash looping with no recorded termination",
			statuses: []corev1.ContainerStatus{{
				State: waiting("CrashLoopBackOff"),
			}},
			expected: false,
		},
		{
			// a pod stuck pulling its image is not under memory pressure, even though it was
			// OOMKilled at some point in the past.
			name: "waiting for another reason after an OOM kill",
			statuses: []corev1.ContainerStatus{{
				State:                waiting("ImagePullBackOff"),
				LastTerminationState: terminated("OOMKilled"),
			}},
			expected: false,
		},
		{
			name: "currently terminated by the OOM killer",
			statuses: []corev1.ContainerStatus{{
				State: terminated("OOMKilled"),
			}},
			expected: true,
		},
		{
			name: "currently terminated for another reason",
			statuses: []corev1.ContainerStatus{{
				State: terminated("Completed"),
			}},
			expected: false,
		},
		{
			name: "a later container is the one OOM killed",
			statuses: []corev1.ContainerStatus{
				{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{State: terminated("OOMKilled")},
			},
			expected: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pod := gatewayPod("odigos-gateway-1", "127.0.0.1")
			pod.Status.ContainerStatuses = tc.statuses

			assert.Equal(t, tc.expected, isPodOOMKilled(pod))
		})
	}
}

// ****************
// scrapeGatewayMetric
// ****************

func TestScrapeGatewayMetric_SumsEveryRejectionSeries(t *testing.T) {
	stub := serveGatewayOwnTelemetry(t, rejectionsPayload(3, 4.5))

	value, err := scrapeGatewayMetric("127.0.0.1")

	require.NoError(t, err)
	assert.Equal(t, 7.5, value)
	assert.Equal(t, "/metrics", *stub.lastPath.Load())
}

func TestScrapeGatewayMetric_MissingRejectionMetricIsNotAnError(t *testing.T) {
	// a freshly started gateway has not rejected anything yet, so the series is absent
	serveGatewayOwnTelemetry(t, "# HELP otelcol_process_uptime uptime\n"+
		"# TYPE otelcol_process_uptime counter\notelcol_process_uptime 12\n")

	value, err := scrapeGatewayMetric("127.0.0.1")

	require.NoError(t, err)
	assert.Zero(t, value)
}

func TestScrapeGatewayMetric_NonCounterSeriesContributesNothing(t *testing.T) {
	serveGatewayOwnTelemetry(t, "# TYPE "+collectorRejectionsSeries+" gauge\n"+
		collectorRejectionsSeries+" 9\n")

	value, err := scrapeGatewayMetric("127.0.0.1")

	require.NoError(t, err)
	assert.Zero(t, value)
}

func TestScrapeGatewayMetric_UnexpectedStatusCode(t *testing.T) {
	stub := serveGatewayOwnTelemetry(t, rejectionsPayload(1))
	stub.status.Store(http.StatusServiceUnavailable)

	_, err := scrapeGatewayMetric("127.0.0.1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code: 503")
}

func TestScrapeGatewayMetric_TruncatedResponse(t *testing.T) {
	stub := serveGatewayOwnTelemetry(t, rejectionsPayload(1))
	stub.truncate.Store(true)

	_, err := scrapeGatewayMetric("127.0.0.1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read response body")
}

func TestScrapeGatewayMetric_UnparsablePayload(t *testing.T) {
	serveGatewayOwnTelemetry(t, "this is not the prometheus text format {\n")

	_, err := scrapeGatewayMetric("127.0.0.1")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse metrics")
}

func TestScrapeGatewayMetric_UnreachablePod(t *testing.T) {
	_, err := scrapeGatewayMetric(unscrapablePodIP)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reach pod")
}

// ****************
// MetricHandler
// ****************

func TestMetricHandler_NoGatewayPods(t *testing.T) {
	resetRejectionSamples(t)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).Build()

	recorder, _ := servedMetricValue(t, k8sClient)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "no gateway pods found")
}

func TestMetricHandler_IgnoresPodsThatAreNotClusterGateways(t *testing.T) {
	resetRejectionSamples(t)

	nodeCollector := gatewayPod("odigos-data-collection-1", "127.0.0.1")
	nodeCollector.Labels = map[string]string{"odigos.io/collector-role": "NODE_COLLECTOR"}
	unlabeled := gatewayPod("some-app-1", "127.0.0.1")
	unlabeled.Labels = nil
	otherNamespace := gatewayPod("odigos-gateway-1", "127.0.0.1")
	otherNamespace.Namespace = "another-namespace"

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(nodeCollector, unlabeled, otherNamespace).Build()

	recorder, _ := servedMetricValue(t, k8sClient)

	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestMetricHandler_ListFailureIsNotReportedAsNoRejections(t *testing.T) {
	resetRejectionSamples(t)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return errors.New("etcd is unavailable")
			},
		}).Build()

	recorder, _ := servedMetricValue(t, k8sClient)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "failed to list gateway pods")
	assert.Contains(t, recorder.Body.String(), "etcd is unavailable")
}

func TestMetricHandler_FirstScrapeNeverReportsRejections(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(4200))

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	recorder, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, http.StatusOK, recorder.Code)
	// a counter is meaningless without a previous sample: the very first scrape must not be read
	// as 4200 fresh rejections and scale the gateway out.
	assert.Equal(t, "0.00", body.Items[0].Value)
	assert.Equal(t, 4200.0, rememberedRejectionSample(t, "odigos-gateway-1"))
}

func TestMetricHandler_RisingCounterReportsRejections(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(11))
	rememberRejectionSample("odigos-gateway-1", 10)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	recorder, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "1.00", body.Items[0].Value)
	assert.Equal(t, 11.0, rememberedRejectionSample(t, "odigos-gateway-1"))
}

func TestMetricHandler_UnchangedCounterReportsNoRejections(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-1", 10)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	_, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, "0.00", body.Items[0].Value)
}

func TestMetricHandler_RestartedGatewayCounterIsNotReadAsRejections(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(2))
	// the pod restarted, so its counter is lower than the previous sample
	rememberRejectionSample("odigos-gateway-1", 500)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	_, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, "0.00", body.Items[0].Value)
	assert.Equal(t, 2.0, rememberedRejectionSample(t, "odigos-gateway-1"))
}

func TestMetricHandler_HalfOfThePodsRejectingCrossesTheThreshold(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-1", 4)
	rememberRejectionSample("odigos-gateway-2", 10)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		gatewayPod("odigos-gateway-1", "127.0.0.1"),
		gatewayPod("odigos-gateway-2", "127.0.0.1"),
	).Build()

	_, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, "1.00", body.Items[0].Value)
}

func TestMetricHandler_MinorityRejectingDoesNotCrossTheThreshold(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-1", 4)
	rememberRejectionSample("odigos-gateway-2", 10)
	rememberRejectionSample("odigos-gateway-3", 10)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		gatewayPod("odigos-gateway-1", "127.0.0.1"),
		gatewayPod("odigos-gateway-2", "127.0.0.1"),
		gatewayPod("odigos-gateway-3", "127.0.0.1"),
	).Build()

	_, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, "0.00", body.Items[0].Value)
}

func TestMetricHandler_OOMKilledPodCountsAsRejectingWithoutBeingScraped(t *testing.T) {
	resetRejectionSamples(t)
	stub := serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-2", 10)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		oomKilledGatewayPod("odigos-gateway-1"),
		gatewayPod("odigos-gateway-2", "127.0.0.1"),
	).Build()

	_, body := servedMetricValue(t, k8sClient)

	// its own telemetry cannot be trusted while it is being killed, so the pod state alone has to
	// make it count as rejecting
	assert.Equal(t, "1.00", body.Items[0].Value)
	assert.Equal(t, int32(1), stub.requests.Load(), "the OOM killed pod must not be scraped")
}

func TestMetricHandler_UnreachablePodsStillCountTowardsTheTotal(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-1", 4)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		gatewayPod("odigos-gateway-1", "127.0.0.1"),
		gatewayPod("odigos-gateway-2", unscrapablePodIP),
		gatewayPod("odigos-gateway-3", unscrapablePodIP),
	).Build()

	_, body := servedMetricValue(t, k8sClient)

	// one of three pods is known to reject, which is below the threshold; a pod that cannot be
	// scraped is not evidence of rejections.
	assert.Equal(t, "0.00", body.Items[0].Value)
	_, remembered := lastSample.Load("odigos-gateway-2")
	assert.False(t, remembered, "an unreachable pod must not record a sample")
}

func TestMetricHandler_ResponseDescribesTheGatewayDeployment(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(1))

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	recorder, body := servedMetricValue(t, k8sClient)

	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "custom.metrics.k8s.io/v1beta1", body.APIVersion)
	assert.Equal(t, "MetricValueList", body.Kind)
	require.Len(t, body.Items, 1)
	assert.Equal(t, "odigos_gateway_rejections", body.Items[0].MetricName)
	assert.Equal(t, map[string]string{
		"kind":      "Deployment",
		"namespace": gatewayTestNamespace,
		"name":      "odigos-gateway",
	}, body.Items[0].DescribedObject)
	assert.False(t, body.Items[0].Timestamp.IsZero())
}

// failingResponseWriter fails every write, standing in for a client that hung up mid response.
type failingResponseWriter struct{ header http.Header }

func (w *failingResponseWriter) Header() http.Header       { return w.header }
func (w *failingResponseWriter) Write([]byte) (int, error) { return 0, errors.New("connection reset") }
func (w *failingResponseWriter) WriteHeader(int)           {}

func TestMetricHandler_ResponseWriteFailureDoesNotPanic(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(1))

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(gatewayPod("odigos-gateway-1", "127.0.0.1")).Build()

	writer := &failingResponseWriter{header: http.Header{}}
	MetricHandler(context.Background(), k8sClient, gatewayTestNamespace)(
		writer, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	assert.Equal(t, "application/json", writer.header.Get("Content-Type"))
}

// ****************
// DiscoveryHandler
// ****************

func TestDiscoveryHandler(t *testing.T) {
	recorder := httptest.NewRecorder()

	DiscoveryHandler(recorder, httptest.NewRequest(http.MethodGet, "/apis/custom.metrics.k8s.io/v1beta1", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "application/json", recorder.Header().Get("Content-Type"))

	var discovery struct {
		Kind         string `json:"kind"`
		APIVersion   string `json:"apiVersion"`
		GroupVersion string `json:"groupVersion"`
		Resources    []struct {
			Name         string   `json:"name"`
			SingularName string   `json:"singularName"`
			Namespaced   bool     `json:"namespaced"`
			Kind         string   `json:"kind"`
			Verbs        []string `json:"verbs"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &discovery))

	assert.Equal(t, "APIResourceList", discovery.Kind)
	assert.Equal(t, "v1", discovery.APIVersion)
	assert.Equal(t, "custom.metrics.k8s.io/v1beta1", discovery.GroupVersion)
	require.Len(t, discovery.Resources, 1)
	// the aggregation layer only routes requests for metrics advertised here, and only for the
	// resource the HPA describes (a Deployment).
	assert.Equal(t, "deployments.apps/odigos_gateway_rejections", discovery.Resources[0].Name)
	assert.True(t, discovery.Resources[0].Namespaced)
	assert.Equal(t, "MetricValueList", discovery.Resources[0].Kind)
	assert.Equal(t, []string{"get"}, discovery.Resources[0].Verbs)
}

// ****************
// RegisterCustomMetricsAPI
// ****************

// registeringWebhookServer records the paths the custom metrics API is served on.
type registeringWebhookServer struct {
	webhook.Server
	handlers map[string]http.Handler
}

func (s *registeringWebhookServer) Register(path string, hook http.Handler) {
	s.handlers[path] = hook
}

// stubManager exposes only the two manager accessors the registration uses.
type stubManager struct {
	manager.Manager
	k8sClient     client.Client
	webhookServer *registeringWebhookServer
}

func (m *stubManager) GetClient() client.Client         { return m.k8sClient }
func (m *stubManager) GetWebhookServer() webhook.Server { return m.webhookServer }

func newRegistrationManager(t *testing.T, k8sClient client.Client) *stubManager {
	t.Helper()

	t.Setenv(consts.CurrentNamespaceEnvVar, gatewayTestNamespace)
	return &stubManager{
		k8sClient:     k8sClient,
		webhookServer: &registeringWebhookServer{handlers: map[string]http.Handler{}},
	}
}

func webhookCertSecret(data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.AutoscalerWebhookSecretName,
			Namespace: gatewayTestNamespace,
		},
		Data: data,
	}
}

func odigosOwnedAPIService(caBundle []byte) *apiregv1.APIService {
	return &apiregv1.APIService{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.CustomMetricsAPIServiceName},
		Spec: apiregv1.APIServiceSpec{
			Service: &apiregv1.ServiceReference{
				Name:      k8sconsts.AutoScalerWebhookServiceName,
				Namespace: gatewayTestNamespace,
			},
			Group:    "custom.metrics.k8s.io",
			Version:  "v1beta1",
			CABundle: caBundle,
		},
	}
}

func storedAPIService(t *testing.T, k8sClient client.Client) *apiregv1.APIService {
	t.Helper()

	stored := &apiregv1.APIService{}
	require.NoError(t, k8sClient.Get(context.Background(),
		client.ObjectKey{Name: k8sconsts.CustomMetricsAPIServiceName}, stored))
	return stored
}

func TestRegisterCustomMetricsAPI_MissingCertSecret(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).Build()

	err := RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get cert secret")
}

func TestRegisterCustomMetricsAPI_CertSecretWithoutCA(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(webhookCertSecret(map[string][]byte{"tls.crt": []byte("leaf")})).Build()

	err := RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ca.crt not found in secret")
}

func TestRegisterCustomMetricsAPI_CreatesTheAPIService(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")})).Build()

	require.NoError(t, RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient)))

	created := storedAPIService(t, k8sClient)
	assert.Equal(t, "custom.metrics.k8s.io", created.Spec.Group)
	assert.Equal(t, "v1beta1", created.Spec.Version)
	assert.Equal(t, []byte("the-ca"), created.Spec.CABundle)
	// the aggregation layer talks to the autoscaler over TLS, so it must verify the served cert
	// against the CA the rotator issued it with.
	assert.False(t, created.Spec.InsecureSkipTLSVerify)
	assert.Equal(t, int32(100), created.Spec.GroupPriorityMinimum)
	assert.Equal(t, int32(100), created.Spec.VersionPriority)
	require.NotNil(t, created.Spec.Service)
	assert.Equal(t, "odigos-autoscaler", created.Spec.Service.Name)
	assert.Equal(t, gatewayTestNamespace, created.Spec.Service.Namespace)
	require.NotNil(t, created.Spec.Service.Port)
	assert.Equal(t, int32(9443), *created.Spec.Service.Port)
}

func TestRegisterCustomMetricsAPI_RefreshesARotatedCA(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
		odigosOwnedAPIService([]byte("expired-ca")),
	).Build()

	require.NoError(t, RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient)))

	assert.Equal(t, []byte("rotated-ca"), storedAPIService(t, k8sClient).Spec.CABundle)
}

func TestRegisterCustomMetricsAPI_UnchangedCAIsNotRewritten(t *testing.T) {
	updates := 0
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")}),
		odigosOwnedAPIService([]byte("the-ca")),
	).WithInterceptorFuncs(interceptor.Funcs{
		Update: func(ctx context.Context, c client.WithWatch, obj client.Object, opts ...client.UpdateOption) error {
			updates++
			return c.Update(ctx, obj, opts...)
		},
	}).Build()

	require.NoError(t, RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient)))

	assert.Zero(t, updates)
}

func TestRegisterCustomMetricsAPI_LeavesAForeignAPIServiceAlone(t *testing.T) {
	// v1beta1.custom.metrics.k8s.io is a cluster-wide singleton; hijacking the one another adapter
	// installed would break that adapter's metrics.
	foreign := odigosOwnedAPIService([]byte("prometheus-adapter-ca"))
	foreign.Spec.Service.Name = "prometheus-adapter"

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("odigos-ca")}),
		foreign,
	).Build()

	require.NoError(t, RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient)))

	stored := storedAPIService(t, k8sClient)
	assert.Equal(t, []byte("prometheus-adapter-ca"), stored.Spec.CABundle)
	assert.Equal(t, "prometheus-adapter", stored.Spec.Service.Name)
}

func TestRegisterCustomMetricsAPI_APIServiceLookupFailure(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")})).
		WithInterceptorFuncs(interceptor.Funcs{
			Get: func(ctx context.Context, c client.WithWatch, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
				if _, ok := obj.(*apiregv1.APIService); ok {
					return errors.New("apiservices is forbidden")
				}
				return c.Get(ctx, key, obj, opts...)
			},
		}).Build()

	err := RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "apiservices is forbidden")
}

func TestRegisterCustomMetricsAPI_CreateFailure(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).
		WithObjects(webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")})).
		WithInterceptorFuncs(interceptor.Funcs{
			Create: func(context.Context, client.WithWatch, client.Object, ...client.CreateOption) error {
				return errors.New("admission denied")
			},
		}).Build()

	err := RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create APIService")
	assert.Contains(t, err.Error(), "admission denied")
}

func TestRegisterCustomMetricsAPI_UpdateFailure(t *testing.T) {
	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("rotated-ca")}),
		odigosOwnedAPIService([]byte("expired-ca")),
	).WithInterceptorFuncs(interceptor.Funcs{
		Update: func(context.Context, client.WithWatch, client.Object, ...client.UpdateOption) error {
			return errors.New("conflict")
		},
	}).Build()

	err := RegisterCustomMetricsAPI(newRegistrationManager(t, k8sClient))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update APIService")
}

func TestRegisterCustomMetricsAPI_ServesDiscoveryAndTheMetricTheHPAAsksFor(t *testing.T) {
	resetRejectionSamples(t)
	serveGatewayOwnTelemetry(t, rejectionsPayload(10))
	rememberRejectionSample("odigos-gateway-1", 1)

	k8sClient := fake.NewClientBuilder().WithScheme(metricsHandlerScheme(t)).WithObjects(
		webhookCertSecret(map[string][]byte{"ca.crt": []byte("the-ca")}),
		gatewayPod("odigos-gateway-1", "127.0.0.1"),
	).Build()
	mgr := newRegistrationManager(t, k8sClient)

	require.NoError(t, RegisterCustomMetricsAPI(mgr))

	// The aggregation layer derives the request path from the discovery document, so the two have
	// to agree exactly or every HPA lookup is a 404 and the gateway never scales on rejections.
	discovery := httptest.NewRecorder()
	DiscoveryHandler(discovery, httptest.NewRequest(http.MethodGet, "/", nil))
	var advertised struct {
		GroupVersion string `json:"groupVersion"`
		Resources    []struct {
			Name string `json:"name"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(discovery.Body.Bytes(), &advertised))

	discoveryPath := "/apis/" + advertised.GroupVersion
	assert.Contains(t, mgr.webhookServer.handlers, discoveryPath)

	metricPath := fmt.Sprintf("/apis/%s/namespaces/%s/%s/%s",
		advertised.GroupVersion, gatewayTestNamespace,
		// "deployments.apps/odigos_gateway_rejections" -> ".../deployments.apps/<deployment>/<metric>"
		strings.Split(advertised.Resources[0].Name, "/")[0]+"/odigos-gateway",
		strings.Split(advertised.Resources[0].Name, "/")[1])
	handler, registered := mgr.webhookServer.handlers[metricPath]
	require.Truef(t, registered, "no handler registered on %s, only %v", metricPath, mgr.webhookServer.handlers)

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, metricPath, nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	var served MetricValueList
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &served))
	require.Len(t, served.Items, 1)
	assert.Equal(t, strings.Split(advertised.Resources[0].Name, "/")[1], served.Items[0].MetricName)
	assert.Equal(t, "odigos-gateway", served.Items[0].DescribedObject["name"])
	assert.Equal(t, "1.00", served.Items[0].Value)
}
