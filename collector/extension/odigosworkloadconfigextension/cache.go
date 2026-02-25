package odigosworkloadconfigextension

import (
	"fmt"
	"strings"
	"sync"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// WorkloadCacheKey returns the cache key for a workload: "namespace/kind/name".
// Kind is the workload kind (e.g. Deployment, StatefulSet).
func WorkloadCacheKey(namespace, kind, name string) string {
	return namespace + "/" + kind + "/" + name
}

// WorkloadSamplingConfig holds the sampling configuration for a single workload,
// derived from an InstrumentationConfig. Uses api mirror types for head sampling and collector config.
type WorkloadSamplingConfig struct {
	// WorkloadCollectorConfig is the collector config (e.g. tail sampling) per container.
	WorkloadCollectorConfig []commonapi.ContainerCollectorConfig `json:"workloadCollectorConfig,omitempty"`
}

// Cache stores workload sampling config by key (namespace/kind/name).
type Cache struct {
	mu   sync.RWMutex
	data map[string]*WorkloadSamplingConfig
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{data: make(map[string]*WorkloadSamplingConfig)}
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
func (c *Cache) Get(key string) (*WorkloadSamplingConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cfg, ok := c.data[key]
	return cfg, ok
}

// Set stores the sampling config for the given workload key.
func (c *Cache) Set(key string, cfg *WorkloadSamplingConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cfg
}

// Delete removes the entry for the given workload key.
func (c *Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// All returns a copy of all keys (for tests/debugging). Keys are "namespace/kind/name".
func (c *Cache) AllKeys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

// ParseWorkloadKey splits "namespace/kind/name" into (namespace, kind, name).
// Returns an error if the key does not have exactly three parts.
func ParseWorkloadKey(key string) (namespace, kind, name string, err error) {
	parts := strings.SplitN(key, "/", 3)
	if len(parts) != 3 {
		return "", "", "", fmt.Errorf("invalid workload key %q: expected namespace/kind/name", key)
	}
	return parts[0], parts[1], parts[2], nil
}
