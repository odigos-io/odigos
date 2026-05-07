package odigosconfigk8sextension

import (
	"strings"
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

// workloadEntry holds per-workload state: the set of container cache keys and the data stream names extracted from IC labels.
type workloadEntry struct {
	containerKeys map[string]struct{}
	dataStreams   []string
}

// keyPrefixFromKey returns the workload prefix for a full cache key (e.g. "ns/kind/name/container" -> "ns/kind/name/").
func keyPrefixFromKey(key string) string {
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return ""
	}
	return key[:i+1]
}

// cache stores workload sampling config by WorkloadKey.
// When Set or Delete is called, the cache invokes all registered callbacks
// so consumers stay in sync without the informer knowing about callbacks.
// workloadKeysIndex maps workload key prefix (e.g. "ns/kind/name/") to the workload entry.
type cache struct {
	mu                sync.RWMutex
	data              map[string]*commonapi.ContainerCollectorConfig
	callbacks         []collector.WorkloadConfigCacheCallback
	workloadKeysIndex map[string]*workloadEntry
}

// newCache creates a new empty cache.
func newCache() *cache {
	return &cache{
		data:              make(map[string]*commonapi.ContainerCollectorConfig),
		workloadKeysIndex: make(map[string]*workloadEntry),
	}
}

// addCallback appends a callback invoked on Set/Delete. Called by the extension when
// a processor registers via RegisterWorkloadConfigCacheCallback. Supports multiple processors.
func (c *cache) addCallback(cb collector.WorkloadConfigCacheCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, cb)
}

// removeCallback removes the callback so it is no longer invoked on Set/Delete.
// Processors call this in Shutdown so the extension stops holding a reference and the processor can release its cache.
func (c *cache) removeCallback(cb collector.WorkloadConfigCacheCallback) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, candidate := range c.callbacks {
		if candidate == cb {
			c.callbacks = append(c.callbacks[:i], c.callbacks[i+1:]...)
			return
		}
	}
}

// clear removes all cache data and callbacks. Used in extension Shutdown to release memory.
func (c *cache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*commonapi.ContainerCollectorConfig)
	c.workloadKeysIndex = make(map[string]*workloadEntry)
	c.callbacks = nil
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
func (c *cache) Get(key string) (*commonapi.ContainerCollectorConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, found := c.data[key]
	return val, found
}

// Set stores the required config for the given workload key, updates the workload keys index, then invokes all registered callbacks.
// We snapshot the callback list under the lock (so we never read c.callbacks after unlock, avoiding
// a race with addCallback), then unlock and invoke each callback.
func (c *cache) Set(key string, cfg *commonapi.ContainerCollectorConfig) {
	c.mu.Lock()
	c.data[key] = cfg
	workloadKey := keyPrefixFromKey(key)
	if workloadKey != "" {
		entry := c.workloadKeysIndex[workloadKey]
		if entry == nil {
			entry = &workloadEntry{containerKeys: make(map[string]struct{})}
			c.workloadKeysIndex[workloadKey] = entry
		}
		entry.containerKeys[key] = struct{}{}
	}
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

// Delete removes the entry for the given key, updates the workload keys index, then invokes all registered callbacks.
// We snapshot the callback list under the lock (so we never read c.callbacks after unlock, avoiding
// a race with addCallback), then unlock and invoke each callback.
func (c *cache) Delete(key string) {
	c.mu.Lock()
	delete(c.data, key)
	workloadKey := keyPrefixFromKey(key)
	if workloadKey != "" {
		if entry := c.workloadKeysIndex[workloadKey]; entry != nil {
			delete(entry.containerKeys, key)
			if len(entry.containerKeys) == 0 && len(entry.dataStreams) == 0 {
				delete(c.workloadKeysIndex, workloadKey)
			}
		}
	}
	n := len(c.callbacks)
	currentCallBacks := make([]collector.WorkloadConfigCacheCallback, n)
	copy(currentCallBacks, c.callbacks)
	c.mu.Unlock()
	for _, cb := range currentCallBacks {
		cb.OnDeleteKey(key)
	}
}

// getContainerKeysForWorkload returns a copy of the full cache keys for the given workload key. Caller must not modify the result.
func (c *cache) getContainerKeysForWorkload(workloadKey string) []string {
	c.mu.RLock()
	entry := c.workloadKeysIndex[workloadKey]
	if entry == nil || len(entry.containerKeys) == 0 {
		c.mu.RUnlock()
		return nil
	}
	out := make([]string, 0, len(entry.containerKeys))
	for k := range entry.containerKeys {
		out = append(out, k)
	}
	c.mu.RUnlock()
	return out
}

// SetDataStreams stores the data stream names for the given workload key prefix.
func (c *cache) SetDataStreams(workloadKey string, streams []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry := c.workloadKeysIndex[workloadKey]
	if entry == nil {
		entry = &workloadEntry{containerKeys: make(map[string]struct{})}
		c.workloadKeysIndex[workloadKey] = entry
	}
	entry.dataStreams = streams
}

// GetDataStreams returns the data stream names for the given workload key prefix.
func (c *cache) GetDataStreams(workloadKey string) ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry := c.workloadKeysIndex[workloadKey]
	if entry == nil {
		return nil, false
	}
	return entry.dataStreams, true
}
