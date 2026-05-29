package odigostailsamplingprocessor

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigostailsamplingprocessor/category"
	commonapi "github.com/odigos-io/odigos/common/api"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/collector"
)

var (
	_ collector.WorkloadConfigCacheCallback = (*processorTailSamplingConfigCache)(nil)
	_ category.TailSamplingConfigProvider     = (*processorTailSamplingConfigCache)(nil)
)

// processorTailSamplingConfigCache caches tail sampling config per workload key (namespace/kind/name/container).
// It registers with odigos_config_extension for updates and resolves config on the hot path from the local cache.
type processorTailSamplingConfigCache struct {
	logger   *zap.Logger
	mu       sync.RWMutex
	data     map[string]*commonapisampling.TailSamplingSourceConfig
	provider collector.OdigosConfigExtension
}

func newProcessorTailSamplingConfigCache(logger *zap.Logger) *processorTailSamplingConfigCache {
	return &processorTailSamplingConfigCache{
		logger: logger,
		data:   make(map[string]*commonapisampling.TailSamplingSourceConfig),
	}
}

// Start resolves odigos_config_extension and registers for workload config updates.
func (c *processorTailSamplingConfigCache) Start(ctx context.Context, host component.Host, extID *component.ID) error {
	if extID == nil {
		return nil
	}
	extensions := host.GetExtensions()
	ext, ok := extensions[*extID]
	if !ok {
		return fmt.Errorf("odigos config extension %q not found or no instance implements OdigosConfigExtension", extID.String())
	}
	return c.attach(ctx, ext, extID.String())
}

func (c *processorTailSamplingConfigCache) attach(ctx context.Context, ext component.Component, extensionID string) error {
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		return fmt.Errorf("extension %q is not an OdigosConfigExtension (got %T)", extensionID, ext)
	}
	c.provider = odigosExt
	c.provider.RegisterWorkloadConfigCacheCallback(c)
	if !c.provider.WaitForCacheSync(ctx) {
		c.logger.Warn("odigos config extension cache sync did not complete; some traces may be missed on startup")
	}
	return nil
}

// Shutdown unregisters from the extension and clears local state.
func (c *processorTailSamplingConfigCache) Shutdown(context.Context) error {
	if c.provider != nil {
		c.provider.UnregisterWorkloadConfigCacheCallback(c)
		c.provider = nil
	}
	c.clear()
	return nil
}

// Attached reports whether the cache is connected to odigos_config_extension.
func (c *processorTailSamplingConfigCache) Attached() bool {
	return c.provider != nil
}

// OnSet implements collector.WorkloadConfigCacheCallback.
func (c *processorTailSamplingConfigCache) OnSet(key string, cfg *commonapi.ContainerCollectorConfig) {
	var tailSampling *commonapisampling.TailSamplingSourceConfig
	if cfg != nil && cfg.TailSampling != nil {
		tailSampling = cfg.TailSampling
	}
	c.set(key, tailSampling)
	c.logger.Debug("workload tail sampling config cache OnSet", zap.String("key", key))
}

// OnDeleteKey implements collector.WorkloadConfigCacheCallback.
func (c *processorTailSamplingConfigCache) OnDeleteKey(key string) {
	c.delete(key)
	c.logger.Debug("workload tail sampling config cache OnDeleteKey", zap.String("key", key))
}

// GetTailSamplingConfig implements category.TailSamplingConfigProvider.
func (c *processorTailSamplingConfigCache) GetTailSamplingConfig(resource pcommon.Resource) (*commonapisampling.TailSamplingSourceConfig, bool) {
	if c.provider == nil {
		return nil, false
	}
	key, err := c.provider.GetWorkloadCacheKey(resource)
	if err != nil {
		return nil, false
	}
	tailSampling, ok := c.get(key)
	if !ok || tailSampling == nil {
		return nil, false
	}
	return tailSampling, true
}

func (c *processorTailSamplingConfigCache) get(key string) (*commonapisampling.TailSamplingSourceConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cfg, ok := c.data[key]
	return cfg, ok
}

func (c *processorTailSamplingConfigCache) set(key string, cfg *commonapisampling.TailSamplingSourceConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cfg
}

func (c *processorTailSamplingConfigCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *processorTailSamplingConfigCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*commonapisampling.TailSamplingSourceConfig)
}
