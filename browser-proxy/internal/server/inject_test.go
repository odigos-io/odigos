package server

import (
	"bytes"
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

func TestBuildSnippet(t *testing.T) {
	cfg := &config.Config{
		ServiceName:        "my-frontend",
		ResourceAttributes: "k8s.namespace.name=demo,k8s.pod.name=p1",
		PropagateCorsUrls:  "https://api.example.com,/.*backend.*/",
	}
	snippet, err := buildSnippet(cfg)
	if err != nil {
		t.Fatalf("buildSnippet error: %v", err)
	}
	s := string(snippet)

	if !strings.Contains(s, "window.__ODIGOS__=") {
		t.Fatalf("missing config assignment: %s", s)
	}
	if !strings.Contains(s, `"serviceName":"my-frontend"`) {
		t.Fatalf("missing service name: %s", s)
	}
	if !strings.Contains(s, `"tracesPath":"`+config.TracesPath+`"`) {
		t.Fatalf("missing traces path: %s", s)
	}
	if !strings.Contains(s, "k8s.namespace.name") || !strings.Contains(s, "demo") {
		t.Fatalf("missing resource attributes: %s", s)
	}
	if !strings.Contains(s, `src="`+config.AgentJsPath+`"`) {
		t.Fatalf("missing agent script tag: %s", s)
	}
	// json.Marshal must escape '<' to avoid breaking out of the inline <script>.
	if bytes.Contains(snippet, []byte("</script><script")) && strings.Count(s, "<script") != 2 {
		t.Fatalf("unexpected extra script tags (possible injection break): %s", s)
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
