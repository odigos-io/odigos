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

// applyEvent decodes a single attribute-stream line and applies the register / unregister to the cache.
func (c *profileAttrCache) applyEvent(line string) {
	ev, ok := unixfd.DecodeAttrEvent(line)
	if !ok {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	switch ev.Type {
	case unixfd.AttrEventRegister:
		c.cache[ev.PID] = ev.Attrs
	case unixfd.AttrEventUnregister:
		delete(c.cache, ev.PID)
	}
}

func (c *profileAttrCache) size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.cache)
}

// reset clears the cache. Called by the unixfd client on every new session so the snapshot
// replay rebuilds state from scratch — without this, entries that were Unregister'd during a
// disconnect window would linger because the snapshot contains only R events, never U.
func (c *profileAttrCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[uint32]string)
}
