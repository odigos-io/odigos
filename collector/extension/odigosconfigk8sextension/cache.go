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
// When Set or Delete is called, the cache invokes all registered callbacks
// so consumers stay in sync without the informer knowing about callbacks.
type cache struct {
	mu        sync.RWMutex
	data      map[string]*commonapi.ContainerCollectorConfig
	callbacks []collector.WorkloadConfigCacheCallback
}

// newCache creates a new empty cache.
func newCache() *cache {
	return &cache{data: make(map[string]*commonapi.ContainerCollectorConfig)}
}

// addCallback appends a callback invoked on Set/Delete. Called by the extension when
// a processor registers via RegisterWorkloadConfigCacheCallback. Supports multiple processors.
func (c *cache) addCallback(cb collector.WorkloadConfigCacheCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, cb)
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
func (c *cache) Get(key string) (*commonapi.ContainerCollectorConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, found := c.data[key]
	return val, found
}

// Set stores the required config for the given workload key, then invokes all registered callbacks.
// We snapshot the callback list under the lock (so we never read c.callbacks after unlock, avoiding
// a race with addCallback), then unlock and invoke each callback.
func (c *cache) Set(key string, cfg *commonapi.ContainerCollectorConfig) {
	c.mu.Lock()
	c.data[key] = cfg
	n := len(c.callbacks)
	currentCallBacks := make([]collector.WorkloadConfigCacheCallback, n)
	copy(currentCallBacks, c.callbacks)
	c.mu.Unlock()
	for _, cb := range currentCallBacks {
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

// Delete removes the entry for the given key, then invokes all registered callbacks.
// We snapshot the callback list under the lock (so we never read c.callbacks after unlock, avoiding
// a race with addCallback), then unlock and invoke each callback.
func (c *cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	n := len(c.callbacks)
	currentCallBacks := make([]collector.WorkloadConfigCacheCallback, n)
	copy(currentCallBacks, c.callbacks)
	c.mu.Unlock()
	for _, cb := range currentCallBacks {
		cb.OnDeleteKey(key)
	}
}
