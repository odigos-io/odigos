// Package tools wires the cluster-state MCP tools onto the mcp-go server.
//
// Tools are grouped by domain (source, collector, destination, citation) and
// each domain has its own file. Shared plumbing - kube clients, approval
// cache, JSON/error helpers - lives here.
package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	odigosclient "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	defaultOdigosNamespace = "odigos-system"
	defaultApprovalTTL     = 5 * time.Minute
)

// Clients bundles every kube client the MCP tools need.
type Clients struct {
	Core   kubernetes.Interface
	Odigos odigosclient.Interface
	Config *rest.Config
}

// BuildClients dials the cluster, preferring in-cluster credentials and
// falling back to a local kubeconfig for dev. Returns ready-to-use typed
// clients for core/apps and odigos CRDs.
func BuildClients() (*Clients, error) {
	config, err := buildRESTConfig()
	if err != nil {
		return nil, fmt.Errorf("build kube config: %w", err)
	}
	core, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("build core client: %w", err)
	}
	odigos, err := odigosclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("build odigos client: %w", err)
	}
	return &Clients{Core: core, Odigos: odigos, Config: config}, nil
}

func buildRESTConfig() (*rest.Config, error) {
	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, _ := os.UserHomeDir()
		kubeconfig = filepath.Join(home, ".kube", "config")
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// OdigosNamespace returns the namespace odigos system components run in.
// Reads CURRENT_NS (matching common/consts.CurrentNamespaceEnvVar) and falls
// back to "odigos-system" so local dev with kubeconfig works out of the box.
func OdigosNamespace() string {
	if namespace := os.Getenv("CURRENT_NS"); namespace != "" {
		return namespace
	}
	return defaultOdigosNamespace
}

// PendingMutation captures the dry-run state of a proposed mutation, awaiting
// user approval via apply_*. Fields are deliberately concrete - the agent
// rebuilds the runtime object from these on apply rather than trusting cached
// bytes.
type PendingMutation struct {
	Operation    string
	Namespace    string
	WorkloadKind string
	WorkloadName string
	YAML         string
	Diff         string
	RollbackHint string
	CreatedAt    time.Time
}

// ApprovalCache stores proposed mutations keyed by an opaque request_id. Pure
// in-memory; v1's MCP runs as a single replica so we don't need cross-process
// state. Entries past the TTL are dropped on the next Put/Take.
type ApprovalCache struct {
	mutex   sync.Mutex
	entries map[string]*PendingMutation
	ttl     time.Duration
	now     func() time.Time
}

// NewApprovalCache returns a cache with the given TTL. Pass 0 for the default
// (5 minutes).
func NewApprovalCache(ttl time.Duration) *ApprovalCache {
	if ttl <= 0 {
		ttl = defaultApprovalTTL
	}
	return &ApprovalCache{
		entries: map[string]*PendingMutation{},
		ttl:     ttl,
		now:     time.Now,
	}
}

// Put stores the mutation under a fresh UUID v4 request_id and returns the id.
func (c *ApprovalCache) Put(mutation *PendingMutation) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.gcLocked()
	id := uuid.NewString()
	mutation.CreatedAt = c.now()
	c.entries[id] = mutation
	return id
}

// Take pops the mutation for the given request_id. Returns nil if the id is
// unknown or its entry has expired.
func (c *ApprovalCache) Take(id string) *PendingMutation {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.gcLocked()
	mutation, ok := c.entries[id]
	if !ok {
		return nil
	}
	delete(c.entries, id)
	return mutation
}

// Size reports the current live entry count. Test-only.
func (c *ApprovalCache) Size() int {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.gcLocked()
	return len(c.entries)
}

func (c *ApprovalCache) gcLocked() {
	cutoff := c.now().Add(-c.ttl)
	for id, mutation := range c.entries {
		if mutation.CreatedAt.Before(cutoff) {
			delete(c.entries, id)
		}
	}
}

// WriteJSON wraps a value as a structured-only MCP tool result.
func WriteJSON(value any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultStructuredOnly(value), nil
}

// ToolError returns an MCP tool result flagged as an error. Callers should
// return (result, nil) - the protocol surfaces IsError to the LLM rather than
// transporting a Go error.
func ToolError(format string, args ...any) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultError(fmt.Sprintf(format, args...)), nil
}

// TailSlice returns up to the last n elements of xs. n<=0 returns xs unchanged.
func TailSlice[T any](xs []T, n int) []T {
	if n <= 0 || len(xs) <= n {
		return xs
	}
	return xs[len(xs)-n:]
}

// ClampInt clamps v into [low, high]. Convenience for tool-arg bounds.
func ClampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}
