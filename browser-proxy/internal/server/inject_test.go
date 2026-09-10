package server

import (
	"strings"
	"testing"

	"github.com/odigos-io/odigos/browser-proxy/internal/config"
)

func TestInjectIntoHTML_AfterHead(t *testing.T) {
	body := []byte("<html><head><title>x</title></head><body>hi</body></html>")
	snippet := []byte("<!--SNIP-->")
	out := string(injectIntoHTML(body, snippet))

	if !strings.Contains(out, "<head><!--SNIP--><title>") {
		t.Fatalf("snippet not injected right after <head>: %s", out)
	}
}

func TestInjectIntoHTML_HeadWithAttributes(t *testing.T) {
	body := []byte(`<head data-x="1">content`)
	snippet := []byte("S")
	out := string(injectIntoHTML(body, snippet))
	if !strings.HasPrefix(out, `<head data-x="1">S`) {
		t.Fatalf("snippet not injected after head tag with attributes: %s", out)
	}
}

func TestInjectIntoHTML_FallbackBody(t *testing.T) {
	body := []byte("<body>only body</body>")
	snippet := []byte("S")
	out := string(injectIntoHTML(body, snippet))
	if !strings.HasPrefix(out, "<body>S") {
		t.Fatalf("snippet not injected after <body>: %s", out)
	}
}

func TestInjectIntoHTML_FallbackPrepend(t *testing.T) {
	body := []byte("no tags here")
	snippet := []byte("S")
	out := string(injectIntoHTML(body, snippet))
	if !strings.HasPrefix(out, "Sno tags") {
		t.Fatalf("snippet not prepended: %s", out)
	}
}

func TestInjectIntoHTML_CaseInsensitive(t *testing.T) {
	body := []byte("<HTML><HEAD></HEAD></HTML>")
	snippet := []byte("S")
	out := string(injectIntoHTML(body, snippet))
	if !strings.Contains(out, "<HEAD>S") {
		t.Fatalf("case-insensitive head match failed: %s", out)
	}
}

func TestBuildSnippetExternalOnly(t *testing.T) {
	s := string(buildSnippet(""))
	if strings.Contains(s, "window.__ODIGOS__") {
		t.Fatalf("snippet must not contain inline config: %s", s)
	}
	if !strings.Contains(s, `src="`+config.ConfigJsPath+`"`) {
		t.Fatalf("missing config.js script tag: %s", s)
	}
	if !strings.Contains(s, `src="`+config.AgentJsPath+`"`) {
		t.Fatalf("missing agent script tag: %s", s)
	}
	if strings.Count(s, "<script") != 2 {
		t.Fatalf("expected exactly 2 script tags: %s", s)
	}
}

func TestBuildSnippetWithNonce(t *testing.T) {
	s := string(buildSnippet("n-1"))
	if strings.Count(s, `nonce="n-1"`) != 2 {
		t.Fatalf("expected nonce on both tags: %s", s)
	}
}

func TestBuildConfigJS(t *testing.T) {
	cfg := &config.Config{
		ServiceName:        "my-frontend",
		ResourceAttributes: "k8s.namespace.name=demo,k8s.pod.name=p1",
		PropagateCorsUrls:  "https://api.example.com,/.*backend.*/",
		ExportToken:        "tok",
	}
	body, err := buildConfigJS(cfg)
	if err != nil {
		t.Fatalf("buildConfigJS error: %v", err)
	}
	s := string(body)

	if !strings.HasPrefix(s, "window.__ODIGOS__=") {
		t.Fatalf("missing config assignment: %s", s)
	}
	if !strings.Contains(s, `"serviceName":"my-frontend"`) {
		t.Fatalf("missing service name: %s", s)
	}
	if !strings.Contains(s, `"tracesPath":"`+config.TracesPath+`"`) {
		t.Fatalf("missing traces path: %s", s)
	}
	if !strings.Contains(s, `"logsPath":"`+config.LogsPath+`"`) {
		t.Fatalf("missing logs path: %s", s)
	}
	if !strings.Contains(s, `"exportToken":"tok"`) {
		t.Fatalf("missing export token: %s", s)
	}
	if !strings.Contains(s, "k8s.namespace.name") || !strings.Contains(s, "demo") {
		t.Fatalf("missing resource attributes: %s", s)
	}
}

func TestExtractCSPNonce(t *testing.T) {
	if got := extractCSPNonce(`default-src 'self'; script-src 'nonce-XYZ' 'self'`); got != "XYZ" {
		t.Fatalf("got %q", got)
	}
	if got := extractCSPNonce(`default-src 'self'`); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestParseResourceAttributes(t *testing.T) {
	got := parseResourceAttributes(" a = 1 , b=2 , ,c= ")
	if got["a"] != "1" || got["b"] != "2" || got["c"] != "" {
		t.Fatalf("unexpected parse result: %#v", got)
	}
	if parseResourceAttributes("") != nil {
		t.Fatalf("empty input should return nil")
	}
}

func TestHostsEqual(t *testing.T) {
	if !hostsEqual("frontend.example.com", "frontend.example.com") {
		t.Fatal("equal hosts")
	}
	if !hostsEqual("frontend.example.com:443", "frontend.example.com") {
		t.Fatal("host with port vs without")
	}
	if hostsEqual("evil.example.com", "frontend.example.com") {
		t.Fatal("different hosts")
	}
}
