package tools

import (
	"container/list"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

const (
	defaultPinnedCommit   = "37cf1aee"
	maxCitationLines      = 200
	maxCitationBytes      = 64 * 1024
	citationFetchTimeout  = 10 * time.Second
	citationCacheTTL      = 30 * time.Minute
	citationCacheMaxBytes = 20 * 1024 * 1024
	rawGithubBase         = "https://raw.githubusercontent.com/odigos-io/odigos"
)

// RegisterCitationTools wires the gh_read_file MCP tool onto the server.
func RegisterCitationTools(server *mcpserver.MCPServer) {
	manager := newCitationManager(http.DefaultClient, os.Getenv)
	manager.register(server)
}

type citationManager struct {
	httpClient   *http.Client
	pinnedCommit string
	githubToken  string
	cache        *citationCache
}

func newCitationManager(httpClient *http.Client, getenv func(string) string) *citationManager {
	pinned := strings.TrimSpace(getenv("ODIGOS_PINNED_COMMIT"))
	if pinned == "" {
		pinned = defaultPinnedCommit
	}
	timeoutClient := *httpClient
	if timeoutClient.Timeout == 0 {
		timeoutClient.Timeout = citationFetchTimeout
	}
	return &citationManager{
		httpClient:   &timeoutClient,
		pinnedCommit: pinned,
		githubToken:  strings.TrimSpace(getenv("GITHUB_TOKEN")),
		cache:        newCitationCache(citationCacheTTL, citationCacheMaxBytes),
	}
}

func (m *citationManager) register(server *mcpserver.MCPServer) {
	server.AddTool(mcp.NewTool(
		"gh_read_file",
		mcp.WithDescription("Fetch a slice of one file from raw.githubusercontent.com at the bundled graph's pinned odigos commit. Cap 200 lines / 64 KB per call. Use only to expand a snippet already located via the graph - never for exploration."),
		mcp.WithString("path", mcp.Required(), mcp.Description("Repo-relative file path, e.g. instrumentor/controllers/manager.go")),
		mcp.WithNumber("line_start", mcp.Required(), mcp.Description("First line to include (1-indexed).")),
		mcp.WithNumber("line_end", mcp.Required(), mcp.Description("Last line to include (1-indexed, inclusive). Must satisfy line_end - line_start + 1 <= 200.")),
	), m.ghReadFile)
}

func (m *citationManager) ghReadFile(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := request.RequireString("path")
	if err != nil {
		return ToolError("path required: %v", err)
	}
	if cleanPath, ok := sanitizeCitationPath(path); ok {
		path = cleanPath
	} else {
		return ToolError("invalid path %q: must be repo-relative, no scheme, no .. segments", path)
	}
	lineStart := request.GetInt("line_start", 0)
	lineEnd := request.GetInt("line_end", 0)
	if lineStart < 1 || lineEnd < lineStart {
		return ToolError("require line_start >= 1 and line_end >= line_start, got start=%d end=%d", lineStart, lineEnd)
	}
	if lineEnd-lineStart+1 > maxCitationLines {
		return ToolError("range too large: %d lines requested, max %d", lineEnd-lineStart+1, maxCitationLines)
	}

	url := fmt.Sprintf("%s/%s/%s", rawGithubBase, m.pinnedCommit, path)
	if cached, hit := m.cache.get(url); hit {
		slice, sliceErr := sliceLines(cached, lineStart, lineEnd)
		if sliceErr != nil {
			return ToolError("%v", sliceErr)
		}
		return WriteJSON(map[string]any{
			"path":       path,
			"commit":     m.pinnedCommit,
			"line_start": lineStart,
			"line_end":   lineEnd,
			"content":    slice,
			"cached":     true,
		})
	}

	body, fetchErr := m.fetch(ctx, url)
	if fetchErr != nil {
		return ToolError("fetch %s: %v", url, fetchErr)
	}
	m.cache.put(url, body)
	slice, sliceErr := sliceLines(body, lineStart, lineEnd)
	if sliceErr != nil {
		return ToolError("%v", sliceErr)
	}
	return WriteJSON(map[string]any{
		"path":       path,
		"commit":     m.pinnedCommit,
		"line_start": lineStart,
		"line_end":   lineEnd,
		"content":    slice,
		"cached":     false,
	})
}

func (m *citationManager) fetch(ctx context.Context, url string) (string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	if m.githubToken != "" {
		request.Header.Set("Authorization", "Bearer "+m.githubToken)
	}
	request.Header.Set("Accept", "text/plain")
	response, err := m.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusNotFound {
		return "", fmt.Errorf("file not found at pinned commit")
	}
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream status %d", response.StatusCode)
	}
	limit := int64(citationCacheMaxBytes / 4)
	bodyBytes, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return "", err
	}
	if int64(len(bodyBytes)) > limit {
		return "", fmt.Errorf("file too large: > %d bytes", limit)
	}
	return string(bodyBytes), nil
}

// sliceLines extracts lines [start, end] (1-indexed, inclusive) and caps the
// resulting slice at maxCitationBytes bytes.
func sliceLines(text string, start, end int) (string, error) {
	if start < 1 {
		return "", fmt.Errorf("line_start must be >= 1")
	}
	lines := strings.Split(text, "\n")
	if start > len(lines) {
		return "", fmt.Errorf("line_start %d exceeds file length %d", start, len(lines))
	}
	if end > len(lines) {
		end = len(lines)
	}
	excerpt := strings.Join(lines[start-1:end], "\n")
	if len(excerpt) > maxCitationBytes {
		excerpt = excerpt[:maxCitationBytes]
	}
	return excerpt, nil
}

// sanitizeCitationPath disallows absolute paths, schemes, and parent-dir
// segments. Returns the cleaned path on success.
func sanitizeCitationPath(input string) (string, bool) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", false
	}
	if strings.Contains(trimmed, "://") {
		return "", false
	}
	if strings.HasPrefix(trimmed, "/") {
		return "", false
	}
	for _, segment := range strings.Split(trimmed, "/") {
		if segment == ".." {
			return "", false
		}
	}
	return trimmed, true
}

// ---- LRU cache ----
//
// Bounded LRU keyed by full URL. Each entry stores the response body and a
// timestamp. Eviction happens on Put when (size_bytes > max_bytes) OR
// (entry.created_at + ttl < now). Get returns false if the entry has expired
// without removing it on the read path (to keep Get cheap and lock-free-ish);
// the next Put cleans up.

type citationCacheEntry struct {
	url       string
	value     string
	bytes     int
	createdAt time.Time
}

type citationCache struct {
	mutex      sync.Mutex
	list       *list.List
	entries    map[string]*list.Element
	ttl        time.Duration
	maxBytes   int
	totalBytes int
	now        func() time.Time
}

func newCitationCache(ttl time.Duration, maxBytes int) *citationCache {
	return &citationCache{
		list:     list.New(),
		entries:  map[string]*list.Element{},
		ttl:      ttl,
		maxBytes: maxBytes,
		now:      time.Now,
	}
}

func (c *citationCache) get(url string) (string, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	element, ok := c.entries[url]
	if !ok {
		return "", false
	}
	entry := element.Value.(*citationCacheEntry)
	if c.now().Sub(entry.createdAt) > c.ttl {
		c.list.Remove(element)
		delete(c.entries, url)
		c.totalBytes -= entry.bytes
		return "", false
	}
	c.list.MoveToFront(element)
	return entry.value, true
}

func (c *citationCache) put(url, value string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if existing, ok := c.entries[url]; ok {
		entry := existing.Value.(*citationCacheEntry)
		c.totalBytes -= entry.bytes
		c.list.Remove(existing)
		delete(c.entries, url)
	}
	entry := &citationCacheEntry{
		url:       url,
		value:     value,
		bytes:     len(value),
		createdAt: c.now(),
	}
	element := c.list.PushFront(entry)
	c.entries[url] = element
	c.totalBytes += entry.bytes
	c.evictExpiredLocked()
	c.evictUntilWithinBudgetLocked()
}

func (c *citationCache) Size() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return len(c.entries)
}

func (c *citationCache) TotalBytes() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.totalBytes
}

func (c *citationCache) evictExpiredLocked() {
	now := c.now()
	for element := c.list.Back(); element != nil; {
		entry := element.Value.(*citationCacheEntry)
		previous := element.Prev()
		if now.Sub(entry.createdAt) > c.ttl {
			c.list.Remove(element)
			delete(c.entries, entry.url)
			c.totalBytes -= entry.bytes
		}
		element = previous
	}
}

func (c *citationCache) evictUntilWithinBudgetLocked() {
	for c.totalBytes > c.maxBytes && c.list.Len() > 0 {
		element := c.list.Back()
		entry := element.Value.(*citationCacheEntry)
		c.list.Remove(element)
		delete(c.entries, entry.url)
		c.totalBytes -= entry.bytes
	}
}

// CitationCacheSize is a test helper.
func (m *citationManager) CitationCacheSize() int { return m.cache.Size() }

// CitationCacheBytes is a test helper.
func (m *citationManager) CitationCacheBytes() int { return m.cache.TotalBytes() }
