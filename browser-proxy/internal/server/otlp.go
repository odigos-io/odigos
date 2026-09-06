package server

import (
	"bytes"
	"crypto/subtle"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

const (
	defaultOTLPPerIPPerMin    = 120
	defaultOTLPPerTokenPerMin = 240
	// Tighter than the previous 16 MiB — browser batches are small; large bodies are abuse.
	maxOTLPBodyBytes = 1 << 20 // 1 MiB
)

// corsHeaders sets CORS for OTLP only when the request Origin passes the same-site check.
// Never emits Access-Control-Allow-Origin: *.
func (s *Server) corsHeaders(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if !sameSiteOrigin(origin, r.Host) {
		return false
	}
	h := w.Header()
	h.Set("Access-Control-Allow-Origin", origin)
	h.Set("Vary", "Origin")
	h.Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "content-type, authorization, traceparent, tracestate, baggage")
	h.Set("Access-Control-Max-Age", "86400")
	return true
}

// handleOTLP authenticates, rate-limits, and forwards browser OTLP/HTTP to the node-local collector.
func (s *Server) handleOTLP(w http.ResponseWriter, r *http.Request) {
	if !s.corsHeaders(w, r) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !s.validateSameSite(r) {
		http.Error(w, "cross-site request blocked", http.StatusForbidden)
		return
	}

	token := bearerToken(r)
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.cfg.ExportToken)) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	ip := clientIP(r)
	if !s.ipLimiter.allow(ip) || !s.tokenLimiter.allow(token) {
		http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Map /__odigos/v1/<signal> -> <collector>/v1/<signal>.
	signalPath := strings.TrimPrefix(r.URL.Path, config.OtlpPathPrefix)
	targetURL := s.cfg.OtlpHTTPEndpoint + "/v1/" + signalPath

	body, err := io.ReadAll(io.LimitReader(r.Body, maxOTLPBodyBytes+1))
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	if len(body) > maxOTLPBodyBytes {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, targetURL, bytes.NewReader(body))
	if err != nil {
		http.Error(w, "failed to build upstream request", http.StatusInternalServerError)
		return
	}
	// Preserve the payload framing so the collector can decode it (protobuf or json, possibly gzip).
	// Do NOT forward Authorization — the export token is gateway-local only.
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

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(h) < len(prefix) || !strings.EqualFold(h[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(h[len(prefix):])
}

// validateSameSite rejects cross-site OTLP POSTs when Origin or Referer is present and mismatches Host.
func (s *Server) validateSameSite(r *http.Request) bool {
	if origin := r.Header.Get("Origin"); origin != "" {
		return sameSiteOrigin(origin, r.Host)
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return sameSiteOrigin(referer, r.Host)
	}
	return true
}

func sameSiteOrigin(originOrURL, requestHost string) bool {
	u, err := url.Parse(originOrURL)
	if err != nil || u.Host == "" {
		return false
	}
	return hostsEqual(u.Host, requestHost)
}

func hostsEqual(a, b string) bool {
	ah, ap, _ := net.SplitHostPort(a)
	if ah == "" {
		ah = a
	}
	bh, bp, _ := net.SplitHostPort(b)
	if bh == "" {
		bh = b
	}
	if !strings.EqualFold(ah, bh) {
		return false
	}
	// If both specify ports, require equality; if one omits port, treat as match on hostname.
	if ap != "" && bp != "" && ap != bp {
		return false
	}
	return true
}

func copyHeader(dst, src http.Header, key string) {
	if v := src.Get(key); v != "" {
		dst.Set(key, v)
	}
}
