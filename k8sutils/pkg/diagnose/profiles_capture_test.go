package diagnose

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8stesting "k8s.io/client-go/testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// A query string embedded in AbsPath is escaped into the path and never reaches the
// apiserver, so a CPU profile requested that way returns the 30-second default instead
// of the 10 seconds asked for. The query has to travel as request parameters.
func TestCaptureProfileSendsItsQueryAsRequestParametersNotInThePath(t *testing.T) {
	for _, tc := range []struct {
		profiler      ProfileInterface
		expectedPath  string
		expectedQuery string
	}{
		{CPUProfiler{}, "/api/v1/namespaces/odigos-system/pods/odiglet-1:6060/proxy/debug/pprof/profile", "seconds=10"},
		{HeapProfiler{}, "/api/v1/namespaces/odigos-system/pods/odiglet-1:6060/proxy/debug/pprof/heap", ""},
		{GoRoutineProfiler{}, "/api/v1/namespaces/odigos-system/pods/odiglet-1:6060/proxy/debug/pprof/goroutine", ""},
		{AllocsProfiler{}, "/api/v1/namespaces/odigos-system/pods/odiglet-1:6060/proxy/debug/pprof/allocs", ""},
	} {
		t.Run(tc.profiler.GetFileName(), func(t *testing.T) {
			var gotPath, gotQuery string
			client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
				_, _ = w.Write([]byte("pprof payload"))
			})

			data, err := captureProfile(context.Background(), client, "odiglet-1",
				k8sconsts.DefaultPprofEndpointPort, dgNamespace, tc.profiler)

			require.NoError(t, err)
			assert.Equal(t, "pprof payload", string(data))
			assert.Equal(t, tc.expectedPath, gotPath)
			assert.Equal(t, tc.expectedQuery, gotQuery)
			assert.NotContains(t, gotPath, "%3F", "an escaped '?' means the query was swallowed by the path")
		})
	}
}

// Each profiler contributes one file name and one pprof endpoint; a pair swapped between
// two of them silently mislabels every profile in the bundle.
func TestProfileTypesPairTheirFileNameWithTheirOwnEndpoint(t *testing.T) {
	expected := map[string]string{
		"cpu_profile.prof":       "/profile?seconds=10",
		"heap_profile.prof":      "/heap",
		"goroutine_profile.prof": "/goroutine",
		"allocs_profile.prof":    "/allocs",
	}

	collected := map[string]string{}
	for _, profiler := range ProfilingMetricsFunctions {
		collected[profiler.GetFileName()] = profiler.GetUrlSuffix()
	}

	assert.Equal(t, expected, collected)
	assert.Len(t, ProfilingMetricsFunctions, len(expected), "a profile type was added without a file name of its own")
}

type dgBadQueryProfiler struct{}

func (dgBadQueryProfiler) GetFileName() string  { return "broken_profile.prof" }
func (dgBadQueryProfiler) GetUrlSuffix() string { return "/profile?seconds=%zz" }

func TestCaptureProfileRejectsAQueryItCannotParse(t *testing.T) {
	client := dgRESTClientset(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("unreachable"))
	})

	_, err := captureProfile(context.Background(), client, "odiglet-1",
		k8sconsts.DefaultPprofEndpointPort, dgNamespace, dgBadQueryProfiler{})

	assert.ErrorContains(t, err, `parse pprof query "seconds=%zz"`)
}

func TestFetchServiceProfilesFilesEveryProfileUnderThePodNodeAndServiceDirectory(t *testing.T) {
	pod := dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui")
	pod.Spec.NodeName = "worker-3"
	server := &dgProxyServer{pods: []corev1.Pod{*pod}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()
	profileDir := GetProfileDir(dgRootDir, dgNamespace)

	require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, profileDir, dgNamespace, "ui"))

	podDir := profileDir + "/odigos-ui-1-worker-3-ui"
	assert.Equal(t, []string{
		podDir + "/allocs_profile.prof",
		podDir + "/cpu_profile.prof",
		podDir + "/goroutine_profile.prof",
		podDir + "/heap_profile.prof",
	}, builder.paths())
	assert.Equal(t, "telemetry from odigos-ui-1:6060", string(builder.body(t, podDir+"/heap_profile.prof")))
}

// The odiglet pod carries both the odiglet name label and the node-collector role label
// because it runs both processes, each with its own pprof port. Every other service has
// its own pods. A selector or port swapped between two services profiles the wrong
// process and the bundle looks complete while holding the wrong data.
func TestEachProfiledServiceReadsItsOwnPodsOnItsOwnPort(t *testing.T) {
	odigletPod := dgPod(dgNamespace, "odiglet-1", map[string]string{
		"app.kubernetes.io/name":           k8sconsts.OdigletAppLabelValue,
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}, k8sconsts.OdigletContainerName, k8sconsts.OdigosNodeCollectorContainerName)
	gatewayPod := dgGatewayPod("odigos-gateway-1")
	uiPod := dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui")
	pods := []corev1.Pod{*odigletPod, *gatewayPod, *uiPod}

	for _, tc := range []struct {
		service  string
		pod      string
		port     string
		selector string
	}{
		{"odiglet", "odiglet-1", "6060", "app.kubernetes.io/name=odiglet"},
		{"data-collection", "odiglet-1", "1777", "odigos.io/collector-role=NODE_COLLECTOR"},
		{"gateway", "odigos-gateway-1", "1777", "odigos.io/collector-role=CLUSTER_GATEWAY"},
		{"ui", "odigos-ui-1", "6060", "app=odigos-ui"},
	} {
		t.Run(tc.service, func(t *testing.T) {
			server := &dgProxyServer{pods: pods}
			client := dgRESTClientset(t, server.handler(t))
			builder := newDgBuilder()

			require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, "Profile", dgNamespace, tc.service))

			assert.Equal(t, []string{tc.selector}, server.selectors())
			assert.Equal(t, []string{
				tc.pod + ":" + tc.port + "/debug/pprof/allocs",
				tc.pod + ":" + tc.port + "/debug/pprof/goroutine",
				tc.pod + ":" + tc.port + "/debug/pprof/heap",
				tc.pod + ":" + tc.port + "/debug/pprof/profile?seconds=10",
			}, server.proxyReads())
			assert.Equal(t, []string{
				"Profile/" + tc.pod + "-node-a-" + tc.service + "/allocs_profile.prof",
				"Profile/" + tc.pod + "-node-a-" + tc.service + "/cpu_profile.prof",
				"Profile/" + tc.pod + "-node-a-" + tc.service + "/goroutine_profile.prof",
				"Profile/" + tc.pod + "-node-a-" + tc.service + "/heap_profile.prof",
			}, builder.paths())
		})
	}
}

// A dry run only needs to know the pods exist: capturing a 10-second CPU profile from
// every component just to throw the bytes away would make the size estimate slower than
// the collection it is estimating.
func TestFetchServiceProfilesDryRunConfirmsThePodsWithoutCapturing(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := NewDryRunBuilder()

	require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, "Profile", dgNamespace, "ui"))

	assert.Equal(t, []string{"app=odigos-ui"}, server.selectors())
	assert.Empty(t, server.proxyReads())
	assert.Equal(t, BuilderStats{}, builder.GetStats())
}

func TestFetchServiceProfilesKeepsGoingWhenOneProfileCannotBeWritten(t *testing.T) {
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"),
	}}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()
	builder.failOn = func(_, filename string) error {
		if filename == (CPUProfiler{}).GetFileName() {
			return fmt.Errorf("no space left on device")
		}
		return nil
	}

	require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, "Profile", dgNamespace, "ui"))

	assert.Equal(t, []string{
		"Profile/odigos-ui-1-node-a-ui/allocs_profile.prof",
		"Profile/odigos-ui-1-node-a-ui/goroutine_profile.prof",
		"Profile/odigos-ui-1-node-a-ui/heap_profile.prof",
	}, builder.paths())
}

func TestFetchServiceProfilesDoesNothingWhenTheServiceHasNoPods(t *testing.T) {
	server := &dgProxyServer{}
	client := dgRESTClientset(t, server.handler(t))
	builder := newDgBuilder()

	require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, "Profile", dgNamespace, "gateway"))

	assert.Empty(t, server.proxyReads())
	assert.Empty(t, builder.paths())
}

func TestFetchServiceProfilesReportsWhyItCouldNotListPods(t *testing.T) {
	client := dgClientset()
	client.PrependReactor("list", "pods", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", assert.AnError)
	})

	err := FetchServiceProfiles(context.Background(), client, newDgBuilder(), "Profile", dgNamespace, "odiglet")

	assert.ErrorContains(t, err, "failed to list pods for service odiglet")
}

// Capture retries three times with a five second pause. When collection has been
// canceled the pause must not be waited out, or a canceled diagnose keeps a UI request
// and its client-go rate limiter busy for a minute per pod.
func TestFetchServiceProfilesStopsRetryingOnceCollectionIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/proxy/") {
			cancel()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		dgWritePodList(t, w, *dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"))
	})
	builder := newDgBuilder()

	start := time.Now()
	require.NoError(t, FetchServiceProfiles(ctx, client, builder, "Profile", dgNamespace, "ui"))

	assert.Less(t, time.Since(start), 2*time.Second, "the five second retry pause must be abandoned on cancellation")
	assert.Empty(t, builder.paths())
}

func TestFetchServiceProfilesRetriesAFailedCaptureAndKeepsTheSuccess(t *testing.T) {
	var attempts int32
	server := &dgProxyServer{pods: []corev1.Pod{
		*dgPod(dgNamespace, "odigos-ui-1", map[string]string{"app": k8sconsts.UIAppLabelValue}, "ui"),
	}}
	client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/debug/pprof/goroutine") && atomic.AddInt32(&attempts, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		server.handler(t)(w, r)
	})
	builder := newDgBuilder()

	require.NoError(t, FetchServiceProfiles(context.Background(), client, builder, "Profile", dgNamespace, "ui"))

	assert.Contains(t, builder.paths(), "Profile/odigos-ui-1-node-a-ui/goroutine_profile.prof")
}
