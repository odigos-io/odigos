package traces

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/logicmonitor/lm-data-sdk-go/internal/client"
	"github.com/logicmonitor/lm-data-sdk-go/model"
	rateLimiter "github.com/logicmonitor/lm-data-sdk-go/pkg/ratelimiter"
	"github.com/logicmonitor/lm-data-sdk-go/utils"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/pdata/ptrace/ptraceotlp"
)

const (
	otlpTraceIngestURI       = "/api/v1/traces"
	defaultBatchingInterval  = 10 * time.Second
	maxHTTPResponseReadBytes = 64 * 1024
	headerRetryAfter         = "Retry-After"
)

type LMTraceIngest struct {
	client             *http.Client
	url                string
	auth               utils.AuthParams
	gzip               bool
	rateLimiterSetting rateLimiter.TraceRateLimiterSetting
	rateLimiter        rateLimiter.RateLimiter
	batch              *traceBatch
	collectorID        string
	userAgent          string
}

type lmTraceIngestRequest struct {
	tracesPayload model.TracesPayload
}

type LMTraceIngestResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type SendTraceResponse struct {
	StatusCode int    `json:"statusCode"`
	Success    bool   `json:"success"`
	Message    string `json:"message"`

	RetryAfter int `json:"retryAfter"`

	Error       error `json:"error"`
	MultiStatus []struct {
		Code  float64 `json:"code"`
		Error string  `json:"error"`
	} `json:"multiStatus"`
}

type traceBatch struct {
	enabled  bool
	data     *lmTraceIngestRequest
	interval time.Duration
	lock     *sync.Mutex
}

// NewLMTraceIngest initializes LMTraceIngest
func NewLMTraceIngest(ctx context.Context, opts ...Option) (*LMTraceIngest, error) {
	traceIngest := LMTraceIngest{
		client:             client.New(),
		auth:               utils.AuthParams{},
		gzip:               true,
		rateLimiterSetting: rateLimiter.TraceRateLimiterSetting{},
		batch:              NewTraceBatch(),
	}

	for _, opt := range opts {
		if err := opt(&traceIngest); err != nil {
			return nil, err
		}
	}

	var err error
	if traceIngest.url == "" {
		tracesURL, err := utils.URL()
		if err != nil {
			return nil, fmt.Errorf("NewLMTraceIngest: failed to create traces URL: %v", err)
		}
		traceIngest.url = tracesURL
	}

	traceIngest.rateLimiter, err = rateLimiter.NewTraceRateLimiter(traceIngest.rateLimiterSetting)
	if err != nil {
		return nil, err
	}
	go traceIngest.rateLimiter.Run(ctx)

	if traceIngest.batch.enabled {
		go traceIngest.processBatch(ctx)
	}
	return &traceIngest, nil
}

func NewTraceBatch() *traceBatch {
	return &traceBatch{
		enabled:  true,
		interval: defaultBatchingInterval,
		lock:     &sync.Mutex{},
		data: &lmTraceIngestRequest{
			tracesPayload: model.TracesPayload{
				TraceData: ptrace.NewTraces(),
			},
		},
	}
}

func (traceIngest *LMTraceIngest) processBatch(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.NewTicker(traceIngest.batch.batchInterval()).C:
			req := traceIngest.batch.combineBatchedTraceRequests()
			if req == nil {
				return
			}
			_, err := traceIngest.export(req)
			if err != nil {
				log.Println(err)
			}
		}
	}
}

// batchInterval returns the time interval for batching
func (batch *traceBatch) batchInterval() time.Duration {
	return batch.interval
}

// SendTraces is the entry point for receiving trace data
func (traceIngest *LMTraceIngest) SendTraces(ctx context.Context, td ptrace.Traces, o ...SendTracesOptionalParameters) (*SendTraceResponse, error) {
	req, err := traceIngest.buildTracesRequest(ctx, td, o...)
	if err != nil {
		return nil, err
	}

	if traceIngest.batch.enabled {
		traceIngest.batch.pushToBatch(req)
		return nil, nil
	}
	return traceIngest.export(req)
}

func (traceIngest *LMTraceIngest) buildTracesRequest(ctx context.Context, td ptrace.Traces, o ...SendTracesOptionalParameters) (*lmTraceIngestRequest, error) {
	tracesPayload := model.TracesPayload{
		TraceData: td,
	}
	return &lmTraceIngestRequest{tracesPayload: tracesPayload}, nil
}

// pushToBatch adds incoming trace requests to traceBatch internal cache
func (batch *traceBatch) pushToBatch(req *lmTraceIngestRequest) {
	batch.lock.Lock()
	defer batch.lock.Unlock()
	req.tracesPayload.TraceData.ResourceSpans().MoveAndAppendTo(ptrace.ResourceSpansSlice(batch.data.tracesPayload.TraceData.ResourceSpans()))
}

func (batch *traceBatch) combineBatchedTraceRequests() *lmTraceIngestRequest {
	batch.lock.Lock()
	defer batch.lock.Unlock()

	if batch.data.tracesPayload.TraceData.SpanCount() == 0 {
		return nil
	}

	req := &lmTraceIngestRequest{tracesPayload: batch.data.tracesPayload}

	// flushing out trace batch
	if batch.enabled {
		batch.data.tracesPayload.TraceData = ptrace.NewTraces()
	}
	return req
}

// export exports trace to the LM platform
func (traceIngest *LMTraceIngest) export(req *lmTraceIngestRequest) (*SendTraceResponse, error) {
	if req.tracesPayload.TraceData.SpanCount() == 0 {
		return nil, nil
	}
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-protobuf"

	if traceIngest.collectorID != "" {
		headers["Collector-ID"] = traceIngest.collectorID
	}

	traceData := req.tracesPayload.TraceData
	tr := ptraceotlp.NewExportRequestFromTraces(traceData)
	body, err := tr.MarshalProto()
	if err != nil {
		return nil, err
	}

	token, err := traceIngest.auth.GetCredentials(http.MethodPost, otlpTraceIngestURI, body)
	if err != nil {
		return nil, fmt.Errorf("LMTraceIngest.export: failed to get auth credentials: %w", err)
	}

	cfg := client.RequestConfig{
		Client:          traceIngest.client,
		RateLimiter:     traceIngest.rateLimiter,
		Url:             traceIngest.url,
		Body:            body,
		Uri:             otlpTraceIngestURI,
		Method:          http.MethodPost,
		Token:           token,
		Gzip:            traceIngest.gzip,
		Headers:         headers,
		UserAgent:       traceIngest.userAgent,
		PayloadMetadata: rateLimiter.TracePayloadMetadata{RequestSpanCount: uint64(req.tracesPayload.TraceData.SpanCount())},
	}

	resp, err := client.DoRequest(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("LMTraceIngest.export: traces export request failed: %w", err)
	}
	parsedResp, err := readResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("LMTraceIngest.export: failed to read response: %w", err)
	}

	sendTraceResp := &SendTraceResponse{
		StatusCode: parsedResp.StatusCode,
		Success:    parsedResp.Success,
		Message:    parsedResp.Message,

		Error:       parsedResp.Error,
		MultiStatus: parsedResp.MultiStatus,

		RetryAfter: parsedResp.RetryAfter,
	}

	if !sendTraceResp.Success {
		return sendTraceResp, fmt.Errorf("LMTraceIngest.export: failed to export traces: %w", sendTraceResp.Error)
	}
	return sendTraceResp, nil
}

// readResponse handles the http response returned by LM platform
func readResponse(resp *http.Response) (*model.TraceIngestAPIResponse, error) {
	defer func() {
		// Discard any remaining response body when we are done reading.
		io.CopyN(io.Discard, resp.Body, maxHTTPResponseReadBytes) // nolint:errcheck
		resp.Body.Close()
	}()

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		// Request is successful.
		return &model.TraceIngestAPIResponse{
			StatusCode: resp.StatusCode,
			Success:    true,
		}, nil
	}

	parsedResponse := decodeResponse(resp)

	// Format the error message. Use the status if it is present in the response.
	var formattedErr error
	if parsedResponse != nil {
		formattedErr = fmt.Errorf(
			"readResponse: error exporting items, request to %s responded with HTTP Status Code %d, Message=%s",
			resp.Request.URL, resp.StatusCode, parsedResponse.Message)
	} else {
		formattedErr = fmt.Errorf(
			"readResponse: error exporting items, request to %s responded with HTTP Status Code %d",
			resp.Request.URL, resp.StatusCode)
	}
	retryAfter := 0
	if val := resp.Header.Get(headerRetryAfter); val != "" {
		if seconds, err2 := strconv.Atoi(val); err2 == nil {
			retryAfter = seconds
		}
	}
	return &model.TraceIngestAPIResponse{
		StatusCode: resp.StatusCode,
		Success:    false,
		Error:      formattedErr,
		RetryAfter: retryAfter,
	}, nil
}

// Read the response and decode
// Returns nil if the response is empty or cannot be decoded.
func decodeResponse(resp *http.Response) *LMTraceIngestResponse {
	var traceIngestResponse *LMTraceIngestResponse
	if resp.StatusCode >= 400 && resp.StatusCode <= 599 {
		// Request failed. Read the body.
		maxRead := resp.ContentLength
		if maxRead == -1 || maxRead > maxHTTPResponseReadBytes {
			maxRead = maxHTTPResponseReadBytes
		}
		respBytes := make([]byte, maxRead)
		n, err := io.ReadFull(resp.Body, respBytes)
		if err == nil && n > 0 {
			traceIngestResponse = &LMTraceIngestResponse{}
			err = json.Unmarshal(respBytes, traceIngestResponse)
			if err != nil {
				traceIngestResponse = nil
			}
		}
	}
	return traceIngestResponse
}
