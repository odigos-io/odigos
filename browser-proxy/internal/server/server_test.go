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
		ExportToken:      "test-export-token",
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
	if strings.Contains(body, "window.__ODIGOS__=") {
		t.Fatalf("must not inject inline config script, got: %s", body)
	}
	if !strings.Contains(body, `src="`+config.ConfigJsPath+`"`) {
		t.Fatalf("expected injected config.js script, got: %s", body)
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
	if !strings.Contains(body, `src="`+config.ConfigJsPath+`"`) {
		t.Fatalf("expected injected config.js in gzipped html, got: %s", body)
	}
	if !strings.Contains(body, "app") {
		t.Fatalf("expected original content preserved, got: %s", body)
	}
}

func TestProxyPropagatesCSPNonce(t *testing.T) {
	app := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'nonce-abc123'")
		_, _ = io.WriteString(w, "<html><head></head><body>app</body></html>")
	}))
	defer app.Close()

	s := newTestServer(t, app.URL)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	body := rec.Body.String()
	if !strings.Contains(body, `nonce="abc123"`) {
		t.Fatalf("expected CSP nonce on injected scripts, got: %s", body)
	}
}

func TestConfigJS(t *testing.T) {
	s := newTestServer(t, "http://unused.local")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, config.ConfigJsPath, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.HasPrefix(body, "window.__ODIGOS__=") {
		t.Fatalf("expected config.js assignment, got: %s", body)
	}
	if !strings.Contains(body, `"exportToken":"test-export-token"`) {
		t.Fatalf("expected export token in config.js, got: %s", body)
	}
	if !strings.Contains(body, `"logsPath":"`+config.LogsPath+`"`) {
		t.Fatalf("expected logsPath in config.js, got: %s", body)
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("expected nosniff header")
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

	if strings.Contains(rec.Body.String(), "__ODIGOS__") || strings.Contains(rec.Body.String(), config.ConfigJsPath) {
		t.Fatalf("must not inject into non-HTML responses: %s", rec.Body.String())
	}
}

func TestOTLPRequiresToken(t *testing.T) {
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer collector.Close()

	s, err := New(&config.Config{
		ListenAddr:       ":0",
		Upstream:         "http://unused.local",
		OtlpHTTPEndpoint: collector.URL,
		AgentDir:         "/var/odigos/browser",
		AgentFile:        "agent.js",
		ExportToken:      "secret-token",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, config.TracesPath, strings.NewReader("payload"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Host = "frontend.example.com"
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without token, got %d", rec.Code)
	}
}

func TestOTLPForwardingAndCORS(t *testing.T) {
	var gotPath string
	var gotBody []byte
	var gotAuth string
	collector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
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
		ExportToken:      "secret-token",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Preflight from same site
	pre := httptest.NewRecorder()
	preReq := httptest.NewRequest(http.MethodOptions, config.TracesPath, nil)
	preReq.Header.Set("Origin", "https://frontend.example.com")
	preReq.Host = "frontend.example.com"
	s.Handler().ServeHTTP(pre, preReq)
	if pre.Code != http.StatusNoContent {
		t.Fatalf("preflight expected 204, got %d", pre.Code)
	}
	if pre.Header().Get("Access-Control-Allow-Origin") != "https://frontend.example.com" {
		t.Fatalf("missing CORS origin on preflight: %v", pre.Header())
	}
	if !strings.Contains(pre.Header().Get("Access-Control-Allow-Headers"), "authorization") {
		t.Fatalf("CORS must allow authorization header: %v", pre.Header())
	}

	// Cross-site preflight rejected
	bad := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodOptions, config.TracesPath, nil)
	badReq.Header.Set("Origin", "https://evil.example.com")
	badReq.Host = "frontend.example.com"
	s.Handler().ServeHTTP(bad, badReq)
	if bad.Code != http.StatusForbidden {
		t.Fatalf("cross-site preflight expected 403, got %d", bad.Code)
	}

	// Authenticated POST
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, config.TracesPath, strings.NewReader("payload"))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Origin", "https://frontend.example.com")
	req.Host = "frontend.example.com"
	s.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from forwarded OTLP, got %d body=%s", rec.Code, rec.Body.String())
	}
	if gotPath != "/v1/traces" {
		t.Fatalf("expected collector path /v1/traces, got %q", gotPath)
	}
	if string(gotBody) != "payload" {
		t.Fatalf("expected forwarded body 'payload', got %q", string(gotBody))
	}
	if gotAuth != "" {
		t.Fatalf("export token must not be forwarded to collector, got %q", gotAuth)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://frontend.example.com" {
		t.Fatalf("expected CORS header on OTLP response")
	}
}
