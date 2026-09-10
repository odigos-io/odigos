package server

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

// browserConfig is the JSON shape written to window.__ODIGOS__ for the agent bundle to read.
// It mirrors the OdigosBrowserConfig contract in the opentelemetry-browser agent (src/config.ts).
type browserConfig struct {
	ServiceName                  string            `json:"serviceName,omitempty"`
	TracesPath                   string            `json:"tracesPath"`
	LogsPath                     string            `json:"logsPath"`
	ExportToken                  string            `json:"exportToken,omitempty"`
	ResourceAttributes           map[string]string `json:"resourceAttributes,omitempty"`
	PropagateTraceHeaderCorsUrls []string          `json:"propagateTraceHeaderCorsUrls,omitempty"`
}

var cspNonceRE = regexp.MustCompile(`(?i)'nonce-([^']+)'`)

// buildConfigJS renders the body of /__odigos/config.js (assigns window.__ODIGOS__).
func buildConfigJS(cfg *config.Config) ([]byte, error) {
	bc := browserConfig{
		ServiceName:                  cfg.ServiceName,
		TracesPath:                   config.TracesPath,
		LogsPath:                     config.LogsPath,
		ExportToken:                  cfg.ExportToken,
		ResourceAttributes:           parseResourceAttributes(cfg.ResourceAttributes),
		PropagateTraceHeaderCorsUrls: parseList(cfg.PropagateCorsUrls),
	}

	configJSON, err := json.Marshal(bc)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	b.WriteString("window.__ODIGOS__=")
	b.Write(configJSON)
	b.WriteString(";")
	return b.Bytes(), nil
}

// buildSnippet renders CSP-safe external <script> tags (no inline JS).
// When nonce is non-empty it is attached to both tags.
func buildSnippet(nonce string) []byte {
	var b bytes.Buffer
	writeScriptTag(&b, config.ConfigJsPath, false, nonce)
	writeScriptTag(&b, config.AgentJsPath, true, nonce)
	return b.Bytes()
}

func writeScriptTag(b *bytes.Buffer, src string, async bool, nonce string) {
	b.WriteString(`<script src="`)
	b.WriteString(src)
	b.WriteByte('"')
	if async {
		b.WriteString(" async")
	}
	if nonce != "" {
		b.WriteString(` nonce="`)
		b.WriteString(nonce)
		b.WriteByte('"')
	}
	b.WriteString("></script>")
}

// extractCSPNonce returns the first CSP nonce value from a Content-Security-Policy header, if any.
func extractCSPNonce(cspHeader string) string {
	m := cspNonceRE.FindStringSubmatch(cspHeader)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

// parseResourceAttributes parses an OTEL_RESOURCE_ATTRIBUTES-style string ("k1=v1,k2=v2").
func parseResourceAttributes(raw string) map[string]string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	out := map[string]string{}
	for _, pair := range strings.Split(raw, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		k, v, found := strings.Cut(pair, "=")
		k = strings.TrimSpace(k)
		if !found || k == "" {
			continue
		}
		out[k] = strings.TrimSpace(v)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func parseList(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []string
	for _, v := range strings.Split(raw, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// injectIntoHTML inserts snippet into the document at the best available location. It prefers to
// inject immediately after the opening <head> tag so the SDK initializes as early as possible.
// Matching is case-insensitive. If no suitable anchor is found, the snippet is prepended.
func injectIntoHTML(body, snippet []byte) []byte {
	lower := bytes.ToLower(body)

	// Inject right after the opening <head ...> tag.
	if idx := indexAfterTag(lower, "<head"); idx >= 0 {
		return spliceAt(body, snippet, idx)
	}
	// Fall back to just before </head>.
	if idx := bytes.Index(lower, []byte("</head>")); idx >= 0 {
		return spliceAt(body, snippet, idx)
	}
	// Fall back to right after the opening <body ...> tag.
	if idx := indexAfterTag(lower, "<body"); idx >= 0 {
		return spliceAt(body, snippet, idx)
	}
	// Fall back to just before </body>.
	if idx := bytes.Index(lower, []byte("</body>")); idx >= 0 {
		return spliceAt(body, snippet, idx)
	}
	// Last resort: prepend.
	return spliceAt(body, snippet, 0)
}

// indexAfterTag returns the byte offset just past the end ('>') of the first tag that starts with
// tagStart (e.g. "<head"). Returns -1 if not found or the tag is unterminated.
func indexAfterTag(lower []byte, tagStart string) int {
	start := bytes.Index(lower, []byte(tagStart))
	if start < 0 {
		return -1
	}
	end := bytes.IndexByte(lower[start:], '>')
	if end < 0 {
		return -1
	}
	return start + end + 1
}

func spliceAt(body, snippet []byte, idx int) []byte {
	out := make([]byte, 0, len(body)+len(snippet))
	out = append(out, body[:idx]...)
	out = append(out, snippet...)
	out = append(out, body[idx:]...)
	return out
}

// snippetForResponse picks a nonce-aware injection snippet for this HTML response.
func (s *Server) snippetForResponse(cspHeader string) []byte {
	nonce := extractCSPNonce(cspHeader)
	if nonce == "" {
		return s.snippet
	}
	return buildSnippet(nonce)
}
