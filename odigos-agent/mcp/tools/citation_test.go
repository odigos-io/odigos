package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// roundTripperFunc lets us inject deterministic responses without standing up
// an actual HTTP server.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func newCitationManagerWithRoundTripper(rt http.RoundTripper, env map[string]string) *citationManager {
	httpClient := &http.Client{Transport: rt, Timeout: 2 * time.Second}
	getenv := func(key string) string { return env[key] }
	return newCitationManager(httpClient, getenv)
}

func TestSanitizeCitationPath(t *testing.T) {
	cases := []struct {
		name  string
		input string
		valid bool
	}{
		{"normal", "instrumentor/manager.go", true},
		{"with leading slash", "/etc/passwd", false},
		{"with scheme", "https://example.com/x", false},
		{"with parent dir", "../etc/passwd", false},
		{"with parent mid", "ok/../etc/passwd", false},
		{"empty", "   ", false},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			_, ok := sanitizeCitationPath(testCase.input)
			if ok != testCase.valid {
				t.Errorf("got %v want %v", ok, testCase.valid)
			}
		})
	}
}

func TestSliceLinesBasic(t *testing.T) {
	text := "alpha\nbeta\ngamma\ndelta\nepsilon\n"
	got, err := sliceLines(text, 2, 4)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "beta\ngamma\ndelta"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestSliceLinesStartBeyondFile(t *testing.T) {
	if _, err := sliceLines("only one line\n", 99, 100); err == nil {
		t.Error("expected error when start exceeds file length")
	}
}

func TestSliceLinesEndClampedToFile(t *testing.T) {
	got, err := sliceLines("a\nb\nc\n", 1, 999)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "c") {
		t.Errorf("unexpected slice: %q", got)
	}
}

func TestOversizedRangeArithmetic(t *testing.T) {
	// The handler guards `line_end - line_start + 1 > maxCitationLines`.
	// Lock the constant so a future change doesn't silently shrink the cap.
	if maxCitationLines != 200 {
		t.Errorf("max citation lines: got %d want 200", maxCitationLines)
	}
}

func TestCitationManagerFetchUsesPinnedCommit(t *testing.T) {
	calls := 0
	var requestedURL string
	manager := newCitationManagerWithRoundTripper(roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		requestedURL = request.URL.String()
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("line1\nline2\nline3\n")),
			Header:     http.Header{},
		}, nil
	}), map[string]string{"ODIGOS_PINNED_COMMIT": "deadbeef"})

	body, err := manager.fetch(context.Background(), fmt.Sprintf("%s/%s/foo.go", rawGithubBase, "deadbeef"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
	if !strings.Contains(requestedURL, "deadbeef") {
		t.Errorf("expected pinned commit in URL, got %q", requestedURL)
	}
	if !strings.Contains(body, "line2") {
		t.Errorf("body missing expected content: %q", body)
	}
}

func TestCitationManagerSendsAuthHeaderWhenTokenSet(t *testing.T) {
	var sawAuth string
	manager := newCitationManagerWithRoundTripper(roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		sawAuth = request.Header.Get("Authorization")
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok\n")),
			Header:     http.Header{},
		}, nil
	}), map[string]string{"GITHUB_TOKEN": "test-token"})

	if _, err := manager.fetch(context.Background(), "https://example.com/x"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sawAuth != "Bearer test-token" {
		t.Errorf("expected Authorization Bearer header, got %q", sawAuth)
	}
}

func TestCitationCacheRoundtrip(t *testing.T) {
	cache := newCitationCache(time.Hour, 1024)
	cache.put("url1", "content1")
	value, ok := cache.get("url1")
	if !ok {
		t.Fatal("expected cache hit")
	}
	if value != "content1" {
		t.Errorf("got %q want %q", value, "content1")
	}
}

func TestCitationCacheTTLEviction(t *testing.T) {
	cache := newCitationCache(50*time.Millisecond, 1024)
	clock := time.Unix(0, 0)
	cache.now = func() time.Time { return clock }
	cache.put("url1", "content1")
	clock = clock.Add(time.Second)
	if _, ok := cache.get("url1"); ok {
		t.Error("expected entry to expire")
	}
}

func TestCitationCacheEvictsToWithinByteBudget(t *testing.T) {
	cache := newCitationCache(time.Hour, 10)
	cache.put("a", "12345")
	cache.put("b", "67890")
	cache.put("c", "ABCDE") // 15 bytes total > budget; oldest gets evicted
	if _, ok := cache.get("a"); ok {
		t.Error("oldest entry should have been evicted")
	}
	if _, ok := cache.get("c"); !ok {
		t.Error("newest entry should remain")
	}
	if cache.TotalBytes() > 10 {
		t.Errorf("total bytes %d exceeds budget", cache.TotalBytes())
	}
}

func TestCitationCacheGetUnknownReturnsFalse(t *testing.T) {
	cache := newCitationCache(time.Hour, 1024)
	if _, ok := cache.get("nope"); ok {
		t.Error("unknown key must miss")
	}
}

func TestCitationCachePutSameKeyUpdatesValue(t *testing.T) {
	cache := newCitationCache(time.Hour, 1024)
	cache.put("k", "first")
	cache.put("k", "second")
	value, ok := cache.get("k")
	if !ok || value != "second" {
		t.Errorf("got (%q, %v) want (\"second\", true)", value, ok)
	}
	if cache.Size() != 1 {
		t.Errorf("expected size 1, got %d", cache.Size())
	}
}
