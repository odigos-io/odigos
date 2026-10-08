package tracesurge

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestInsightsArchivePut(t *testing.T) {
	var path string
	var body map[string]any
	status := http.StatusNoContent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		path = r.URL.EscapedPath()
		raw, _ := io.ReadAll(r.Body)
		require.NoError(t, json.Unmarshal(raw, &body))
		w.WriteHeader(status)
	}))
	defer server.Close()
	archive := &insightsArchive{baseURL: server.URL, client: server.Client()}

	started := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	restored := metav1.NewTime(started.Add(5 * time.Minute))
	surge := &odigosv1.TraceSurge{
		Name:      "abc-payments-1",
		StartedAt: metav1.NewTime(started),
		Spec: odigosv1.TraceSurgeSpec{SamplingName: "default", RuleID: "abc",
			Service: k8sconsts.PodWorkload{Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment, Name: "payments"}},
		Status: odigosv1.TraceSurgeStatus{Phase: odigosv1.TraceSurgePhaseRestored, RestoredAt: &restored,
			Timeline: []odigosv1.TraceSurgeEvent{{At: restored, Title: "Sampling back at 1%"}}},
	}
	require.NoError(t, archive.put(context.Background(), surge))
	assert.Equal(t, "/api/v1/trace-surges/abc-payments-1", path)
	assert.Equal(t, "abc-payments-1", body["name"])
	assert.Equal(t, "abc", body["rule_id"])
	assert.Equal(t, "shop", body["namespace"])
	assert.Equal(t, "Deployment", body["workload_kind"])
	assert.Equal(t, "payments", body["workload_name"])
	assert.Equal(t, "2026-10-06T10:00:00Z", body["started_at"])
	assert.Equal(t, "2026-10-06T10:05:00Z", body["ended_at"], "it ended when sampling was restored")
	record, _ := body["record"].(map[string]any)
	assert.Equal(t, "abc-payments-1", record["name"], "the record is the surge as the status holds it")

	later := metav1.NewTime(restored.Add(time.Minute))
	surge.Status.Timeline = append(surge.Status.Timeline, odigosv1.TraceSurgeEvent{At: later, Title: "Rule updated"})
	require.NoError(t, archive.put(context.Background(), surge))
	assert.Equal(t, "2026-10-06T10:06:00Z", body["ended_at"], "an event after the restore is the last change")

	status = http.StatusInternalServerError
	assert.Error(t, archive.put(context.Background(), surge))
}
