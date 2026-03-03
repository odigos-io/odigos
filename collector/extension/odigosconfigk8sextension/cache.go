package odigosconfigk8sextension

import (
	"strings"
	"sync"

	commonapi "github.com/odigos-io/odigos/common/api"
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
type cache struct {
	mu   sync.RWMutex
	data map[string]*commonapi.ContainerCollectorConfig
}

// newCache creates a new empty cache.
func newCache() *cache {
	return &cache{data: make(map[string]*commonapi.ContainerCollectorConfig)}
}

// Get returns the WorkloadSamplingConfig for the given workload key, and true if found.
func (c *cache) Get(key string) (*commonapi.ContainerCollectorConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	val, found := c.data[key]
	return val, found
}

// Set stores the sampling config for the given workload key.
func (c *cache) Set(key string, cfg *commonapi.ContainerCollectorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cfg
}

// DeleteWorkload removes the entry for the given workload key.
func (c *cache) DeleteWorkload(workloadKey workloadKey) {
	c.mu.Lock()
	defer c.mu.Unlock()

	keyPrefix := k8sSourceKey(workloadKey.Namespace, workloadKey.Kind, workloadKey.Name, "")

	// cache key is in container level, this function delete on the workload level.
	// iterate over the data and delete each entry where the key starts with the given key.
	// since this is very rare, and cache size is in the hundreds maximum, we can afford to iterate here.
	for k := range c.data {
		if strings.HasPrefix(k, keyPrefix) {
			delete(c.data, k)
		}
	}
}
