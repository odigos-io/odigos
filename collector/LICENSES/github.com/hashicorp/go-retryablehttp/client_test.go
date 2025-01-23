// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package retryablehttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
)

func TestRequest(t *testing.T) {
	// Fails on invalid request
	_, err := NewRequest("GET", "://foo", nil)
	if err == nil {
		t.Fatalf("should error")
	}

	// Works with no request body
	_, err = NewRequest("GET", "http://foo", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Works with request body
	body := bytes.NewReader([]byte("yo"))
	req, err := NewRequest("GET", "/", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Request allows typical HTTP request forming methods
	req.Header.Set("X-Test", "foo")
	if v, ok := req.Header["X-Test"]; !ok || len(v) != 1 || v[0] != "foo" {
		t.Fatalf("bad headers: %v", req.Header)
	}

	// Sets the Content-Length automatically for LenReaders
	if req.ContentLength != 2 {
		t.Fatalf("bad ContentLength: %d", req.ContentLength)
	}
}

func TestFromRequest(t *testing.T) {
	// Works with no request body
	httpReq, err := http.NewRequest("GET", "http://foo", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	_, err = FromRequest(httpReq)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Works with request body
	body := bytes.NewReader([]byte("yo"))
	httpReq, err = http.NewRequest("GET", "/", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req, err := FromRequest(httpReq)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Preserves headers
	httpReq.Header.Set("X-Test", "foo")
	if v, ok := req.Header["X-Test"]; !ok || len(v) != 1 || v[0] != "foo" {
		t.Fatalf("bad headers: %v", req.Header)
	}

	// Preserves the Content-Length automatically for LenReaders
	if req.ContentLength != 2 {
		t.Fatalf("bad ContentLength: %d", req.ContentLength)
	}
}

// Since normal ways we would generate a Reader have special cases, use a
// custom type here
type custReader struct {
	val string
	pos int
}

func (c *custReader) Read(p []byte) (n int, err error) {
	if c.val == "" {
		c.val = "hello"
	}
	if c.pos >= len(c.val) {
		return 0, io.EOF
	}
	var i int
	for i = 0; i < len(p) && i+c.pos < len(c.val); i++ {
		p[i] = c.val[i+c.pos]
	}
	c.pos += i
	return i, nil
}

func TestClient_Do(t *testing.T) {
	testBytes := []byte("hello")
	// Native func
	testClientDo(t, ReaderFunc(func() (io.Reader, error) {
		return bytes.NewReader(testBytes), nil
	}))
	// Native func, different Go type
	testClientDo(t, func() (io.Reader, error) {
		return bytes.NewReader(testBytes), nil
	})
	// []byte
	testClientDo(t, testBytes)
	// *bytes.Buffer
	testClientDo(t, bytes.NewBuffer(testBytes))
	// *bytes.Reader
	testClientDo(t, bytes.NewReader(testBytes))
	// io.ReadSeeker
	testClientDo(t, strings.NewReader(string(testBytes)))
	// io.Reader
	testClientDo(t, &custReader{})
}

func testClientDo(t *testing.T, body interface{}) {
	// Create a request
	req, err := NewRequest("PUT", "http://127.0.0.1:28934/v1/foo", body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req.Header.Set("foo", "bar")

	// Track the number of times the logging hook was called
	retryCount := -1

	// Create the client. Use short retry windows.
	client := NewClient()
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 50 * time.Millisecond
	client.RetryMax = 50
	client.RequestLogHook = func(logger Logger, req *http.Request, retryNumber int) {
		retryCount = retryNumber

		if logger != client.Logger {
			t.Fatalf("Client logger was not passed to logging hook")
		}

		dumpBytes, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			t.Fatal("Dumping requests failed")
		}

		dumpString := string(dumpBytes)
		if !strings.Contains(dumpString, "PUT /v1/foo") {
			t.Fatalf("Bad request dump:\n%s", dumpString)
		}
	}

	// Send the request
	var resp *http.Response
	doneCh := make(chan struct{})
	errCh := make(chan error, 1)
	go func() {
		defer close(doneCh)
		defer close(errCh)
		var err error
		resp, err = client.Do(req)
		errCh <- err
	}()

	select {
	case <-doneCh:
		t.Fatalf("should retry on error")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to connection failure.
	}

	// Create the mock handler. First we return a 500-range response to ensure
	// that we power through and keep retrying in the face of recoverable
	// errors.
	code := int64(500)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check the request details
		if r.Method != "PUT" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/v1/foo" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}

		// Check the headers
		if v := r.Header.Get("foo"); v != "bar" {
			t.Fatalf("bad header: expect foo=bar, got foo=%v", v)
		}

		// Check the payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte("hello")
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(int(atomic.LoadInt64(&code)))
	})

	// Create a test server
	list, err := net.Listen("tcp", ":28934")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	defer list.Close()
	go http.Serve(list, handler)

	// Wait again
	select {
	case <-doneCh:
		t.Fatalf("should retry on 500-range")
	case <-time.After(200 * time.Millisecond):
		// Client should still be retrying due to 500's.
	}

	// Start returning 200's
	atomic.StoreInt64(&code, 200)

	// Wait again
	select {
	case <-doneCh:
	case <-time.After(time.Second):
		t.Fatalf("timed out")
	}

	if resp.StatusCode != 200 {
		t.Fatalf("exected 200, got: %d", resp.StatusCode)
	}

	if retryCount < 0 {
		t.Fatal("request log hook was not called")
	}

	err = <-errCh
	if err != nil {
		t.Fatalf("err: %v", err)
	}
}

func TestClient_Do_WithResponseHandler(t *testing.T) {
	// Create the client. Use short retry windows so we fail faster.
	client := NewClient()
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 10 * time.Millisecond
	client.RetryMax = 2

	var checks int
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		checks++
		if err != nil && strings.Contains(err.Error(), "nonretryable") {
			return false, nil
		}
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	var shouldSucceed bool
	tests := []struct {
		name           string
		handler        ResponseHandlerFunc
		expectedChecks int // often 2x number of attempts since we check twice
		err            string
	}{
		{
			name:           "nil handler",
			handler:        nil,
			expectedChecks: 1,
		},
		{
			name: "handler always succeeds",
			handler: func(*http.Response) error {
				return nil
			},
			expectedChecks: 2,
		},
		{
			name: "handler always fails in a retryable way",
			handler: func(*http.Response) error {
				return errors.New("retryable failure")
			},
			expectedChecks: 6,
		},
		{
			name: "handler always fails in a nonretryable way",
			handler: func(*http.Response) error {
				return errors.New("nonretryable failure")
			},
			expectedChecks: 2,
		},
		{
			name: "handler succeeds on second attempt",
			handler: func(*http.Response) error {
				if shouldSucceed {
					return nil
				}
				shouldSucceed = true
				return errors.New("retryable failure")
			},
			expectedChecks: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks = 0
			shouldSucceed = false
			// Create the request
			req, err := NewRequest("GET", ts.URL, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			req.SetResponseHandler(tt.handler)

			// Send the request.
			_, err = client.Do(req)
			if err != nil && !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("error does not match expectation, expected: %s, got: %s", tt.err, err.Error())
			}
			if err == nil && tt.err != "" {
				t.Fatalf("no error, expected: %s", tt.err)
			}

			if checks != tt.expectedChecks {
				t.Fatalf("expected %d attempts, got %d attempts", tt.expectedChecks, checks)
			}
		})
	}
}

func TestClient_Do_WithPrepareRetry(t *testing.T) {
	// Create the client. Use short retry windows so we fail faster.
	client := NewClient()
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 10 * time.Millisecond
	client.RetryMax = 2

	var checks int
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		checks++
		if err != nil && strings.Contains(err.Error(), "nonretryable") {
			return false, nil
		}
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	var prepareChecks int
	client.PrepareRetry = func(req *http.Request) error {
		prepareChecks++
		req.Header.Set("foo", strconv.Itoa(prepareChecks))
		return nil
	}

	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	var shouldSucceed bool
	tests := []struct {
		name                  string
		handler               ResponseHandlerFunc
		expectedChecks        int // often 2x number of attempts since we check twice
		expectedPrepareChecks int
		err                   string
	}{
		{
			name:                  "nil handler",
			handler:               nil,
			expectedChecks:        1,
			expectedPrepareChecks: 0,
		},
		{
			name: "handler always succeeds",
			handler: func(*http.Response) error {
				return nil
			},
			expectedChecks:        2,
			expectedPrepareChecks: 0,
		},
		{
			name: "handler always fails in a retryable way",
			handler: func(*http.Response) error {
				return errors.New("retryable failure")
			},
			expectedChecks:        6,
			expectedPrepareChecks: 2,
		},
		{
			name: "handler always fails in a nonretryable way",
			handler: func(*http.Response) error {
				return errors.New("nonretryable failure")
			},
			expectedChecks:        2,
			expectedPrepareChecks: 0,
		},
		{
			name: "handler succeeds on second attempt",
			handler: func(*http.Response) error {
				if shouldSucceed {
					return nil
				}
				shouldSucceed = true
				return errors.New("retryable failure")
			},
			expectedChecks:        4,
			expectedPrepareChecks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checks = 0
			prepareChecks = 0
			shouldSucceed = false
			// Create the request
			req, err := NewRequest("GET", ts.URL, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			req.SetResponseHandler(tt.handler)

			// Send the request.
			_, err = client.Do(req)
			if err != nil && !strings.Contains(err.Error(), tt.err) {
				t.Fatalf("error does not match expectation, expected: %s, got: %s", tt.err, err.Error())
			}
			if err == nil && tt.err != "" {
				t.Fatalf("no error, expected: %s", tt.err)
			}

			if checks != tt.expectedChecks {
				t.Fatalf("expected %d attempts, got %d attempts", tt.expectedChecks, checks)
			}

			if prepareChecks != tt.expectedPrepareChecks {
				t.Fatalf("expected %d attempts of prepare check, got %d attempts", tt.expectedPrepareChecks, prepareChecks)
			}
			header := req.Request.Header.Get("foo")
			if tt.expectedPrepareChecks == 0 && header != "" {
				t.Fatalf("expected no changes to request header 'foo', but got '%s'", header)
			}
			expectedHeader := strconv.Itoa(tt.expectedPrepareChecks)
			if tt.expectedPrepareChecks != 0 && header != expectedHeader {
				t.Fatalf("expected changes in request header 'foo' '%s', but got '%s'", expectedHeader, header)
			}

		})
	}
}

func TestClient_Do_fails(t *testing.T) {
	// Mock server which always responds 500.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer ts.Close()

	serverUrlWithBasicAuth, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatalf("failed parsing test server url: %s", ts.URL)
	}
	serverUrlWithBasicAuth.User = url.UserPassword("user", "pasten")

	tests := []struct {
		url  string
		name string
		cr   CheckRetry
		err  string
	}{
		{
			url:  ts.URL,
			name: "default_retry_policy",
			cr:   DefaultRetryPolicy,
			err:  "giving up after 3 attempt(s)",
		},
		{
			url:  serverUrlWithBasicAuth.String(),
			name: "default_retry_policy_url_with_basic_auth",
			cr:   DefaultRetryPolicy,
			err:  redactURL(serverUrlWithBasicAuth) + " giving up after 3 attempt(s)",
		},
		{
			url:  ts.URL,
			name: "error_propagated_retry_policy",
			cr:   ErrorPropagatedRetryPolicy,
			err:  "giving up after 3 attempt(s): unexpected HTTP status 500 Internal Server Error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the client. Use short retry windows so we fail faster.
			client := NewClient()
			client.RetryWaitMin = 10 * time.Millisecond
			client.RetryWaitMax = 10 * time.Millisecond
			client.CheckRetry = tt.cr
			client.RetryMax = 2

			// Create the request
			req, err := NewRequest("POST", tt.url, nil)
			if err != nil {
				t.Fatalf("err: %v", err)
			}

			// Send the request.
			_, err = client.Do(req)
			if err == nil || !strings.HasSuffix(err.Error(), tt.err) {
				t.Fatalf("expected %#v, got: %#v", tt.err, err)
			}
		})
	}
}

func TestClient_Get(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Get(ts.URL + "/foo/bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_RequestLogHook(t *testing.T) {
	t.Run("RequestLogHook successfully called with default Logger", func(t *testing.T) {
		testClientRequestLogHook(t, defaultLogger)
	})
	t.Run("RequestLogHook successfully called with nil Logger", func(t *testing.T) {
		testClientRequestLogHook(t, nil)
	})
	t.Run("RequestLogHook successfully called with nil typed Logger", func(t *testing.T) {
		testClientRequestLogHook(t, Logger(nil))
	})
	t.Run("RequestLogHook successfully called with nil typed LeveledLogger", func(t *testing.T) {
		testClientRequestLogHook(t, LeveledLogger(nil))
	})
}

func testClientRequestLogHook(t *testing.T, logger interface{}) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	retries := -1
	testURIPath := "/foo/bar"

	client := NewClient()
	client.Logger = logger
	client.RequestLogHook = func(logger Logger, req *http.Request, retry int) {
		retries = retry

		if logger != client.Logger {
			t.Fatalf("Client logger was not passed to logging hook")
		}

		dumpBytes, err := httputil.DumpRequestOut(req, false)
		if err != nil {
			t.Fatal("Dumping requests failed")
		}

		dumpString := string(dumpBytes)
		if !strings.Contains(dumpString, "GET "+testURIPath) {
			t.Fatalf("Bad request dump:\n%s", dumpString)
		}
	}

	// Make the request.
	resp, err := client.Get(ts.URL + testURIPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()

	if retries < 0 {
		t.Fatal("Logging hook was not called")
	}
}

func TestClient_ResponseLogHook(t *testing.T) {
	t.Run("ResponseLogHook successfully called with hclog Logger", func(t *testing.T) {
		buf := new(bytes.Buffer)
		l := hclog.New(&hclog.LoggerOptions{
			Output: buf,
		})
		testClientResponseLogHook(t, l, buf)
	})
	t.Run("ResponseLogHook successfully called with nil Logger", func(t *testing.T) {
		buf := new(bytes.Buffer)
		testClientResponseLogHook(t, nil, buf)
	})
	t.Run("ResponseLogHook successfully called with nil typed Logger", func(t *testing.T) {
		buf := new(bytes.Buffer)
		testClientResponseLogHook(t, Logger(nil), buf)
	})
	t.Run("ResponseLogHook successfully called with nil typed LeveledLogger", func(t *testing.T) {
		buf := new(bytes.Buffer)
		testClientResponseLogHook(t, LeveledLogger(nil), buf)
	})
}

func testClientResponseLogHook(t *testing.T, l interface{}, buf *bytes.Buffer) {
	passAfter := time.Now().Add(100 * time.Millisecond)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if time.Now().After(passAfter) {
			w.WriteHeader(200)
			w.Write([]byte("test_200_body"))
		} else {
			w.WriteHeader(500)
			w.Write([]byte("test_500_body"))
		}
	}))
	defer ts.Close()

	client := NewClient()

	client.Logger = l
	client.RetryWaitMin = 10 * time.Millisecond
	client.RetryWaitMax = 10 * time.Millisecond
	client.RetryMax = 15
	client.ResponseLogHook = func(logger Logger, resp *http.Response) {
		if resp.StatusCode == 200 {
			successLog := "test_log_pass"
			// Log something when we get a 200
			if logger != nil {
				logger.Printf(successLog)
			} else {
				buf.WriteString(successLog)
			}
		} else {
			// Log the response body when we get a 500
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("err: %v", err)
			}
			failLog := string(body)
			if logger != nil {
				logger.Printf(failLog)
			} else {
				buf.WriteString(failLog)
			}
		}
	}

	// Perform the request. Exits when we finally get a 200.
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Make sure we can read the response body still, since we did not
	// read or close it from the response log hook.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if string(body) != "test_200_body" {
		t.Fatalf("expect %q, got %q", "test_200_body", string(body))
	}

	// Make sure we wrote to the logger on callbacks.
	out := buf.String()
	if !strings.Contains(out, "test_log_pass") {
		t.Fatalf("expect response callback on 200: %q", out)
	}
	if !strings.Contains(out, "test_500_body") {
		t.Fatalf("expect response callback on 500: %q", out)
	}
}

func TestClient_NewRequestWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, err := NewRequestWithContext(ctx, http.MethodGet, "/abc", nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if r.Context() != ctx {
		t.Fatal("Context must be set")
	}
}

func TestClient_RequestWithContext(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("test_200_body"))
	}))
	defer ts.Close()

	req, err := NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	ctx, cancel := context.WithCancel(req.Request.Context())
	reqCtx := req.WithContext(ctx)
	if reqCtx == req {
		t.Fatal("WithContext must return a new Request object")
	}

	client := NewClient()

	called := 0
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		called++
		return DefaultRetryPolicy(reqCtx.Request.Context(), resp, err)
	}

	cancel()
	_, err = client.Do(reqCtx)

	if called != 1 {
		t.Fatalf("CheckRetry called %d times, expected 1", called)
	}

	e := fmt.Sprintf("GET %s giving up after 1 attempt(s): %s", ts.URL, context.Canceled.Error())

	if err.Error() != e {
		t.Fatalf("Expected err to contain %s, got: %v", e, err)
	}
}

func TestClient_CheckRetry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "test_500_body", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient()

	retryErr := errors.New("retryError")
	called := 0
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		if called < 1 {
			called++
			return DefaultRetryPolicy(context.TODO(), resp, err)
		}

		return false, retryErr
	}

	// CheckRetry should return our retryErr value and stop the retry loop.
	_, err := client.Get(ts.URL)

	if called != 1 {
		t.Fatalf("CheckRetry called %d times, expected 1", called)
	}

	if err.Error() != fmt.Sprintf("GET %s giving up after 2 attempt(s): retryError", ts.URL) {
		t.Fatalf("Expected retryError, got:%v", err)
	}
}

func testStaticTime(t *testing.T) {
	timeNow = func() time.Time {
		now, err := time.Parse(time.RFC1123, "Fri, 31 Dec 1999 23:59:57 GMT")
		if err != nil {
			panic(err)
		}
		return now
	}
	t.Cleanup(func() {
		timeNow = time.Now
	})
}

func TestParseRetryAfterHeader(t *testing.T) {
	testStaticTime(t)
	tests := []struct {
		name    string
		headers []string
		sleep   time.Duration
		ok      bool
	}{
		{"seconds", []string{"2"}, time.Second * 2, true},
		{"date", []string{"Fri, 31 Dec 1999 23:59:59 GMT"}, time.Second * 2, true},
		{"past-date", []string{"Fri, 31 Dec 1999 23:59:00 GMT"}, 0, true},
		{"nil", nil, 0, false},
		{"two-headers", []string{"2", "3"}, time.Second * 2, true},
		{"empty", []string{""}, 0, false},
		{"negative", []string{"-2"}, 0, false},
		{"bad-date", []string{"Fri, 32 Dec 1999 23:59:59 GMT"}, 0, false},
		{"bad-date-format", []string{"badbadbad"}, 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sleep, ok := parseRetryAfterHeader(test.headers)
			if ok != test.ok {
				t.Fatalf("expected ok=%t, got ok=%t", test.ok, ok)
			}
			if sleep != test.sleep {
				t.Fatalf("expected sleep=%v, got sleep=%v", test.sleep, sleep)
			}
		})
	}
}

func TestClient_DefaultBackoff(t *testing.T) {
	testStaticTime(t)
	tests := []struct {
		name        string
		code        int
		retryHeader string
	}{
		{"http_429_seconds", http.StatusTooManyRequests, "2"},
		{"http_429_date", http.StatusTooManyRequests, "Fri, 31 Dec 1999 23:59:59 GMT"},
		{"http_503_seconds", http.StatusServiceUnavailable, "2"},
		{"http_503_date", http.StatusServiceUnavailable, "Fri, 31 Dec 1999 23:59:59 GMT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Retry-After", test.retryHeader)
				http.Error(w, fmt.Sprintf("test_%d_body", test.code), test.code)
			}))
			defer ts.Close()

			client := NewClient()

			var retryAfter time.Duration
			retryable := false

			client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
				retryable, _ = DefaultRetryPolicy(context.Background(), resp, err)
				retryAfter = DefaultBackoff(client.RetryWaitMin, client.RetryWaitMax, 1, resp)
				return false, nil
			}

			_, err := client.Get(ts.URL)
			if err != nil {
				t.Fatalf("expected no errors since retryable")
			}

			if !retryable {
				t.Fatal("Since the error is recoverable, the default policy shall return true")
			}

			if retryAfter != 2*time.Second {
				t.Fatalf("The header Retry-After specified 2 seconds, and shall not be %d seconds", retryAfter/time.Second)
			}
		})
	}
}

func TestClient_DefaultRetryPolicy_TLS(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	attempts := 0
	client := NewClient()
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		attempts++
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatalf("expected x509 error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestClient_DefaultRetryPolicy_redirects(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/", http.StatusFound)
	}))
	defer ts.Close()

	attempts := 0
	client := NewClient()
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		attempts++
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	_, err := client.Get(ts.URL)
	if err == nil {
		t.Fatalf("expected redirect error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestClient_DefaultRetryPolicy_invalidscheme(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	attempts := 0
	client := NewClient()
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		attempts++
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	url := strings.Replace(ts.URL, "http", "ftp", 1)
	_, err := client.Get(url)
	if err == nil {
		t.Fatalf("expected scheme error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestClient_DefaultRetryPolicy_invalidheadername(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	attempts := 0
	client := NewClient()
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		attempts++
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req.Header.Set("Header-Name-\033", "header value")
	_, err = client.StandardClient().Do(req)
	if err == nil {
		t.Fatalf("expected header error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestClient_DefaultRetryPolicy_invalidheadervalue(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	attempts := 0
	client := NewClient()
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		attempts++
		return DefaultRetryPolicy(context.TODO(), resp, err)
	}

	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	req.Header.Set("Header-Name", "bad header value \033")
	_, err = client.StandardClient().Do(req)
	if err == nil {
		t.Fatalf("expected header value error, got nil")
	}
	if attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", attempts)
	}
}

func TestClient_CheckRetryStop(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "test_500_body", http.StatusInternalServerError)
	}))
	defer ts.Close()

	client := NewClient()

	// Verify that this stops retries on the first try, with no errors from the client.
	called := 0
	client.CheckRetry = func(_ context.Context, resp *http.Response, err error) (bool, error) {
		called++
		return false, nil
	}

	_, err := client.Get(ts.URL)

	if called != 1 {
		t.Fatalf("CheckRetry called %d times, expected 1", called)
	}

	if err != nil {
		t.Fatalf("Expected no error, got:%v", err)
	}
}

func TestClient_Head(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "HEAD" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Head(ts.URL + "/foo/bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_Post(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("bad content-type: %s", ct)
		}

		// Check the payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte(`{"hello":"world"}`)
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := NewClient().Post(
		ts.URL+"/foo/bar",
		"application/json",
		strings.NewReader(`{"hello":"world"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestClient_PostForm(t *testing.T) {
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("bad method: %s", r.Method)
		}
		if r.RequestURI != "/foo/bar" {
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("bad content-type: %s", ct)
		}

		// Check the payload
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		expected := []byte(`hello=world`)
		if !bytes.Equal(body, expected) {
			t.Fatalf("bad: %v", body)
		}

		w.WriteHeader(200)
	}))
	defer ts.Close()

	// Create the form data.
	form, err := url.ParseQuery("hello=world")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	// Make the request.
	resp, err := NewClient().PostForm(ts.URL+"/foo/bar", form)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
}

func TestBackoff(t *testing.T) {
	type tcase struct {
		min    time.Duration
		max    time.Duration
		i      int
		expect time.Duration
	}
	cases := []tcase{
		{
			time.Second,
			5 * time.Minute,
			0,
			time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			1,
			2 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			2,
			4 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			3,
			8 * time.Second,
		},
		{
			time.Second,
			5 * time.Minute,
			63,
			5 * time.Minute,
		},
		{
			time.Second,
			5 * time.Minute,
			128,
			5 * time.Minute,
		},
	}

	for _, tc := range cases {
		if v := DefaultBackoff(tc.min, tc.max, tc.i, nil); v != tc.expect {
			t.Fatalf("bad: %#v -> %s", tc, v)
		}
	}
}

func TestClient_BackoffCustom(t *testing.T) {
	var retries int32

	client := NewClient()
	client.Backoff = func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
		atomic.AddInt32(&retries, 1)
		return time.Millisecond * 1
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&retries) == int32(client.RetryMax) {
			w.WriteHeader(200)
			return
		}
		w.WriteHeader(500)
	}))
	defer ts.Close()

	// Make the request.
	resp, err := client.Get(ts.URL + "/foo/bar")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()
	if retries != int32(client.RetryMax) {
		t.Fatalf("expected retries: %d != %d", client.RetryMax, retries)
	}
}

func TestClient_StandardClient(t *testing.T) {
	// Create a retryable HTTP client.
	client := NewClient()

	// Get a standard client.
	standard := client.StandardClient()

	// Ensure the underlying retrying client is set properly.
	if v := standard.Transport.(*RoundTripper).Client; v != client {
		t.Fatalf("expected %v, got %v", client, v)
	}
}

func TestClient_RedirectWithBody(t *testing.T) {
	var redirects int32
	// Mock server which always responds 200.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.RequestURI {
		case "/redirect":
			w.Header().Set("Location", "/target")
			w.WriteHeader(http.StatusTemporaryRedirect)
		case "/target":
			atomic.AddInt32(&redirects, 1)
			w.WriteHeader(http.StatusCreated)
		default:
			t.Fatalf("bad uri: %s", r.RequestURI)
		}
	}))
	defer ts.Close()

	client := NewClient()
	client.RequestLogHook = func(logger Logger, req *http.Request, retryNumber int) {
		if _, err := req.GetBody(); err != nil {
			t.Fatalf("unexpected error with GetBody: %v", err)
		}
	}
	// create a request with a body
	req, err := NewRequest(http.MethodPost, ts.URL+"/redirect", strings.NewReader(`{"foo":"bar"}`))
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201, got: %d", resp.StatusCode)
	}

	// now one without a body
	if err := req.SetBody(nil); err != nil {
		t.Fatalf("err: %v", err)
	}

	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code 201, got: %d", resp.StatusCode)
	}

	if atomic.LoadInt32(&redirects) != 2 {
		t.Fatalf("Expected the client to be redirected 2 times, got: %d", atomic.LoadInt32(&redirects))
	}
}
