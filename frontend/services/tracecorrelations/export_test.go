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
		`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default","odigos.collector.instance.id":"a"},"timestamps":[1781332967250,1781332977250]}`,
		`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default","odigos.collector.instance.id":"b"},"timestamps":[1781332900000]}`,
	}, "\n")

	firstSeen, err := parseExportFirstSeen(strings.NewReader(body))
	require.NoError(t, err)
	require.Len(t, firstSeen, 2)

	for _, ts := range firstSeen {
		require.False(t, ts.IsZero())
	}
}

func TestQueryFirstSeenFromExport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/export", r.URL.Path)
		require.Equal(t, metricNameConnectionTotal, r.URL.Query().Get("match[]"))
		require.NotEmpty(t, r.URL.Query().Get("start"))

		_, _ = w.Write([]byte(`{"metric":{"__name__":"traces_service_io_connection_total","k8s.namespace.name":"default"},"timestamps":[1781332967250]}`))
	}))
	defer server.Close()

	firstSeen, err := queryFirstSeenFromExport(context.Background(), server.URL, time.Now().Add(-exportLookback))
	require.NoError(t, err)
	require.Len(t, firstSeen, 1)
}
