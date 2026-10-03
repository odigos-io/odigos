package diagnose

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestFetchWorkloadLogsCollectsEveryContainerOfEveryPod(t *testing.T) {
	pods := []corev1.Pod{
		*dgPod(dgNamespace, "odiglet-1", nil, "odiglet", "data-collection"),
		*dgPod(dgNamespace, "odiglet-2", nil, "odiglet"),
	}
	pods[0].Spec.InitContainers = []corev1.Container{{Name: "odiglet-init"}}
	builder := newDgBuilder()

	require.NoError(t, FetchWorkloadLogs(context.Background(), dgClientset(), builder, dgNamespace, "dir", pods))

	assert.Equal(t, []string{
		"dir/pod-odiglet-1.data-collection.log.gz",
		"dir/pod-odiglet-1.odiglet-init.log.gz",
		"dir/pod-odiglet-1.odiglet.log.gz",
		"dir/pod-odiglet-2.odiglet.log.gz",
	}, builder.gzippedPaths(), "an init container that crashed is often the whole story, so it is collected too")
}

// A container that restarted has already lost the logs that explain why; the previous
// container's logs are the only copy.
func TestFetchWorkloadLogsAlsoCollectsPreviousLogsOnlyForRestartedContainers(t *testing.T) {
	pod := dgPod(dgNamespace, "odiglet-1", nil, "odiglet", "data-collection")
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "odiglet", RestartCount: 3},
		{Name: "data-collection", RestartCount: 0},
	}
	builder := newDgBuilder()

	require.NoError(t, FetchWorkloadLogs(context.Background(), dgClientset(), builder, dgNamespace, "dir", []corev1.Pod{*pod}))

	assert.Equal(t, []string{
		"dir/pod-odiglet-1.data-collection.log.gz",
		"dir/pod-odiglet-1.odiglet.log.gz",
		"dir/pod-odiglet-1.odiglet.previous.log.gz",
	}, builder.gzippedPaths())
}

func TestFetchWorkloadLogsIgnoresAStatusForAContainerThatIsGone(t *testing.T) {
	pod := dgPod(dgNamespace, "odiglet-1", nil, "odiglet")
	pod.Status.ContainerStatuses = []corev1.ContainerStatus{
		{Name: "removed-sidecar", RestartCount: 5},
	}
	builder := newDgBuilder()

	require.NoError(t, FetchWorkloadLogs(context.Background(), dgClientset(), builder, dgNamespace, "dir", []corev1.Pod{*pod}))

	assert.Equal(t, []string{"dir/pod-odiglet-1.odiglet.log.gz"}, builder.gzippedPaths())
}

// The support engineer reading the bundle must be able to tell "this container produced
// no logs" from "we were not allowed to read them", so the failure is archived as the
// log file's content.
func TestAddContainerLogsRecordsWhyAStreamCouldNotBeRead(t *testing.T) {
	client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"kind":"Status","apiVersion":"v1","status":"Failure","code":403,"reason":"Forbidden","message":"pods/log is forbidden"}`))
	})
	builder := newDgBuilder()

	addContainerLogs(context.Background(), client, builder, dgNamespace, "dir", "odiglet-1", "odiglet", false)

	require.Equal(t, []string{"dir/pod-odiglet-1.odiglet.log.gz"}, builder.paths())
	assert.Contains(t, string(builder.body(t, "dir/pod-odiglet-1.odiglet.log.gz")), "Error fetching logs:")
	assert.Contains(t, string(builder.body(t, "dir/pod-odiglet-1.odiglet.log.gz")), "pods/log is forbidden")
	assert.Empty(t, builder.gzippedPaths(), "an error message is small and is stored uncompressed")
}

func TestAddContainerLogsSurvivesBeingUnableToRecordTheFailure(t *testing.T) {
	client := dgRESTClientset(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	builder := newDgBuilder()
	builder.failOn = func(string, string) error { return assert.AnError }

	addContainerLogs(context.Background(), client, builder, dgNamespace, "dir", "odiglet-1", "odiglet", false)

	assert.Empty(t, builder.paths())
}

func TestAddContainerLogsSurvivesAFailureToArchiveTheStream(t *testing.T) {
	builder := newDgBuilder()
	builder.failOn = func(string, string) error { return assert.AnError }

	addContainerLogs(context.Background(), dgClientset(), builder, dgNamespace, "dir", "odiglet-1", "odiglet", false)

	assert.Empty(t, builder.paths())
}

func TestAddContainerLogsRequestsTheContainerAndRevisionItWasAskedFor(t *testing.T) {
	var queries []string
	client := dgRESTClientset(t, func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Path+"?"+r.URL.Query().Encode())
		_, _ = w.Write([]byte("log body"))
	})
	builder := newDgBuilder()

	addContainerLogs(context.Background(), client, builder, dgNamespace, "dir", "odiglet-1", "data-collection", true)

	require.Len(t, queries, 1)
	assert.Equal(t, "/api/v1/namespaces/odigos-system/pods/odiglet-1/log?container=data-collection&previous=true", queries[0])
	assert.Equal(t, []byte("log body"), builder.body(t, "dir/pod-odiglet-1.data-collection.previous.log.gz"))
}

func TestFetchWorkloadLogsWithNoPods(t *testing.T) {
	builder := newDgBuilder()
	require.NoError(t, FetchWorkloadLogs(context.Background(), dgClientset(), builder, dgNamespace, "dir", nil))
	assert.Empty(t, builder.paths())
}
