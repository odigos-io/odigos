package tracecorrelations

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseExportFirstSeen(t *testing.T) {
	body := strings.Join([]string{
		`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default","k8s.container.name":"app","k8s.deployment.name":"checkout","input.http.route":"/login","output.db.system":"postgresql","odigos.collector.instance.id":"a"},"timestamps":[1781332967250,1781332977250]}`,
		`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default","k8s.container.name":"app","k8s.deployment.name":"checkout","input.http.route":"/login","output.db.system":"postgresql","odigos.collector.instance.id":"b"},"timestamps":[1781332900000]}`,
	}, "\n")

	firstSeen, err := parseExportFirstSeen(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, firstSeen, 1)

	for _, ts := range firstSeen {
		require.Equal(t, time.UnixMilli(1781332900000).UTC(), ts)
	}
}

func TestQueryFirstSeenFromExport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/export", r.URL.Path)
		require.Equal(t, metricSelector, r.URL.Query().Get("match[]"))
		require.NotEmpty(t, r.URL.Query().Get("start"))
		require.NotEmpty(t, r.URL.Query().Get("end"))

		_, _ = w.Write([]byte(`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default","k8s.container.name":"app","k8s.deployment.name":"checkout","input.http.route":"/login","output.db.system":"postgresql"},"timestamps":[1781332967250]}`))
	}))
	defer server.Close()

	firstSeen, err := queryFirstSeenFromExport(context.Background(), server.URL, time.Now().Add(-exportLookback), time.Now())
	require.NoError(t, err)
	require.Len(t, firstSeen, 1)
}

func TestEarliestExportTimestamp(t *testing.T) {
	require.Equal(t, time.UnixMilli(1781332900000).UTC(), earliestExportTimestamp([]int64{1781332967250, 1781332900000}))
	require.True(t, earliestExportTimestamp(nil).IsZero())
}
