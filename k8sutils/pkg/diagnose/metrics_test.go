package diagnose

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// dgProxyServer answers the two reads metrics collection performs: listing pods by role
// and reading a pod's own-telemetry endpoint through the apiserver proxy. Every proxy
// response names the pod and port it came from so a test can prove which endpoint ended
// up in which file.
type dgProxyServer struct {
	pods []corev1.Pod
	// unreachable makes one "<pod>:<port>" target fail, as an unstarted collector does.
	unreachable string
	failList    string

	mu            sync.Mutex
	listSelectors []string
	proxied       []string
}

func (s *dgProxyServer) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pods") {
			selector := r.URL.Query().Get("labelSelector")
			s.mu.Lock()
			s.listSelectors = append(s.listSelectors, selector)
			s.mu.Unlock()
			if s.failList != "" && selector == s.failList {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			dgWritePodList(t, w, dgSelectPods(t, s.pods, selector)...)
			return
		}

		target := dgProxyTarget(r.URL.Path)
		read := target + r.URL.Path[strings.Index(r.URL.Path, "/proxy/")+len("/proxy"):]
		if r.URL.RawQuery != "" {
			read += "?" + r.URL.RawQuery
		}
		s.mu.Lock()
		s.proxied = append(s.proxied, read)
		s.mu.Unlock()

		if target == s.unreachable {
			http.Error(w, "connection refused", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("telemetry from " + target))
	}
}

func (s *dgProxyServer) selectors() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]string(nil), s.listSelectors...)
	sort.Strings(out)
	return out
}

func (s *dgProxyServer) proxyReads() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]string(nil), s.proxied...)
	sort.Strings(out)
	return out
}

// dgProxyTarget pulls "<pod>:<port>" out of a pod proxy path.
func dgProxyTarget(urlPath string) string {
	segments := strings.Split(urlPath, "/")
	for i, segment := range segments {
		if segment == "pods" && i+1 < len(segments) {
			return segments[i+1]
		}
	}
	return ""
}

func dgNodeCollectorPod(name string) *corev1.Pod {
	return dgPod(dgNamespace, name, map[string]string{
		"app.kubernetes.io/name":           k8sconsts.OdigletAppLabelValue,
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}, k8sconsts.OdigletContainerName, k8sconsts.OdigosNodeCollectorContainerName)
}

func dgGatewayPod(name string) *corev1.Pod {
	return dgPod(dgNamespace, name, map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleClusterGateway),
	}, "gateway")
}

func TestCaptureMetricsReadsThePodsOwnTelemetryThroughTheApiserverProxy(t *testing.T) {
	var paths []string
	client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		_, _ = w.Write([]byte("# HELP otelcol_receiver_accepted_spans\n"))
	})

	data, err := captureMetrics(context.Background(), client, "odiglet-xyz", dgNamespace, 8080)

	require.NoError(t, err)
	assert.Equal(t, "# HELP otelcol_receiver_accepted_spans\n", string(data))
	assert.Equal(t, []string{"/api/v1/namespaces/odigos-system/pods/odiglet-xyz:8080/proxy/metrics"}, paths)
}

func TestCaptureMetricsReportsTheBodyTheApiserverReturned(t *testing.T) {
	// What the apiserver answers when the pod is up but nothing is listening on the port.
	const unreachable = `{"kind":"Status","apiVersion":"v1","status":"Failure",` +
		`"message":"error trying to reach service: dial tcp 10.0.0.1:8080: connect: connection refused",` +
		`"reason":"ServiceUnavailable","code":503}`
	client := dgRESTClientset(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(unreachable))
	})

	_, err := captureMetrics(context.Background(), client, "odiglet-xyz", dgNamespace, 8080)

	require.Error(t, err)
	assert.True(t, apierrors.IsServiceUnavailable(err), "the caller logs this and moves on to the next pod")
	// The raw body is appended after the decoded error, which is where the reason the
	// scrape failed actually shows up.
	assert.ErrorContains(t, err, ": "+unreachable)
}

// The odiglet pod runs two processes that expose telemetry on two different ports, and
// the file suffix and the port are two same-shaped values one line apart.
func TestOdigletPodMetricsAreFiledPerContainerAndPort(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{*dgNodeCollectorPod("odiglet-abc")}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, fetchMetricsFromOdigletPods(context.Background(), client, builder, dgNamespace, "Metrics"))

	assert.Equal(t, []string{"Metrics/odiglet-abc-data-collection", "Metrics/odiglet-abc-odiglet"}, builder.paths())
	assert.Equal(t, "telemetry from odiglet-abc:8080", string(builder.body(t, "Metrics/odiglet-abc-odiglet")),
		"the odiglet container's metrics server")
	assert.Equal(t, "telemetry from odiglet-abc:55682", string(builder.body(t, "Metrics/odiglet-abc-data-collection")),
		"the node collector's own telemetry")
}

func TestOdigletPodMetricsAreScrapedOnlyFromNodeCollectorPods(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgNodeCollectorPod("odiglet-abc"),
		*dgGatewayPod("odigos-gateway-1"),
		*dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, fetchMetricsFromOdigletPods(context.Background(), client, builder, dgNamespace, "Metrics"))

	assert.Equal(t, []string{"odigos.io/collector-role=NODE_COLLECTOR"}, server.selectors())
	assert.Equal(t, []string{"Metrics/odiglet-abc-data-collection", "Metrics/odiglet-abc-odiglet"}, builder.paths())
}

func TestGatewayPodMetricsAreFiledUnderThePodName(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgGatewayPod("odigos-gateway-1"),
		*dgNodeCollectorPod("odiglet-abc"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, fetchMetricsFromGatewayPods(context.Background(), client, builder, dgNamespace, "Metrics"))

	assert.Equal(t, []string{"odigos.io/collector-role=CLUSTER_GATEWAY"}, server.selectors())
	assert.Equal(t, []string{"Metrics/odigos-gateway-1"}, builder.paths())
	assert.Equal(t, "telemetry from odigos-gateway-1:8888", string(builder.body(t, "Metrics/odigos-gateway-1")))
}

func TestFetchOdigosMetricsCollectsBothCollectorTiers(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgNodeCollectorPod("odiglet-abc"),
		*dgGatewayPod("odigos-gateway-1"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()
	dir := GetMetricsDir(dgRootDir, dgNamespace)

	require.NoError(t, FetchOdigosMetrics(context.Background(), client, builder, dir, dgNamespace))

	assert.Equal(t, []string{
		dir + "/odiglet-abc-data-collection",
		dir + "/odiglet-abc-odiglet",
		dir + "/odigos-gateway-1",
	}, builder.paths())
}

func TestFetchOdigosMetricsStillCollectsTheGatewayWhenOdigletPodsCannotBeListed(t *testing.T) {
	server := &dgProxyServer{
		pods:     []corev1.Pod{*dgNodeCollectorPod("odiglet-abc"), *dgGatewayPod("odigos-gateway-1")},
		failList: "odigos.io/collector-role=NODE_COLLECTOR",
	}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosMetrics(context.Background(), client, builder, "Metrics", dgNamespace))

	assert.Equal(t, []string{"Metrics/odigos-gateway-1"}, builder.paths())
}

func TestFetchOdigosMetricsStillCollectsOdigletWhenGatewayPodsCannotBeListed(t *testing.T) {
	server := &dgProxyServer{
		pods:     []corev1.Pod{*dgNodeCollectorPod("odiglet-abc"), *dgGatewayPod("odigos-gateway-1")},
		failList: "odigos.io/collector-role=CLUSTER_GATEWAY",
	}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosMetrics(context.Background(), client, builder, "Metrics", dgNamespace))

	assert.Equal(t, []string{"Metrics/odiglet-abc-data-collection", "Metrics/odiglet-abc-odiglet"}, builder.paths())
}

// Each tier writes its files from its own goroutine; a file that cannot be written must
// not take the rest of the scrape down with it.
func TestMetricsCollectionKeepsGoingWhenAFileCannotBeWritten(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgNodeCollectorPod("odiglet-abc"), *dgGatewayPod("odigos-gateway-1"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()
	builder.failOn = func(_, filename string) error {
		if filename == "odiglet-abc-odiglet" || filename == "odigos-gateway-1" {
			return fmt.Errorf("no space left on device")
		}
		return nil
	}

	require.NoError(t, FetchOdigosMetrics(context.Background(), client, builder, "Metrics", dgNamespace))

	assert.Equal(t, []string{"Metrics/odiglet-abc-data-collection"}, builder.paths())
}

// A collector that is still starting refuses the scrape; the rest of the bundle must not
// be lost over it.
func TestOdigletPodMetricsKeepGoingWhenOnePortRefusesTheScrape(t *testing.T) {
	server := &dgProxyServer{
		pods:        []corev1.Pod{*dgNodeCollectorPod("odiglet-abc")},
		unreachable: "odiglet-abc:55682",
	}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, fetchMetricsFromOdigletPods(context.Background(), client, builder, dgNamespace, "Metrics"))

	assert.Equal(t, []string{"Metrics/odiglet-abc-odiglet"}, builder.paths())
	assert.Equal(t, []string{"odiglet-abc:55682/metrics", "odiglet-abc:8080/metrics"}, server.proxyReads())
}

func TestGatewayPodMetricsKeepGoingWhenOneGatewayRefusesTheScrape(t *testing.T) {
	server := &dgProxyServer{
		pods:        []corev1.Pod{*dgGatewayPod("odigos-gateway-1"), *dgGatewayPod("odigos-gateway-2")},
		unreachable: "odigos-gateway-1:8888",
	}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, fetchMetricsFromGatewayPods(context.Background(), client, builder, dgNamespace, "Metrics"))

	assert.Equal(t, []string{"Metrics/odigos-gateway-2"}, builder.paths())
}

func TestMetricsCollectionScrapesNothingWhenNoCollectorPodsExist(t *testing.T) {
	server := &dgProxyServer{}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, FetchOdigosMetrics(context.Background(), client, builder, "Metrics", dgNamespace))

	assert.Empty(t, builder.paths())
	assert.Empty(t, server.proxyReads())
}
