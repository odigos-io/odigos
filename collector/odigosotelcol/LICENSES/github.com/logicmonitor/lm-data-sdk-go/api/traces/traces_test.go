package traces

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/logicmonitor/lm-data-sdk-go/internal/testutil"
	"github.com/logicmonitor/lm-data-sdk-go/model"
	rateLimiter "github.com/logicmonitor/lm-data-sdk-go/pkg/ratelimiter"
	"github.com/logicmonitor/lm-data-sdk-go/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

var (
	TestSpanStartTime      = time.Date(2020, 2, 11, 20, 26, 12, 321, time.UTC)
	TestSpanStartTimestamp = pcommon.NewTimestampFromTime(TestSpanStartTime)

	TestSpanEventTime      = time.Date(2020, 2, 11, 20, 26, 13, 123, time.UTC)
	TestSpanEventTimestamp = pcommon.NewTimestampFromTime(TestSpanEventTime)

	TestSpanEndTime      = time.Date(2020, 2, 11, 20, 26, 13, 789, time.UTC)
	TestSpanEndTimestamp = pcommon.NewTimestampFromTime(TestSpanEndTime)
)

func TestNewLMTraceIngest(t *testing.T) {

	testutil.SetTestLMEnvVars()
	defer testutil.CleanupTestLMEnvVars()

	t.Run("should return Trace Ingest instance with default values", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lli, err := NewLMTraceIngest(ctx)
		assert.NoError(t, err)
		assert.Equal(t, true, lli.batch.enabled)
		assert.Equal(t, defaultBatchingInterval, lli.batch.interval)
		assert.Equal(t, true, lli.gzip)
		assert.NotNil(t, lli.client)
	})

	t.Run("should return Trace Ingest instance with options applied", func(t *testing.T) {

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		lli, err := NewLMTraceIngest(ctx, WithTraceBatchingInterval(5*time.Second))
		assert.NoError(t, err)
		assert.Equal(t, true, lli.batch.enabled)
		assert.Equal(t, 5*time.Second, lli.batch.interval)
	})
}

func TestSendTraces(t *testing.T) {

	testutil.SetTestLMEnvVars()
	defer testutil.CleanupTestLMEnvVars()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := LMTraceIngestResponse{
			Success: true,
			Message: "Accepted",
		}
		w.WriteHeader(http.StatusAccepted)
		assert.NoError(t, json.NewEncoder(w).Encode(&response))
	}))

	defer ts.Close()

	t.Run("send traces without batching", func(t *testing.T) {

		rateLimiter, _ := rateLimiter.NewLogRateLimiter(rateLimiter.LogRateLimiterSetting{RequestCount: 100})

		e := &LMTraceIngest{
			client:      ts.Client(),
			url:         ts.URL,
			auth:        utils.AuthParams{},
			rateLimiter: rateLimiter,
			batch:       &traceBatch{enabled: false},
		}

		resp, err := e.SendTraces(context.Background(), createTraceData())
		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("send traces with batching enabled", func(t *testing.T) {

		rateLimiter, _ := rateLimiter.NewLogRateLimiter(rateLimiter.LogRateLimiterSetting{RequestCount: 100})
		e := &LMTraceIngest{
			client:      ts.Client(),
			url:         ts.URL,
			auth:        utils.AuthParams{},
			rateLimiter: rateLimiter,
			batch:       &traceBatch{enabled: true, interval: 1 * time.Second, lock: &sync.Mutex{}, data: &lmTraceIngestRequest{tracesPayload: model.TracesPayload{TraceData: ptrace.NewTraces()}}},
		}
		_, err := e.SendTraces(context.Background(), createTraceData())
		assert.NoError(t, err)
	})
}

func TestPushToBatch(t *testing.T) {
	t.Run("should add traces to batch", func(t *testing.T) {

		traceIngest := LMTraceIngest{batch: NewTraceBatch()}

		testData := createTraceData()

		req, err := traceIngest.buildTracesRequest(context.Background(), createTraceData())
		assert.NoError(t, err)

		before := traceIngest.batch.data.tracesPayload.TraceData.SpanCount()

		traceIngest.batch.pushToBatch(req)

		expectedSpanCount := before + testData.SpanCount()

		assert.Equal(t, expectedSpanCount, traceIngest.batch.data.tracesPayload.TraceData.SpanCount())
	})
}

func TestReadResponse(t *testing.T) {
	t.Run("success response", func(t *testing.T) {
		ingestResponse, err := readResponse(&http.Response{
			StatusCode: http.StatusAccepted,
			Body:       ioutil.NopCloser(bytes.NewBufferString("Accepted")),
		})
		require.NoError(t, err)
		assert.Equal(t, model.TraceIngestAPIResponse{
			Success:    true,
			StatusCode: http.StatusAccepted,
		}, *ingestResponse)
	})

	t.Run("non multi-status error response", func(t *testing.T) {
		data := []byte(`{
			"success": false,
			"message": "Too Many Requests"
		  }`)
		ingestResponse, err := readResponse(&http.Response{
			StatusCode:    http.StatusTooManyRequests,
			ContentLength: int64(len(data)),
			Request:       httptest.NewRequest(http.MethodPost, "https://example.logicmonitor.com"+otlpTraceIngestURI, nil),
			Body:          ioutil.NopCloser(bytes.NewReader(data)),
		})
		require.NoError(t, err)
		assert.Equal(t, model.TraceIngestAPIResponse{
			Success:    false,
			StatusCode: http.StatusTooManyRequests,
			Error:      fmt.Errorf("readResponse: error exporting items, request to https://example.logicmonitor.com%s responded with HTTP Status Code 429, Message=Too Many Requests", otlpTraceIngestURI),
		}, *ingestResponse)
	})
}

func createTraceData() ptrace.Traces {
	td := GenerateTracesOneEmptyInstrumentationLibrary()
	scopespan := td.ResourceSpans().At(0).ScopeSpans().At(0)
	fillSpanOne(scopespan.Spans().AppendEmpty())
	return td
}

func GenerateTracesOneEmptyInstrumentationLibrary() ptrace.Traces {
	td := GenerateTracesNoLibraries()
	td.ResourceSpans().At(0).ScopeSpans().AppendEmpty()
	return td
}

func GenerateTracesNoLibraries() ptrace.Traces {
	td := GenerateTracesOneEmptyResourceSpans()
	return td
}

func GenerateTracesOneEmptyResourceSpans() ptrace.Traces {
	td := ptrace.NewTraces()
	td.ResourceSpans().AppendEmpty()
	return td
}

func fillSpanOne(span ptrace.Span) {
	span.SetName("operationA")
	span.SetStartTimestamp(TestSpanStartTimestamp)
	span.SetEndTimestamp(TestSpanEndTimestamp)
	span.SetDroppedAttributesCount(1)
	evs := span.Events()
	ev0 := evs.AppendEmpty()
	ev0.SetTimestamp(TestSpanEventTimestamp)
	ev0.SetName("event-with-attr")
	//initSpanEventAttributes(ev0.Attributes())
	ev0.SetDroppedAttributesCount(2)
	ev1 := evs.AppendEmpty()
	ev1.SetTimestamp(TestSpanEventTimestamp)
	ev1.SetName("event")
	ev1.SetDroppedAttributesCount(2)
	span.SetDroppedEventsCount(1)
	status := span.Status()
	status.SetCode(ptrace.StatusCodeError)
	status.SetMessage("status-cancelled")
}
