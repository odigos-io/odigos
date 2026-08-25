package config

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
)

const cacheTestWorkloadKeyAttr = "odigos.test.workload.key"

// cacheTestExtension resolves the workload cache key from a resource attribute, and records the
// callback registration so the processor's lifecycle contract with the extension can be asserted.
type cacheTestExtension struct {
	registered   []collector.WorkloadConfigCacheCallback
	unregistered []collector.WorkloadConfigCacheCallback
	synced       bool
	keyErr       error
}

func (e *cacheTestExtension) Start(context.Context, component.Host) error { return nil }
func (e *cacheTestExtension) Shutdown(context.Context) error              { return nil }

func (e *cacheTestExtension) GetFromResource(pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	return nil, false
}

func (e *cacheTestExtension) IsActiveSource(pcommon.Resource) bool { return true }

func (e *cacheTestExtension) GetWorkloadCacheKey(res pcommon.Resource) (string, error) {
	key, found := res.Attributes().Get(cacheTestWorkloadKeyAttr)
	if !found {
		return "", errors.New("no workload key on the resource")
	}
	// a real extension can fail after having built a partial key; the caller must not use it.
	return key.Str(), e.keyErr
}

func (e *cacheTestExtension) GetWorkloadIdentityFromResource(res pcommon.Resource) (string, pcommon.Map, error) {
	key, err := e.GetWorkloadCacheKey(res)
	return key, pcommon.NewMap(), err
}

func (e *cacheTestExtension) RegisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	e.registered = append(e.registered, cb)
}

func (e *cacheTestExtension) UnregisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	e.unregistered = append(e.unregistered, cb)
}

func (e *cacheTestExtension) WaitForCacheSync(context.Context) bool { return e.synced }

func (e *cacheTestExtension) GetDataStreamsForWorkload(pcommon.Resource) ([]string, bool) {
	return nil, false
}

// notAnOdigosExtension stands in for a component whose id is configured as the odigos config
// extension but which does not implement the interface.
type notAnOdigosExtension struct{}

func (notAnOdigosExtension) Start(context.Context, component.Host) error { return nil }
func (notAnOdigosExtension) Shutdown(context.Context) error              { return nil }

type cacheTestHost struct {
	extensions map[component.ID]component.Component
}

func (h cacheTestHost) GetExtensions() map[component.ID]component.Component { return h.extensions }

func cacheTestResource(key string) pcommon.Resource {
	res := pcommon.NewResource()
	res.Attributes().PutStr(cacheTestWorkloadKeyAttr, key)
	return res
}

func startedCache(t *testing.T, ext *cacheTestExtension) (*ConfigCache, component.ID) {
	t.Helper()
	extID := component.MustNewID("odigosconfigk8s")
	cache := NewConfigCache(zap.NewNop(), false)
	require.NoError(t, cache.Start(context.Background(), cacheTestHost{
		extensions: map[component.ID]component.Component{extID: ext},
	}, &extID))
	return cache, extID
}

func TestConfigCacheStart(t *testing.T) {
	// Without a configured extension there is nothing to attach to. Attached() must stay false so
	// the processor forwards every trace instead of sampling with an empty rule set.
	t.Run("no extension configured leaves the cache detached", func(t *testing.T) {
		cache := NewConfigCache(zap.NewNop(), false)
		require.NoError(t, cache.Start(context.Background(), cacheTestHost{}, nil))
		assert.False(t, cache.Attached())
	})

	t.Run("configured extension missing from the host is an error", func(t *testing.T) {
		extID := component.MustNewID("odigosconfigk8s")
		cache := NewConfigCache(zap.NewNop(), false)
		err := cache.Start(context.Background(), cacheTestHost{}, &extID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
		assert.False(t, cache.Attached())
	})

	t.Run("component that is not an odigos config extension is an error", func(t *testing.T) {
		extID := component.MustNewID("odigosconfigk8s")
		cache := NewConfigCache(zap.NewNop(), false)
		err := cache.Start(context.Background(), cacheTestHost{
			extensions: map[component.ID]component.Component{extID: notAnOdigosExtension{}},
		}, &extID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not an OdigosConfigExtension")
		assert.False(t, cache.Attached())
	})

	t.Run("attaching registers the cache for config updates", func(t *testing.T) {
		ext := &cacheTestExtension{synced: true}
		cache, _ := startedCache(t, ext)

		assert.True(t, cache.Attached())
		require.Len(t, ext.registered, 1)
		assert.Same(t, cache, ext.registered[0])
	})

	// A cache that never synced must still attach: the extension keeps pushing updates afterwards,
	// and refusing to start would take the whole collector down.
	t.Run("failed cache sync still attaches", func(t *testing.T) {
		ext := &cacheTestExtension{synced: false}
		cache, _ := startedCache(t, ext)
		assert.True(t, cache.Attached())
	})
}

func TestConfigCacheShutdown(t *testing.T) {
	ext := &cacheTestExtension{synced: true}
	cache, _ := startedCache(t, ext)
	cache.OnSet("default/deployment/checkout/app", &commonapi.ContainerCollectorConfig{
		TailSampling: &commonapisampling.TailSamplingSourceConfig{
			CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-1"}},
		},
	})

	require.NoError(t, cache.Shutdown(context.Background()))

	assert.False(t, cache.Attached())
	require.Len(t, ext.unregistered, 1)
	assert.Same(t, cache, ext.unregistered[0])

	_, found := cache.GetTailSamplingConfig(cacheTestResource("default/deployment/checkout/app"))
	assert.False(t, found, "shutdown must drop the cached config")

	// Shutdown runs again during collector teardown paths; it must not unregister twice or panic.
	require.NoError(t, cache.Shutdown(context.Background()))
	assert.Len(t, ext.unregistered, 1)
}

// A collector config reload stops and starts the processor. The rules of the previous run must not
// survive it, or a workload whose sampling config was removed in the meantime keeps being sampled.
func TestConfigCacheDoesNotCarryConfigAcrossARestart(t *testing.T) {
	const workloadKey = "default/deployment/checkout/app"

	extID := component.MustNewID("odigosconfigk8s")
	host := cacheTestHost{extensions: map[component.ID]component.Component{
		extID: &cacheTestExtension{synced: true},
	}}

	cache := NewConfigCache(zap.NewNop(), false)
	require.NoError(t, cache.Start(context.Background(), host, &extID))
	cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{
		TailSampling: &commonapisampling.TailSamplingSourceConfig{
			CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-1"}},
		},
	})
	require.NoError(t, cache.Shutdown(context.Background()))

	require.NoError(t, cache.Start(context.Background(), host, &extID))
	t.Cleanup(func() { require.NoError(t, cache.Shutdown(context.Background())) })

	_, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
	assert.False(t, found)
}

func TestConfigCacheGetTailSamplingConfig(t *testing.T) {
	const workloadKey = "default/deployment/checkout/app"

	sourceConfig := &commonapisampling.TailSamplingSourceConfig{
		CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-1", PercentageAtMost: 10}},
	}

	t.Run("config set for a workload is resolved from its resource", func(t *testing.T) {
		cache, _ := startedCache(t, &cacheTestExtension{synced: true})
		cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{TailSampling: sourceConfig})

		got, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
		require.True(t, found)
		require.Len(t, got.CostReductionRules, 1)
		assert.Equal(t, "cost-1", got.CostReductionRules[0].RuleId)
	})

	t.Run("another workload's resource resolves nothing", func(t *testing.T) {
		cache, _ := startedCache(t, &cacheTestExtension{synced: true})
		cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{TailSampling: sourceConfig})

		_, found := cache.GetTailSamplingConfig(cacheTestResource("default/deployment/cart/app"))
		assert.False(t, found)
	})

	// The key is only trustworthy when the extension did not fail. Using it anyway would apply one
	// workload's sampling rules to whatever workload the failed lookup happened to name.
	t.Run("a resource the extension cannot key resolves nothing", func(t *testing.T) {
		cache, _ := startedCache(t, &cacheTestExtension{synced: true, keyErr: errors.New("boom")})
		cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{TailSampling: sourceConfig})

		_, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
		assert.False(t, found)
	})

	t.Run("detached cache resolves nothing", func(t *testing.T) {
		cache := NewConfigCache(zap.NewNop(), false)
		cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{TailSampling: sourceConfig})

		_, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
		assert.False(t, found)
	})
}

// When a user removes the sampling rules from a source, the extension pushes a config without a
// tail sampling section rather than deleting the key. Treating that as "keep the previous rules"
// would keep dropping the workload's traces forever.
func TestConfigCacheOnSetWithoutTailSamplingRemovesTheConfig(t *testing.T) {
	const workloadKey = "default/deployment/checkout/app"

	tests := []struct {
		name string
		cfg  *commonapi.ContainerCollectorConfig
	}{
		{name: "nil container config", cfg: nil},
		{name: "container config without a tail sampling section", cfg: &commonapi.ContainerCollectorConfig{ContainerName: "app"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache, _ := startedCache(t, &cacheTestExtension{synced: true})
			cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{
				TailSampling: &commonapisampling.TailSamplingSourceConfig{
					CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-1"}},
				},
			})
			_, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
			require.True(t, found)

			cache.OnSet(workloadKey, tt.cfg)

			_, found = cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
			assert.False(t, found)
		})
	}
}

func TestConfigCacheOnDeleteKey(t *testing.T) {
	const workloadKey = "default/deployment/checkout/app"
	const otherKey = "default/deployment/cart/app"

	cache, _ := startedCache(t, &cacheTestExtension{synced: true})
	for _, key := range []string{workloadKey, otherKey} {
		cache.OnSet(key, &commonapi.ContainerCollectorConfig{
			TailSampling: &commonapisampling.TailSamplingSourceConfig{
				CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-" + key}},
			},
		})
	}

	cache.OnDeleteKey(workloadKey)

	_, found := cache.GetTailSamplingConfig(cacheTestResource(workloadKey))
	assert.False(t, found)
	_, found = cache.GetTailSamplingConfig(cacheTestResource(otherKey))
	assert.True(t, found, "deleting one workload must not affect another")

	// The extension may replay a delete for a key it already removed.
	cache.OnDeleteKey(workloadKey)
}

// The extension pushes config updates from its informer goroutine while traces are being sampled on
// the pipeline goroutines, so the cache has to be safe for concurrent use.
func TestConfigCacheConcurrentUpdatesAndReads(t *testing.T) {
	const workloadKey = "default/deployment/checkout/app"

	cache, _ := startedCache(t, &cacheTestExtension{synced: true})
	resource := cacheTestResource(workloadKey)

	var wg sync.WaitGroup
	for writer := 0; writer < 4; writer++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				cache.OnSet(workloadKey, &commonapi.ContainerCollectorConfig{
					TailSampling: &commonapisampling.TailSamplingSourceConfig{
						CostReductionRules: []commonapisampling.CostReductionRule{{Id: "cost-1", PercentageAtMost: float64(i % 100)}},
					},
				})
				cache.OnDeleteKey(workloadKey)
			}
		}()
	}
	for reader := 0; reader < 4; reader++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				if cfg, found := cache.GetTailSamplingConfig(resource); found {
					_ = cfg.CostReductionRules
				}
			}
		}()
	}
	wg.Wait()
}
