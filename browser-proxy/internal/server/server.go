// Package server implements the odigos-browser-proxy HTTP server: a reverse proxy in front of a
// web-server container that injects CSP-safe OpenTelemetry browser SDK <script> tags into HTML
// responses and proxies authenticated browser OTLP/HTTP telemetry to the node-local collector.
package server

import (
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

const (
	// Upper bound on HTML bodies we will buffer to inject into. Larger responses are streamed
	// through untouched (a multi-MB HTML document is almost certainly not a normal page).
	maxHTMLInjectBytes = 8 << 20 // 8 MiB
)

// Server is the browser-proxy HTTP server.
type Server struct {
	cfg          *config.Config
	snippet      []byte // default (no CSP nonce) injection tags
	configJS     []byte
	proxy        *httputil.ReverseProxy
	otlpClient   *http.Client
	ipLimiter    *rateLimiter
	tokenLimiter *rateLimiter
}

// New builds a Server from the given configuration.
func New(cfg *config.Config) (*Server, error) {
	if cfg.ExportToken == "" {
		tok, err := generateToken()
		if err != nil {
			return nil, fmt.Errorf("generate export token: %w", err)
		}
		cfg.ExportToken = tok
	}

	upstreamURL, err := url.Parse(cfg.Upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream URL %q: %w", cfg.Upstream, err)
	}

	configJS, err := buildConfigJS(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build config.js: %w", err)
	}

	s := &Server{
		cfg:          cfg,
		snippet:      buildSnippet(""),
		configJS:     configJS,
		otlpClient:   &http.Client{Timeout: 30 * time.Second},
		ipLimiter:    newRateLimiter(defaultOTLPPerIPPerMin, 30),
		tokenLimiter: newRateLimiter(defaultOTLPPerTokenPerMin, 60),
	}

	proxy := httputil.NewSingleHostReverseProxy(upstreamURL)
	defaultDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		defaultDirector(req)
		req.Host = upstreamURL.Host
		// Only accept encodings we can decode for injection. This avoids receiving brotli/zstd
		// HTML that we would otherwise have to skip. Non-HTML responses are passed through as-is.
		req.Header.Set("Accept-Encoding", "gzip")
	}
	proxy.ModifyResponse = s.injectResponse
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("browser-proxy: upstream error for %s: %v", r.URL.Path, err)
		w.WriteHeader(http.StatusBadGateway)
	}
	s.proxy = proxy

	return s, nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Handler returns the root HTTP handler with routing for the reserved /__odigos/ paths and the
// reverse proxy fallthrough.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(config.HealthPath, s.handleHealth)
	mux.HandleFunc(config.ConfigJsPath, s.handleConfigJS)
	mux.HandleFunc(config.AgentJsPath, s.handleAgentJS)
	// All OTLP signals (traces/metrics/logs) under the reserved prefix.
	mux.HandleFunc(config.OtlpPathPrefix, s.handleOTLP)
	mux.HandleFunc("/", s.handleProxy)
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

// Run starts the HTTP server and blocks.
func (s *Server) Run() error {
	srv := &http.Server{
		Addr:              s.cfg.ListenAddr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("browser-proxy: listening on %s, forwarding to %s (service=%q, collector=%s)",
		s.cfg.ListenAddr, s.cfg.Upstream, s.cfg.ServiceName, s.cfg.OtlpHTTPEndpoint)
	return srv.ListenAndServe()
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	s.proxy.ServeHTTP(w, r)
}

func (s *Server) handleConfigJS(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=60")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = w.Write(s.configJS)
}

// handleAgentJS serves the browser SDK bundle from the mounted agents directory.
func (s *Server) handleAgentJS(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(s.cfg.AgentDir, filepath.Base(s.cfg.AgentFile))
	f, err := os.Open(path)
	if err != nil {
		log.Printf("browser-proxy: agent bundle not found at %s: %v", path, err)
		http.Error(w, "agent bundle not available", http.StatusNotFound)
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		http.Error(w, "agent bundle not available", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, s.cfg.AgentFile, info.ModTime(), f)
}

// injectResponse is the ReverseProxy ModifyResponse hook. It injects the SDK snippet into HTML
// responses, decompressing/recompressing gzip as needed, and leaves all other responses untouched.
func (s *Server) injectResponse(resp *http.Response) error {
	if !isHTML(resp.Header.Get("Content-Type")) {
		return nil
	}

	encoding := strings.ToLower(strings.TrimSpace(resp.Header.Get("Content-Encoding")))
	if encoding != "" && encoding != "gzip" && encoding != "identity" {
		// We forced Accept-Encoding: gzip upstream, so this is unexpected; skip injection to be safe.
		return nil
	}

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxHTMLInjectBytes+1))
	if err != nil {
		return err
	}
	_ = resp.Body.Close()

	if len(raw) > maxHTMLInjectBytes {
		// Too large to safely buffer/inject; pass through unchanged.
		resp.Body = io.NopCloser(bytes.NewReader(raw))
		return nil
	}

	decoded := raw
	if encoding == "gzip" {
		gr, gzErr := gzip.NewReader(bytes.NewReader(raw))
		if gzErr != nil {
			// Not actually gzip / corrupt; pass through unchanged.
			resp.Body = io.NopCloser(bytes.NewReader(raw))
			return nil
		}
		decoded, err = io.ReadAll(gr)
		_ = gr.Close()
		if err != nil {
			resp.Body = io.NopCloser(bytes.NewReader(raw))
			return nil
		}
	}

	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		csp = resp.Header.Get("Content-Security-Policy-Report-Only")
	}
	snippet := s.snippetForResponse(csp)
	injected := injectIntoHTML(decoded, snippet)

	out := injected
	if encoding == "gzip" {
		var buf bytes.Buffer
		gw := gzip.NewWriter(&buf)
		if _, err := gw.Write(injected); err != nil {
			_ = gw.Close()
			resp.Header.Del("Content-Encoding")
			resp.Body = io.NopCloser(bytes.NewReader(injected))
			resp.ContentLength = int64(len(injected))
			resp.Header.Set("Content-Length", strconv.Itoa(len(injected)))
			return nil
		}
		if err := gw.Close(); err != nil {
			resp.Header.Del("Content-Encoding")
			resp.Body = io.NopCloser(bytes.NewReader(injected))
			resp.ContentLength = int64(len(injected))
			resp.Header.Set("Content-Length", strconv.Itoa(len(injected)))
			return nil
		}
		out = buf.Bytes()
		resp.Header.Set("Content-Encoding", "gzip")
	} else {
		resp.Header.Del("Content-Encoding")
	}

	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	resp.Header.Set("Content-Length", strconv.Itoa(len(out)))
	return nil
}

func isHTML(contentType string) bool {
	return strings.Contains(strings.ToLower(contentType), "text/html")
}
