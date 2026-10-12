package metricshandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	cmNamespace = "odigos-test-ns"

	// Spelled out rather than taken from the production identifiers, so that renaming one of them
	// shows up here instead of silently renaming both sides of the assertion.
	cmGatewayDeploymentName = "odigos-gateway"
	cmRejectionCounterName  = "odigos_collector_memory_limiter_batch_rejections_total"
	cmHelmManagedByLabel    = "app.kubernetes.io/managed-by"
)

// MetricHandler keys its delta bookkeeping in the package-global lastSample, which is never
// pruned. Unique pod names per fixture keep every test independent and let the package survive
// `go test -count=N`.
var cmSeq atomic.Int64

func cmScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(s))
	require.NoError(t, apiregv1.AddToScheme(s))
	return s
}

func cmPodName(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("odigos-gateway-%d", cmSeq.Add(1))
}

// cmLoopbackIP hands out a distinct 127.x address per gateway pod so every fixture can run a
// real scrape target on the collector's own-telemetry port without colliding.
func cmLoopbackIP(t *testing.T) string {
	t.Helper()
	n := cmSeq.Add(1)
	return fmt.Sprintf("127.0.%d.%d", 1+(n/250)%250, 1+n%250)
}

func cmGatewayPod(name, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cmNamespace,
			Labels: map[string]string{
				k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
			},
		},
		Status: corev1.PodStatus{PodIP: ip},
	}
}

func cmOOMKilledPod(name, ip string) *corev1.Pod {
	pod := cmGatewayPod(name, ip)
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{{
		State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "OOMKilled"}},
	}}
	return pod
}

// cmServeRejections starts a /metrics endpoint on ip:<own-telemetry-port> serving the gateway
// rejection counter. The returned setter changes the value the next scrape observes.
func cmServeRejections(t *testing.T, ip string, value float64) func(float64) {
	t.Helper()

	addr := fmt.Sprintf("%s:%d", ip, k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("cannot bind %s, the collector own-telemetry port is taken on this machine: %v", addr, err)
	}

	current := value
	srv := &httptest.Server{
		Listener: ln,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprintf(w, "# TYPE %s counter\n%s{exporter=\"otlp\"} %g\n",
				cmRejectionCounterName, cmRejectionCounterName, current)
		})},
	}
	srv.Start()
	t.Cleanup(srv.Close)

	return func(v float64) { current = v }
}

func cmServeBody(t *testing.T, ip string, status int, body string) {
	t.Helper()

	addr := fmt.Sprintf("%s:%d", ip, k8sconsts.OdigosClusterCollectorOwnTelemetryPortDefault)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Skipf("cannot bind %s, the collector own-telemetry port is taken on this machine: %v", addr, err)
	}

	srv := &httptest.Server{
		Listener: ln,
		Config: &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
			fmt.Fprint(w, body)
		})},
	}
	srv.Start()
	t.Cleanup(srv.Close)
}

func cmCall(t *testing.T, handler http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	handler(rec, httptest.NewRequest(http.MethodGet, "/apis/custom.metrics.k8s.io/v1beta1", nil))
	return rec
}

func cmServedValue(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var list MetricValueList
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
	require.Len(t, list.Items, 1)
	return list.Items[0].Value
}

func TestMetricHandler_NoGatewayPodsIsNotFound(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).Build()

	rec := cmCall(t, MetricHandler(context.Background(), c, cmNamespace))

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "no gateway pods found")
}

func TestMetricHandler_ListErrorIsServerError(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithInterceptorFuncs(interceptor.Funcs{
		List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
			return errors.New("the gateway pod list is unavailable")
		},
	}).Build()

	rec := cmCall(t, MetricHandler(context.Background(), c, cmNamespace))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "the gateway pod list is unavailable")
}

// The handler must look only at cluster-gateway pods in its own namespace: an OOMKilled node
// collector, or a gateway in another namespace, would otherwise scale this gateway out.
func TestMetricHandler_OnlyGatewayPodsInOwnNamespaceCount(t *testing.T) {
	healthy := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))

	otherNamespace := cmOOMKilledPod(cmPodName(t), cmLoopbackIP(t))
	otherNamespace.Namespace = "some-other-namespace"

	otherRole := cmOOMKilledPod(cmPodName(t), cmLoopbackIP(t))
	otherRole.Labels[k8sconsts.OdigosCollectorRoleLabel] = string(k8sconsts.CollectorsRoleNodeCollector)

	unlabeled := cmOOMKilledPod(cmPodName(t), cmLoopbackIP(t))
	unlabeled.Labels = nil

	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).
		WithObjects(healthy, otherNamespace, otherRole, unlabeled).Build()
	cmServeRejections(t, healthy.Status.PodIP, 0)

	value := cmServedValue(t, cmCall(t, MetricHandler(context.Background(), c, cmNamespace)))

	assert.Equal(t, "0.00", value, "out-of-scope OOMKilled pods must not raise the gateway metric")
}

// The first scrape of a pod has no previous sample, so a gateway that has been rejecting since
// before the instrumentor started must not immediately report pressure.
func TestMetricHandler_FirstSampleIsNeverRejecting(t *testing.T) {
	pod := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pod).Build()
	cmServeRejections(t, pod.Status.PodIP, 9999)

	handler := MetricHandler(context.Background(), c, cmNamespace)

	assert.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)),
		"a pod observed for the first time has no delta and cannot be rejecting")
	assert.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)),
		"an unchanged counter is still not a rejection")
}

// The metric is binary: 1 when at least half of the gateway pods rejected data since the previous
// scrape, 0 otherwise. The HPA object metric targets 500m, so the 50% boundary is what decides
// whether the gateway scales out.
func TestMetricHandler_RejectingRatioThreshold(t *testing.T) {
	tests := []struct {
		name string
		// deltas[i] is how much pod i's rejection counter moves between the two scrapes.
		deltas []float64
		want   string
	}{
		{name: "single pod idle", deltas: []float64{0}, want: "0.00"},
		{name: "single pod rejecting", deltas: []float64{1}, want: "1.00"},
		{name: "one of two is exactly the threshold", deltas: []float64{7, 0}, want: "1.00"},
		{name: "one of three is below the threshold", deltas: []float64{7, 0, 0}, want: "0.00"},
		{name: "two of three is above the threshold", deltas: []float64{7, 3, 0}, want: "1.00"},
		{name: "two of four is exactly the threshold", deltas: []float64{1, 1, 0, 0}, want: "1.00"},
		{name: "a fractional increase still counts", deltas: []float64{0.5, 0}, want: "1.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pods := make([]client.Object, 0, len(tt.deltas))
			setters := make([]func(float64), 0, len(tt.deltas))
			for range tt.deltas {
				pod := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
				pods = append(pods, pod)
				setters = append(setters, cmServeRejections(t, pod.Status.PodIP, 100))
			}

			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pods...).Build()
			handler := MetricHandler(context.Background(), c, cmNamespace)

			require.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)), "priming scrape")
			for i, delta := range tt.deltas {
				setters[i](100 + delta)
			}

			assert.Equal(t, tt.want, cmServedValue(t, cmCall(t, handler)))
		})
	}
}

// A gateway pod that restarts resets its counter to 0. Treating that as a negative delta would be
// fine, but treating the next increase from 0 as a huge delta must not be lost either.
func TestMetricHandler_CounterResetIsNotARejection(t *testing.T) {
	pod := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pod).Build()
	set := cmServeRejections(t, pod.Status.PodIP, 500)
	handler := MetricHandler(context.Background(), c, cmNamespace)

	require.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)), "priming scrape")

	set(0)
	assert.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)), "a counter reset is not a rejection")

	set(4)
	assert.Equal(t, "1.00", cmServedValue(t, cmCall(t, handler)),
		"the baseline must follow the reset, so the next increase is still observed")
}

// An OOMKilled gateway is the case the custom metric exists for: it is not scraping-reachable and
// reports nothing to the metrics server, so it has to count as rejecting without a scrape.
func TestMetricHandler_OOMKilledPodCountsWithoutScraping(t *testing.T) {
	oom := cmOOMKilledPod(cmPodName(t), cmLoopbackIP(t))
	healthy := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))

	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(oom, healthy).Build()
	// Deliberately no server on the OOMKilled pod's IP.
	cmServeRejections(t, healthy.Status.PodIP, 0)

	value := cmServedValue(t, cmCall(t, MetricHandler(context.Background(), c, cmNamespace)))

	assert.Equal(t, "1.00", value, "1 of 2 gateway pods OOMKilled is the 50%% threshold")
}

// Unreachable pods are skipped for scraping but still belong to the fleet, so they must dilute the
// ratio rather than disappear from it.
func TestMetricHandler_UnreachablePodsStayInTheDenominator(t *testing.T) {
	tests := []struct {
		name        string
		unreachable int
		want        string
	}{
		{name: "one rejecting one unreachable is the threshold", unreachable: 1, want: "1.00"},
		{name: "one rejecting three unreachable is below it", unreachable: 3, want: "0.00"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rejecting := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
			pods := []client.Object{rejecting}
			for i := 0; i < tt.unreachable; i++ {
				pods = append(pods, cmGatewayPod(cmPodName(t), cmLoopbackIP(t)))
			}

			c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pods...).Build()
			set := cmServeRejections(t, rejecting.Status.PodIP, 10)
			handler := MetricHandler(context.Background(), c, cmNamespace)

			require.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)), "priming scrape")
			set(11)

			assert.Equal(t, tt.want, cmServedValue(t, cmCall(t, handler)))
		})
	}
}

// The HPA addresses the metric through describedObject, so every field of the envelope is part of
// the contract with the custom metrics API server.
func TestMetricHandler_ResponseEnvelope(t *testing.T) {
	pod := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
	c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pod).Build()
	cmServeRejections(t, pod.Status.PodIP, 0)

	before := time.Now()
	rec := cmCall(t, MetricHandler(context.Background(), c, cmNamespace))
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var list MetricValueList
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))

	assert.Equal(t, "custom.metrics.k8s.io/v1beta1", list.APIVersion)
	assert.Equal(t, "MetricValueList", list.Kind)
	require.Len(t, list.Items, 1)

	item := list.Items[0]
	assert.Equal(t, "odigos_gateway_rejections", item.MetricName)
	assert.Equal(t, cmAdvertisedMetricName(t), item.MetricName,
		"the served metric must be the one discovery advertises")
	assert.Equal(t, map[string]string{
		"kind":      "Deployment",
		"namespace": cmNamespace,
		"name":      cmGatewayDeploymentName,
	}, item.DescribedObject)
	assert.Equal(t, cmGatewayDeploymentName, k8sconsts.OdigosClusterCollectorDeploymentName,
		"the route RegisterCustomMetricsAPI serves is built from this literal")
	assert.False(t, item.Timestamp.Before(before), "the sample must be stamped at scrape time")
	assert.False(t, item.Timestamp.After(time.Now()))
}

func TestIsPodOOMKilled(t *testing.T) {
	t.Parallel()

	waiting := func(reason string) corev1.ContainerState {
		return corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}}
	}
	terminated := func(reason string) corev1.ContainerState {
		return corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: reason}}
	}

	tests := []struct {
		name     string
		statuses []corev1.ContainerStatus
		want     bool
	}{
		{name: "no container statuses", want: false},
		{name: "running", statuses: []corev1.ContainerStatus{{
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		}}, want: false},
		{name: "terminated by OOM", statuses: []corev1.ContainerStatus{{
			State: terminated("OOMKilled"),
		}}, want: true},
		{name: "terminated by something else", statuses: []corev1.ContainerStatus{{
			State: terminated("Error"),
		}}, want: false},
		{name: "crash looping after an OOM kill", statuses: []corev1.ContainerStatus{{
			State:                waiting("CrashLoopBackOff"),
			LastTerminationState: terminated("OOMKilled"),
		}}, want: true},
		{name: "crash looping after a non-OOM exit", statuses: []corev1.ContainerStatus{{
			State:                waiting("CrashLoopBackOff"),
			LastTerminationState: terminated("Error"),
		}}, want: false},
		{name: "waiting for another reason after an OOM kill", statuses: []corev1.ContainerStatus{{
			State:                waiting("ImagePullBackOff"),
			LastTerminationState: terminated("OOMKilled"),
		}}, want: false},
		{name: "crash looping with no previous termination", statuses: []corev1.ContainerStatus{{
			State: waiting("CrashLoopBackOff"),
		}}, want: false},
		{name: "a later container is the one OOMKilled", statuses: []corev1.ContainerStatus{
			{State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
			{State: terminated("OOMKilled")},
		}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			pod := &corev1.Pod{Status: corev1.PodStatus{ContainerStatuses: tt.statuses}}
			assert.Equal(t, tt.want, isPodOOMKilled(pod))
		})
	}
}

func TestScrapeGatewayMetric(t *testing.T) {
	counter := func(samples ...string) string {
		var b strings.Builder
		fmt.Fprintf(&b, "# TYPE %s counter\n", cmRejectionCounterName)
		for _, s := range samples {
			fmt.Fprintf(&b, "%s%s\n", cmRejectionCounterName, s)
		}
		return b.String()
	}

	tests := []struct {
		name    string
		status  int
		body    string
		want    float64
		wantErr string
	}{
		{
			name:   "sums every label set of the rejection counter",
			status: http.StatusOK,
			body:   counter(`{exporter="otlp"} 3`, `{exporter="otlp/insights"} 4`),
			want:   7,
		},
		{
			name:   "an absent metric is zero rejections, not a failure",
			status: http.StatusOK,
			body:   "# TYPE otelcol_process_uptime counter\notelcol_process_uptime 12\n",
			want:   0,
		},
		{
			name:   "ignores a family that is not a counter",
			status: http.StatusOK,
			body:   fmt.Sprintf("# TYPE %s gauge\n%s 5\n", cmRejectionCounterName, cmRejectionCounterName),
			want:   0,
		},
		{
			name:    "a non-200 response is a failure",
			status:  http.StatusServiceUnavailable,
			body:    counter(` 3`),
			wantErr: "unexpected status code: 503",
		},
		{
			name:    "unparsable exposition text is a failure",
			status:  http.StatusOK,
			body:    "this is not prometheus exposition format {{{\n",
			wantErr: "failed to parse metrics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := cmLoopbackIP(t)
			cmServeBody(t, ip, tt.status, tt.body)

			got, err := scrapeGatewayMetric(ip)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestScrapeGatewayMetric_UnreachablePod(t *testing.T) {
	_, err := scrapeGatewayMetric(cmLoopbackIP(t))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to reach pod")
}

// The aggregated API server reads this document to decide which metrics it may forward, so the
// resource it advertises has to be the one MetricHandler actually serves.
func TestDiscoveryHandler(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	DiscoveryHandler(rec, httptest.NewRequest(http.MethodGet, "/apis/custom.metrics.k8s.io/v1beta1", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var doc map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))

	assert.Equal(t, "APIResourceList", doc["kind"])
	assert.Equal(t, "v1", doc["apiVersion"])
	assert.Equal(t, "custom.metrics.k8s.io/v1beta1", doc["groupVersion"])

	resources, ok := doc["resources"].([]any)
	require.True(t, ok)
	require.Len(t, resources, 1)

	assert.Equal(t, map[string]any{
		"name":         "deployments.apps/odigos_gateway_rejections",
		"singularName": "",
		"namespaced":   true,
		"kind":         "MetricValueList",
		"verbs":        []any{"get"},
	}, resources[0])
}

// k8sconsts.CustomMetricsAPIServiceName is the object the CA sync and the delete migration act on.
// An APIService object is named "<version>.<group>", so it has to spell out exactly the group
// version this package serves - otherwise both controllers quietly manage the wrong object.
func TestAPIServiceNameMatchesTheServedGroupVersion(t *testing.T) {
	t.Parallel()

	group, version := cmServedGroupVersion(t)

	assert.Equal(t, version+"."+group, k8sconsts.CustomMetricsAPIServiceName)
}

// cmSplitAdvertisedResource reads back the single resource the discovery document advertises,
// which the custom metrics API server reads as "<resource type>/<metric name>".
func cmSplitAdvertisedResource(t *testing.T) (resourceType, metricName string) {
	t.Helper()

	rec := httptest.NewRecorder()
	DiscoveryHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	var doc struct {
		Resources []struct {
			Name string `json:"name"`
		} `json:"resources"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))
	require.Len(t, doc.Resources, 1)

	resourceType, metricName, found := strings.Cut(doc.Resources[0].Name, "/")
	require.True(t, found, "an advertised resource must be <resource type>/<metric name>, got %q", doc.Resources[0].Name)
	return resourceType, metricName
}

func cmServedGroupVersion(t *testing.T) (group, version string) {
	t.Helper()

	rec := httptest.NewRecorder()
	DiscoveryHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	var doc struct {
		GroupVersion string `json:"groupVersion"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &doc))

	group, version, found := strings.Cut(doc.GroupVersion, "/")
	require.True(t, found, "groupVersion must be <group>/<version>, got %q", doc.GroupVersion)
	return group, version
}

// The gateway HPA reads a binary signal and compares it against a 500m object-metric target
// (see buildv2Metrics in the clustercollector package). Serving anything other than 0 or 1 - a raw
// ratio, say - would silently move the point at which the gateway scales out.
func TestMetricHandler_ServesOnlyTheTwoValuesTheHPATargetSitsBetween(t *testing.T) {
	const hpaTargetMilliValue = 500

	served := map[string]bool{}
	for _, rejecting := range []bool{false, true} {
		pod := cmGatewayPod(cmPodName(t), cmLoopbackIP(t))
		c := fake.NewClientBuilder().WithScheme(cmScheme(t)).WithObjects(pod).Build()
		set := cmServeRejections(t, pod.Status.PodIP, 1)
		handler := MetricHandler(context.Background(), c, cmNamespace)

		require.Equal(t, "0.00", cmServedValue(t, cmCall(t, handler)), "priming scrape")
		if rejecting {
			set(2)
		}
		served[cmServedValue(t, cmCall(t, handler))] = true
	}

	require.Len(t, served, 2, "the metric must be binary, got %v", served)

	target := float64(hpaTargetMilliValue) / 1000
	idle, err := strconv.ParseFloat(cmSortedKeys(served)[0], 64)
	require.NoError(t, err)
	rejecting, err := strconv.ParseFloat(cmSortedKeys(served)[1], 64)
	require.NoError(t, err)

	assert.Less(t, idle, target, "the idle value must leave the HPA below its target")
	assert.Greater(t, rejecting, target, "the rejecting value must push the HPA over its target")
}

func cmSortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
