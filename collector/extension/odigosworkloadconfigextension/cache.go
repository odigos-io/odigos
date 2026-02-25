package odigosworkloadconfigextension

import (
	"sync"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// WorkloadKey identifies a workload by namespace, kind, and name.
// Kind is the workload kind (e.g. Deployment, StatefulSet).
// Fields may be empty depending on context.
type WorkloadKey struct {
	Namespace string
	Kind      string
	Name      string
}

// WorkloadConfig holds the sampling configuration for a single workload,
// derived from an InstrumentationConfig. Uses api mirror types for head sampling and collector config.
type WorkloadConfig struct {
	// WorkloadCollectorConfig is the collector config (e.g. tail sampling) per container.
	WorkloadCollectorConfig []commonapi.ContainerCollectorConfig `json:"workloadCollectorConfig,omitempty"`
}

// Cache stores workload sampling config by WorkloadKey.
type Cache struct {
	mu   sync.RWMutex
	data map[WorkloadKey]*WorkloadConfig
}

// NewCache creates a new empty cache.
func NewCache() *Cache {
	return &Cache{data: make(map[WorkloadKey]*WorkloadConfig)}
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
// If the exact key is not found and key has a non-empty Kind, Get also tries a key with
// empty Kind so that lookups from resource attributes (with Kind) match entries stored
// from InstrumentationConfig metadata (namespace and name only).
func (c *Cache) Get(key WorkloadKey) (*WorkloadConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if cfg, ok := c.data[key]; ok {
		return cfg, true
	}
	if key.Kind != "" {
		fallback := WorkloadKey{Namespace: key.Namespace, Name: key.Name}
		if cfg, ok := c.data[fallback]; ok {
			return cfg, true
		}
	}
	return nil, false
}

// Set stores the sampling config for the given workload key.
func (c *Cache) Set(key WorkloadKey, cfg *WorkloadConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cfg
}

// Delete removes the entry for the given workload key.
func (c *Cache) Delete(key WorkloadKey) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

// AllKeys returns a copy of all keys (for tests/debugging).
func (c *Cache) AllKeys() []WorkloadKey {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]WorkloadKey, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}
