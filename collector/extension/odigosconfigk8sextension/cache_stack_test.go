// Cache stack-pointer tests.
//
// The extension cache stores *ContainerCollectorConfig. Callers must pass a pointer
// to heap-allocated memory; passing &localVar (stack) causes a dangling pointer
// after the function returns. Main's handleInstrumentationConfig uses the buggy
// pattern: o.cache.Set(cacheKey, &c) with local c. To verify: run
//   go build -gcflags="-m" ./...
// and check informer.go; before the fix the config in the loop does not escape.
package odigosconfigk8sextension

import (
	"runtime"
	"sync"
	"testing"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// setWithStackPointer mimics the buggy pattern in handleInstrumentationConfig on main:
//   var c commonapi.ContainerCollectorConfig
//   ...
//   o.cache.Set(cacheKey, &c)
// When this function returns, c lives on the stack and is invalid; the cache holds
// a dangling pointer. Do not use this pattern. Use setWithHeapAllocated instead.
func setWithStackPointer(c *cache, key string, containerName string) {
	var cfg commonapi.ContainerCollectorConfig
	cfg.ContainerName = containerName
	c.Set(key, &cfg)
}

// setWithHeapAllocated is the correct pattern: allocate config on the heap so the
// cache can store a pointer that remains valid after the function returns. The
// informer (handleInstrumentationConfig) should use this pattern.
func setWithHeapAllocated(c *cache, key string, containerName string) {
	cfg := new(commonapi.ContainerCollectorConfig)
	cfg.ContainerName = containerName
	c.Set(key, cfg)
}

// useStack uses stack space to encourage reuse of a previous frame (for tests that
// try to trigger use-after-return).
func useStack(depth int) {
	if depth <= 0 {
		return
	}
	var buf [256]byte
	_ = buf
	useStack(depth - 1)
}

// TestCacheSetWithHeapAllocatedConfig verifies that when we Set a heap-allocated
// config, Get returns the correct value. This is the pattern the informer must use.
// If the informer instead uses Set(key, &c) with a local c (stack), the cache holds
// a dangling pointer and behavior is undefined. This test locks in the correct usage.
func TestCacheSetWithHeapAllocatedConfig(t *testing.T) {
	c := newCache()
	key := "default/deployment/myapp/nginx"
	expectedName := "nginx"

	setWithHeapAllocated(c, key, expectedName)

	cfg, found := c.Get(key)
	if !found {
		t.Fatal("cache.Get: entry not found")
	}
	if cfg == nil {
		t.Fatal("cache.Get: got nil config")
	}
	if cfg.ContainerName != expectedName {
		t.Errorf("got ContainerName %q, want %q", cfg.ContainerName, expectedName)
	}

	runtime.GC()
	if cfg.ContainerName != expectedName {
		t.Errorf("after GC: got ContainerName %q, want %q", cfg.ContainerName, expectedName)
	}
}

// TestCacheSetWithStackPointerIsUnsafe runs the buggy pattern (set with &local)
// then stresses the runtime; if the stack is reused we may see wrong data or panic.
// This test is best-effort: it may pass even with the bug because stack reuse is
// not guaranteed. Use escape analysis to verify: go build -gcflags="-m" ./... and
// ensure the config in the informer's loop escapes to heap after the fix.
func TestCacheSetWithStackPointerIsUnsafe(t *testing.T) {
	const key = "default/deployment/myapp/nginx"
	const expectedName = "nginx"

	for iter := 0; iter < 30; iter++ {
		c := newCache()

		done := make(chan struct{})
		go func() {
			setWithStackPointer(c, key, expectedName)
			close(done)
		}()
		<-done

		var wg sync.WaitGroup
		for g := 0; g < 20; g++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				useStack(24)
			}()
		}
		wg.Wait()
		runtime.GC()

		cfg, found := c.Get(key)
		if !found {
			t.Fatalf("iter %d: entry not found", iter)
		}
		if cfg != nil && cfg.ContainerName != expectedName {
			t.Fatalf("iter %d: got ContainerName %q, want %q (dangling pointer)", iter, cfg.ContainerName, expectedName)
		}
	}
}
