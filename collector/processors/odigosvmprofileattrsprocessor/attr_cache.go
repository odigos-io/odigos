package odigosvmprofileattrsprocessor

import (
	"sync"

	"github.com/odigos-io/odigos/common/unixfd"
)

// profileAttrCache maps process PID to packed resource attributes from the VM agent.
type profileAttrCache struct {
	mu    sync.RWMutex
	cache map[uint32]string
}

func newProfileAttrCache() *profileAttrCache {
	return &profileAttrCache{cache: make(map[uint32]string)}
}

func (c *profileAttrCache) get(pid uint32) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.cache[pid]
	return v, ok
}

func (c *profileAttrCache) applyEvent(line string) {
	ev, ok := unixfd.DecodeLogsAttrEvent(line)
	if !ok {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch ev.Type {
	case unixfd.LogsAttrRegister:
		c.cache[ev.PID] = ev.Attrs
	case unixfd.LogsAttrUnregister:
		delete(c.cache, ev.PID)
	}
}

func (c *profileAttrCache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}
