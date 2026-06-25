package server

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

// corsHeaders sets permissive CORS headers so the browser SDK (running on the application's own
// origin) can POST OTLP/HTTP telemetry to the sidecar. Since the sidecar shares the application's
// origin, this is effectively same-origin; the headers also cover the case where the page was
// loaded from a different origin (e.g. a CDN) and still posts back here.
func corsHeaders(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = "*"
	}
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Vary", "Origin")
	h.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "content-type, traceparent, tracestate, baggage")
	h.Set("Access-Control-Max-Age", "86400")
}

// handleOTLP forwards browser OTLP/HTTP telemetry to the node-local collector. The browser posts to
// a same-origin path under /__odigos/v1/ (e.g. /__odigos/v1/traces); the sidecar maps it to the
// collector's /v1/<signal> path and adds CORS headers to the response.
func (s *Server) handleOTLP(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w, r)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Map /__odigos/v1/<signal> -> <collector>/v1/<signal>.
	signalPath := strings.TrimPrefix(r.URL.Path, config.OtlpPathPrefix)
	targetURL := s.cfg.OtlpHTTPEndpoint + "/v1/" + signalPath

	body, err := io.ReadAll(io.LimitReader(r.Body, maxOTLPBodyBytes))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}
	// Preserve the payload framing so the collector can decode it (protobuf or json, possibly gzip).
	copyHeader(req.Header, r.Header, "Content-Type")
	copyHeader(req.Header, r.Header, "Content-Encoding")

	resp, err := s.otlpClient.Do(req)
	if err != nil {
		// The browser cannot reach the collector directly; swallow upstream errors as 502 but keep
		// the page healthy (telemetry loss must never surface to end users).
		log.Printf("browser-proxy: failed to forward OTLP to %s: %v", targetURL, err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header, "Content-Type")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(resp.Body, maxOTLPBodyBytes))
}

func copyHeader(dst, src http.Header, key string) {
	if v := src.Get(key); v != "" {
		dst.Set(key, v)
	}
}
