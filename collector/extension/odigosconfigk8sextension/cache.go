package odigosconfigk8sextension

import (
	"sync"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

// workloadKey identifies a workload by namespace, kind, and name.
// Kind is the workload kind (e.g. Deployment, StatefulSet).
// Fields may be empty depending on context.
type workloadKey struct {
	Namespace string
	Kind      string
	Name      string
}

// cache stores workload sampling config by WorkloadKey.
// When Set or Delete is called, the cache invokes the registered callback (if any)
// so consumers stay in sync without the informer knowing about callbacks.
type cache struct {
	mu   sync.RWMutex
	data map[string]*commonapi.ContainerCollectorConfig

	cb   collector.WorkloadConfigCacheCallback
	cbMu sync.RWMutex
}

// newCache creates a new empty cache.
func newCache() *cache {
	return &cache{data: make(map[string]*commonapi.ContainerCollectorConfig)}
}

// setCallback sets the callback invoked on Set/Delete. Called by the extension when
// a processor registers via RegisterWorkloadConfigCacheCallback.
func (c *cache) setCallback(cb collector.WorkloadConfigCacheCallback) {
	c.cbMu.Lock()
	defer c.cbMu.Unlock()
	c.cb = cb
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
func (c *cache) Get(key string) (*commonapi.ContainerCollectorConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, found := c.data[key]
	return val, found
}

// Set stores the sampling config for the given workload key, then invokes the callback if set.
func (c *cache) Set(key string, cfg *commonapi.ContainerCollectorConfig) {
	c.mu.Lock()
	c.data[key] = cfg
	c.mu.Unlock()
	c.cbMu.RLock()
	cb := c.cb
	c.cbMu.RUnlock()
	if cb != nil {
		cb.OnSet(key, cfg)
	}
}

// Range calls f for each key and config in the cache. Caller must not modify the cache from f.
func (c *cache) Range(f func(key string, cfg *commonapi.ContainerCollectorConfig)) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	for k, v := range c.data {
		f(k, v)
	}
}

// Delete removes the entry for the given key, then invokes the callback if set.
func (c *cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	c.mu.Unlock()
	c.cbMu.RLock()
	cb := c.cb
	c.cbMu.RUnlock()
	if cb != nil {
		cb.OnDeleteKey(key)
	}
}
