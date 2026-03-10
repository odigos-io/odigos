package odigosurltemplateprocessor

import "sync"

// processorURLTemplateParsedRulesCache caches parsed rules per workload key (namespace/kind/name/container).
// Updated via extension callback on cache add/update/delete; hot path only does a read.
type processorURLTemplateParsedRulesCache struct {
	mu   sync.RWMutex
	data map[string]parsedWorkloadEntry
}

func newProcessorURLTemplateParsedRulesCache() *processorURLTemplateParsedRulesCache {
	return &processorURLTemplateParsedRulesCache{data: make(map[string]parsedWorkloadEntry)}
}

func (c *processorURLTemplateParsedRulesCache) get(key string) (parsedWorkloadEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	return e, ok
}

func (c *processorURLTemplateParsedRulesCache) set(key string, e parsedWorkloadEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = e
}

// delete removes the entry for the given full cache key.
// The extension always notifies with full keys (namespace/kind/name/containerName).
func (c *processorURLTemplateParsedRulesCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *processorURLTemplateParsedRulesCache) keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	keys := make([]string, 0, len(c.data))
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}
