package odigospiimaskingprocessor

import "sync"

// processorPiiMaskingCache caches compiled PII masking rules per workload key.
// Updated via extension callback on cache add/update/delete; hot path only does a read.
type processorPiiMaskingCache struct {
	mu   sync.RWMutex
	data map[string]compiledPiiMaskingConfig
}

func newProcessorPiiMaskingCache() *processorPiiMaskingCache {
	return &processorPiiMaskingCache{data: make(map[string]compiledPiiMaskingConfig)}
}

func (c *processorPiiMaskingCache) get(key string) (compiledPiiMaskingConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.data[key]
	return e, ok
}

func (c *processorPiiMaskingCache) set(key string, e compiledPiiMaskingConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = e
}

func (c *processorPiiMaskingCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *processorPiiMaskingCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]compiledPiiMaskingConfig)
}
