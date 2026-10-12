package odigosconfigk8sextension

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	commonapi "github.com/odigos-io/odigos/common/api"
)

func containerConfig(containerName string) *commonapi.ContainerCollectorConfig {
	return &commonapi.ContainerCollectorConfig{ContainerName: containerName}
}

// countingCallback records how many notifications it received.
type countingCallback struct {
	mu      sync.Mutex
	sets    int
	deletes int
}

func (c *countingCallback) OnSet(_ string, _ *commonapi.ContainerCollectorConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sets++
}

func (c *countingCallback) OnDeleteKey(_ string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deletes++
}

func (c *countingCallback) counts() (int, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.sets, c.deletes
}

func TestKeyPrefixFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"prod/Deployment/checkout/app", "prod/Deployment/checkout/"},
		{"prod/Deployment/checkout/", "prod/Deployment/checkout/"},
		{"app", ""},
		{"", ""},
		{"/app", "/"},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			require.Equal(t, tt.want, keyPrefixFromKey(tt.key))
		})
	}
}

func TestCacheSetAndGet(t *testing.T) {
	c := newCache()

	_, found := c.Get("prod/Deployment/checkout/app")
	require.False(t, found)

	c.Set("prod/Deployment/checkout/app", containerConfig("app"))

	cfg, found := c.Get("prod/Deployment/checkout/app")
	require.True(t, found)
	require.Equal(t, "app", cfg.ContainerName)

	c.Set("prod/Deployment/checkout/app", containerConfig("app-updated"))
	cfg, found = c.Get("prod/Deployment/checkout/app")
	require.True(t, found)
	require.Equal(t, "app-updated", cfg.ContainerName)
}

// The workload index is maintained as a side effect of Set/Delete. It must appear with the first
// container and disappear with the last, otherwise IsActiveSource answers for workloads that no
// longer have any configuration.
func TestCacheWorkloadIndexLifecycle(t *testing.T) {
	c := newCache()
	const prefix = "prod/Deployment/checkout/"

	require.False(t, c.isSourceWorkloadPrefix(prefix))
	require.Nil(t, c.getContainerKeysForWorkload(prefix))

	c.Set(prefix+"app", containerConfig("app"))
	c.Set(prefix+"sidecar", containerConfig("sidecar"))

	require.True(t, c.isSourceWorkloadPrefix(prefix))
	require.ElementsMatch(t, []string{prefix + "app", prefix + "sidecar"}, c.getContainerKeysForWorkload(prefix))

	c.Delete(prefix + "app")
	require.True(t, c.isSourceWorkloadPrefix(prefix))
	require.Equal(t, []string{prefix + "sidecar"}, c.getContainerKeysForWorkload(prefix))

	c.Delete(prefix + "sidecar")
	require.False(t, c.isSourceWorkloadPrefix(prefix))
	require.Nil(t, c.getContainerKeysForWorkload(prefix))
}

// A workload that belongs to a data stream stays indexed even with no container config, so data
// stream routing survives a Source whose containers are not individually configured.
func TestCacheWorkloadIndexOutlivesContainersWhileDataStreamsRemain(t *testing.T) {
	c := newCache()
	const prefix = "prod/Deployment/checkout/"

	c.Set(prefix+"app", containerConfig("app"))
	c.SetDataStreams(prefix, []string{"payments"})

	c.Delete(prefix + "app")

	require.True(t, c.isSourceWorkloadPrefix(prefix))
	require.Nil(t, c.getContainerKeysForWorkload(prefix))

	streams, found := c.GetDataStreams(prefix)
	require.True(t, found)
	require.Equal(t, []string{"payments"}, streams)

	c.removeWorkloadEntry(prefix)
	require.False(t, c.isSourceWorkloadPrefix(prefix))
	_, found = c.GetDataStreams(prefix)
	require.False(t, found)
}

func TestCacheDataStreams(t *testing.T) {
	c := newCache()
	const prefix = "prod/Deployment/checkout/"

	streams, found := c.GetDataStreams(prefix)
	require.False(t, found)
	require.Nil(t, streams)

	// SetDataStreams creates the workload entry for a workload with no container config.
	c.SetDataStreams(prefix, []string{"payments"})
	streams, found = c.GetDataStreams(prefix)
	require.True(t, found)
	require.Equal(t, []string{"payments"}, streams)

	c.SetDataStreams(prefix, nil)
	streams, found = c.GetDataStreams(prefix)
	require.True(t, found)
	require.Nil(t, streams)
}

// A key with no separator has no workload prefix, so it must not create an index entry keyed by
// the empty string that every prefix lookup would then have to step around.
func TestCacheIgnoresKeysWithoutAWorkloadPrefix(t *testing.T) {
	c := newCache()

	c.Set("malformed", containerConfig("app"))

	_, found := c.Get("malformed")
	require.True(t, found)
	require.False(t, c.isSourceWorkloadPrefix(""))

	c.Delete("malformed")
	_, found = c.Get("malformed")
	require.False(t, found)
}

func TestCacheNotifiesEveryRegisteredCallback(t *testing.T) {
	c := newCache()
	first, second := &countingCallback{}, &countingCallback{}
	c.addCallback(first)
	c.addCallback(second)

	c.Set("prod/Deployment/checkout/app", containerConfig("app"))
	c.Delete("prod/Deployment/checkout/app")

	for _, cb := range []*countingCallback{first, second} {
		sets, deletes := cb.counts()
		require.Equal(t, 1, sets)
		require.Equal(t, 1, deletes)
	}
}

func TestCacheRemoveCallback(t *testing.T) {
	c := newCache()
	removed, kept := &countingCallback{}, &countingCallback{}
	c.addCallback(removed)
	c.addCallback(kept)

	c.removeCallback(removed)
	c.Set("prod/Deployment/checkout/app", containerConfig("app"))

	sets, _ := removed.counts()
	require.Equal(t, 0, sets)
	sets, _ = kept.counts()
	require.Equal(t, 1, sets)

	// Removing a callback that was never registered must leave the remaining ones in place.
	c.removeCallback(&countingCallback{})
	c.Set("prod/Deployment/checkout/app", containerConfig("app"))
	sets, _ = kept.counts()
	require.Equal(t, 2, sets)
}

func TestCacheClear(t *testing.T) {
	c := newCache()
	cb := &countingCallback{}
	c.addCallback(cb)
	c.Set("prod/Deployment/checkout/app", containerConfig("app"))
	c.SetDataStreams("prod/Deployment/checkout/", []string{"payments"})

	c.clear()

	_, found := c.Get("prod/Deployment/checkout/app")
	require.False(t, found)
	require.False(t, c.isSourceWorkloadPrefix("prod/Deployment/checkout/"))

	c.Set("prod/Deployment/checkout/app", containerConfig("app"))
	sets, _ := cb.counts()
	require.Equal(t, 1, sets, "callbacks must be dropped by clear")
}

// Set and Delete invoke callbacks after releasing the lock, working from a snapshot taken while
// holding it. A processor registering as the informer delivers events must not race the slice.
func TestCacheConcurrentSetAndCallbackRegistration(t *testing.T) {
	c := newCache()
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			c.Set("prod/Deployment/checkout/app", containerConfig("app"))
			c.Delete("prod/Deployment/checkout/app")
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			cb := &countingCallback{}
			c.addCallback(cb)
			c.removeCallback(cb)
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			c.Get("prod/Deployment/checkout/app")
			c.isSourceWorkloadPrefix("prod/Deployment/checkout/")
			c.getContainerKeysForWorkload("prod/Deployment/checkout/")
		}
	}()

	wg.Wait()
}
