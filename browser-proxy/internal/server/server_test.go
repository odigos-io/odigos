package server

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

func newTestServer(t *testing.T, upstream string) *Server {
	t.Helper()
	s, err := New(&config.Config{
		ListenAddr:       ":0",
		Upstream:         upstream,
		AgentDir:         "/var/odigos/browser",
		AgentFile:        "agent.js",
		OtlpHTTPEndpoint: "http://collector:4318",
		ServiceName:      "test-frontend",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s
}

func TestProxyInjectsHTML(t *testing.T) {
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html><head></head><body>app</body></html>")
	}))
	defer app.Close()

	s := newTestServer(t, app.URL)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, "window.__ODIGOS__=") {
		t.Fatalf("expected injected config script, got: %s", body)
	}
	if !strings.Contains(body, `src="`+config.AgentJsPath+`"`) {
		t.Fatalf("expected injected agent script, got: %s", body)
	}
	if rec.Header().Get("Content-Encoding") != "" {
		t.Fatalf("expected content-encoding to be stripped after injection of identity response")
	}
}

func TestProxyInjectsGzippedHTML(t *testing.T) {
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		_, _ = gw.Write([]byte("<html><head></head><body>app</body></html>"))
		_ = gw.Close()
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "gzip")
		_, _ = w.Write(buf.Bytes())
	}))
	defer app.Close()

	s := newTestServer(t, app.URL)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Header().Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected gzip content-encoding after recompression, got %q", rec.Header().Get("Content-Encoding"))
	}
	gr, err := gzip.NewReader(rec.Body)
	if err != nil {
		t.Fatalf("response is not valid gzip: %v", err)
	}
	defer gr.Close()
	decoded, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzipped body: %v", err)
	}
	body := string(decoded)
	if !strings.Contains(body, "window.__ODIGOS__=") {
		t.Fatalf("expected injected config in gzipped html, got: %s", body)
	}
	if !strings.Contains(body, "app") {
		t.Fatalf("expected original content preserved, got: %s", body)
	}
}

func TestHealthz(t *testing.T) {
	s := newTestServer(t, "http://unused.local")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, config.HealthPath, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("expected ok body, got %q", rec.Body.String())
	}
}

func TestProxyDoesNotInjectNonHTML(t *testing.T) {
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"ok":true}`)
	}))
	defer app.Close()

	s := newTestServer(t, app.URL)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api", nil))

	if strings.Contains(rec.Body.String(), "__ODIGOS__") {
		t.Fatalf("must not inject into non-HTML responses: %s", rec.Body.String())
	}
}

func TestOTLPForwardingAndCORS(t *testing.T) {
	var gotPath string
	var gotBody []byte
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	s, err := New(&config.Config{
		ListenAddr:       ":0",
		Upstream:         "http://unused.local",
		OtlpHTTPEndpoint: collector.URL,
		AgentDir:         "/var/odigos/browser",
		AgentFile:        "agent.js",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Preflight
	pre := httptest.NewRecorder()
	preReq := httptest.NewRequest(http.MethodOptions, config.TracesPath, nil)
	preReq.Header.Set("Origin", "https://frontend.example.com")
	s.Handler().ServeHTTP(pre, preReq)
	if pre.Code != http.StatusNoContent {
		t.Fatalf("preflight expected 204, got %d", pre.Code)
	}
	if pre.Header().Get("Access-Control-Allow-Origin") != "https://frontend.example.com" {
		t.Fatalf("missing CORS origin on preflight: %v", pre.Header())
	}

	// Actual POST
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, config.TracesPath, strings.NewReader("payload"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from forwarded OTLP, got %d", rec.Code)
	}
	if gotPath != "/v1/traces" {
		t.Fatalf("expected collector path /v1/traces, got %q", gotPath)
	}
	if string(gotBody) != "payload" {
		t.Fatalf("expected forwarded body 'payload', got %q", string(gotBody))
	}
	if rec.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatalf("expected CORS header on OTLP response")
	}
}
