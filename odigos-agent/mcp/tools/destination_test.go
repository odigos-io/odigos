package tools

import (
	"strings"
	"testing"
)

func TestSummarizeSecretKeysHidesValues(t *testing.T) {
	data := map[string][]byte{
		"API_TOKEN":  []byte("super-secret-do-not-leak"),
		"EMPTY":      []byte(""),
		"WHITESPACE": []byte("   "),
	}
	got := summarizeSecretKeys(data)
	if len(got) != 3 {
		t.Fatalf("want 3 entries, got %d", len(got))
	}
	// Sort order is deterministic; "API_TOKEN" comes first alphabetically.
	for _, entry := range got {
		if entry["key"] == "API_TOKEN" {
			if entry["length"] != len(data["API_TOKEN"]) {
				t.Errorf("API_TOKEN length wrong: %v", entry["length"])
			}
			if entry["looks_empty"] != false {
				t.Errorf("API_TOKEN should not look empty")
			}
		}
		if entry["key"] == "EMPTY" || entry["key"] == "WHITESPACE" {
			if entry["looks_empty"] != true {
				t.Errorf("%v should look empty", entry["key"])
			}
		}
		if _, hasValue := entry["value"]; hasValue {
			t.Errorf("entry must not include a value field: %v", entry)
		}
	}

	// Belt-and-suspenders: serialize all keys and assert no raw value appears.
	flat := ""
	for _, entry := range got {
		for _, value := range entry {
			flat += " "
			if str, ok := value.(string); ok {
				flat += str
			}
		}
	}
	if strings.Contains(flat, "super-secret-do-not-leak") {
		t.Fatal("raw secret value leaked into summarized output")
	}
}

func TestPickDestinationEndpointHonorsPriority(t *testing.T) {
	data := map[string]string{
		"OTLP_GRPC_ENDPOINT": "grpcs://primary:4317",
		"endpoint":           "https://fallback:443",
	}
	endpoint, sourceKey, found := pickDestinationEndpoint(data)
	if !found {
		t.Fatal("expected to find an endpoint")
	}
	if endpoint != "grpcs://primary:4317" {
		t.Errorf("expected primary endpoint, got %q", endpoint)
	}
	if sourceKey != "OTLP_GRPC_ENDPOINT" {
		t.Errorf("expected OTLP_GRPC_ENDPOINT, got %q", sourceKey)
	}
}

func TestPickDestinationEndpointReturnsFalseWhenAllEmpty(t *testing.T) {
	data := map[string]string{
		"endpoint": "   ",
		"url":      "",
	}
	_, _, found := pickDestinationEndpoint(data)
	if found {
		t.Error("blank values must not be reported as found")
	}
}

func TestNormalizeEndpointURLAcceptsBareHostPort(t *testing.T) {
	parsed, err := normalizeEndpointURL("api.example.com:4317")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Hostname() != "api.example.com" {
		t.Errorf("hostname: got %q", parsed.Hostname())
	}
	if parsed.Port() != "4317" {
		t.Errorf("port: got %q", parsed.Port())
	}
}

func TestNormalizeEndpointURLAcceptsHTTPSURL(t *testing.T) {
	parsed, err := normalizeEndpointURL("https://otlp.example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if parsed.Scheme != "https" {
		t.Errorf("scheme: got %q", parsed.Scheme)
	}
	if parsed.Hostname() != "otlp.example.com" {
		t.Errorf("hostname: got %q", parsed.Hostname())
	}
}

func TestNormalizeEndpointURLRejectsEmpty(t *testing.T) {
	if _, err := normalizeEndpointURL("   "); err == nil {
		t.Error("expected error on empty endpoint")
	}
}

func TestDefaultPortForScheme(t *testing.T) {
	cases := map[string]string{
		"https": "443",
		"grpcs": "443",
		"http":  "80",
		"":      "4317",
		"foo":   "4317",
	}
	for scheme, want := range cases {
		if got := defaultPortForScheme(scheme); got != want {
			t.Errorf("scheme %q: got %q want %q", scheme, got, want)
		}
	}
}

func TestSchemeUsesTLS(t *testing.T) {
	for _, scheme := range []string{"https", "grpcs", "HTTPS", "tls"} {
		if !schemeUsesTLS(scheme) {
			t.Errorf("%q should use TLS", scheme)
		}
	}
	for _, scheme := range []string{"http", "grpc", "tcp", "", "ftp"} {
		if schemeUsesTLS(scheme) {
			t.Errorf("%q should not use TLS", scheme)
		}
	}
}

func TestFindExportersForDestination(t *testing.T) {
	parsed := map[string]any{
		"exporters": map[string]any{
			"otlp/datadog":   map[string]any{"endpoint": "https://api.datadoghq.com"},
			"otlp/honeycomb": map[string]any{"endpoint": "https://api.honeycomb.io"},
			"logging":        map[string]any{},
		},
	}
	got := findExportersForDestination(parsed, "datadog")
	if len(got) != 1 {
		t.Fatalf("expected 1 match, got %d (%v)", len(got), got)
	}
	if _, has := got["otlp/datadog"]; !has {
		t.Error("expected otlp/datadog match")
	}
}

func TestFindExportersForDestinationRejectsAccidentalSubstringMatches(t *testing.T) {
	parsed := map[string]any{
		"exporters": map[string]any{
			"otlp/datadog":         map[string]any{},
			"otlp/datadog-staging": map[string]any{},
			"otlp/not-datadog":     map[string]any{},
			"awsxray/aws-traces":   map[string]any{},
			"logging":              map[string]any{},
		},
	}
	got := findExportersForDestination(parsed, "datadog")
	if _, has := got["otlp/datadog"]; !has {
		t.Error("expected otlp/datadog match")
	}
	if _, has := got["otlp/datadog-staging"]; has {
		t.Error("otlp/datadog-staging is a different destination - must not match")
	}
	if _, has := got["otlp/not-datadog"]; has {
		t.Error("otlp/not-datadog has the destination name as a suffix only, must not match")
	}

	// Short destination name (the actual bug the substring approach hid).
	got = findExportersForDestination(parsed, "aws")
	if _, has := got["awsxray/aws-traces"]; has {
		t.Error("destination 'aws' must not bind to awsxray/aws-traces")
	}
}

func TestExporterKeyMatchesDestination(t *testing.T) {
	cases := []struct {
		key  string
		name string
		want bool
	}{
		{"otlp/datadog", "datadog", true},
		{"datadog", "datadog", true},
		{"otlp/datadog-staging", "datadog", false},
		{"otlp/not-datadog", "datadog", false},
		{"otlp/datadog-traces", "datadog", false},
		{"logging", "datadog", false},
		{"", "datadog", false},
	}
	for _, testCase := range cases {
		got := exporterKeyMatchesDestination(testCase.key, testCase.name)
		if got != testCase.want {
			t.Errorf("(%q, %q): got %v want %v", testCase.key, testCase.name, got, testCase.want)
		}
	}
}

func TestFindExportersForDestinationNoExportersKey(t *testing.T) {
	parsed := map[string]any{"service": map[string]any{}}
	got := findExportersForDestination(parsed, "datadog")
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

func TestFindPipelinesUsingExporters(t *testing.T) {
	parsed := map[string]any{
		"service": map[string]any{
			"pipelines": map[string]any{
				"traces": map[string]any{
					"receivers": []any{"otlp"},
					"exporters": []any{"otlp/datadog", "logging"},
				},
				"metrics": map[string]any{
					"receivers": []any{"otlp"},
					"exporters": []any{"otlp/honeycomb"},
				},
			},
		},
	}
	got := findPipelinesUsingExporters(parsed, []string{"otlp/datadog"})
	if len(got) != 1 {
		t.Fatalf("expected 1 pipeline, got %d", len(got))
	}
	if got[0]["pipeline"] != "traces" {
		t.Errorf("expected traces pipeline, got %v", got[0]["pipeline"])
	}
}

func TestFilterExportErrorLines(t *testing.T) {
	logs := strings.Join([]string{
		`2026-05-16T10:00:00Z info exporter otlp/datadog start`,
		`2026-05-16T10:00:01Z error exporter otlp/datadog failed to send: 429 too many requests`,
		`2026-05-16T10:00:02Z info exporter otlp/honeycomb timeout`,
		`2026-05-16T10:00:03Z error exporter otlp/datadog connection refused`,
		`2026-05-16T10:00:04Z info something unrelated`,
	}, "\n")
	got := filterExportErrorLines(logs, "datadog")
	if !strings.Contains(got, "429") {
		t.Errorf("missing 429 error in %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("missing connection refused in %q", got)
	}
	if strings.Contains(got, "honeycomb") {
		t.Errorf("non-matching destination must not appear: %q", got)
	}
	if strings.Contains(got, "unrelated") {
		t.Errorf("non-error line must not appear: %q", got)
	}
	if strings.Contains(got, "start") {
		t.Errorf("info line must not appear: %q", got)
	}
}

func TestFilterExportErrorLinesEmptyOnNoDestinationName(t *testing.T) {
	if got := filterExportErrorLines("error something", ""); got != "" {
		t.Errorf("empty destination_name must yield empty output, got %q", got)
	}
}

func TestCountLines(t *testing.T) {
	cases := map[string]int{
		"":           0,
		"one":        1,
		"one\n":      2,
		"one\ntwo":   2,
		"one\ntwo\n": 3,
	}
	for input, want := range cases {
		if got := countLines(input); got != want {
			t.Errorf("countLines(%q): got %d want %d", input, got, want)
		}
	}
}

func TestTLSVersionString(t *testing.T) {
	if got := tlsVersionString(0x0303); got != "TLS 1.2" {
		t.Errorf("0x0303: got %q", got)
	}
	if got := tlsVersionString(0x0304); got != "TLS 1.3" {
		t.Errorf("0x0304: got %q", got)
	}
	if got := tlsVersionString(0x9999); !strings.HasPrefix(got, "0x") {
		t.Errorf("unknown version: got %q", got)
	}
}
