package odigosurltemplateprocessor

import (
	"strings"
	"sync"
)

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

// delete removes the entry for key. If key ends with "/", treats it as a workload prefix
// and removes all entries whose key has that prefix (so extension can notify once per workload delete).
func (c *processorURLTemplateParsedRulesCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if strings.HasSuffix(key, "/") {
		for k := range c.data {
			if strings.HasPrefix(k, key) {
				delete(c.data, k)
			}
		}
		return
	}
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
