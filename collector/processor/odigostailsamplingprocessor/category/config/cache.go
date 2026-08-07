package config

import (
	"context"
	"fmt"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

var (
	_ collector.WorkloadConfigCacheCallback = (*ConfigCache)(nil)
	_ TailSamplingConfigProvider            = (*ConfigCache)(nil)
)

// processorTailSamplingConfigCache caches tail sampling config per workload key (namespace/kind/name/container).
// It registers with odigos_config_extension for updates and resolves config on the hot path from the local ConfigCache.
type ConfigCache struct {
	logger   *zap.Logger
	mu       sync.RWMutex
	data     map[string]*ComputedWorkloadConfig
	provider collector.OdigosConfigExtension

	dryRun bool
}

func NewConfigCache(logger *zap.Logger, dryRun bool) *ConfigCache {
	return &ConfigCache{
		logger: logger,
		data:   make(map[string]*ComputedWorkloadConfig),
		dryRun: dryRun,
	}
}

// Start resolves odigos_config_extension and registers for workload config updates.
func (c *ConfigCache) Start(ctx context.Context, host component.Host, extID *component.ID) error {
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

func (c *ConfigCache) attach(ctx context.Context, ext component.Component, extensionID string) error {
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
func (c *ConfigCache) Shutdown(context.Context) error {
	if c.provider != nil {
		c.provider.UnregisterWorkloadConfigCacheCallback(c)
		c.provider = nil
	}
	c.clear()
	return nil
}

// Attached reports whether the cache is connected to odigos_config_extension.
func (c *ConfigCache) Attached() bool {
	return c.provider != nil
}

// OnSet implements collector.WorkloadConfigCacheCallback.
func (c *ConfigCache) OnSet(key string, cfg *commonapi.ContainerCollectorConfig) {

	if cfg == nil || cfg.TailSampling == nil {
		c.delete(key)
		return
	}

	computed := precomputeWorkloadConfig(cfg.TailSampling, c.dryRun)
	c.set(key, computed)

	c.logger.Debug("workload tail sampling config cache OnSet", zap.String("key", key))
}

// OnDeleteKey implements collector.WorkloadConfigCacheCallback.
func (c *ConfigCache) OnDeleteKey(key string) {
	c.delete(key)
	c.logger.Debug("workload tail sampling config cache OnDeleteKey", zap.String("key", key))
}

// GetTailSamplingConfig implements category.TailSamplingConfigProvider.
func (c *ConfigCache) GetTailSamplingConfig(resource pcommon.Resource) (*ComputedWorkloadConfig, bool) {
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

func (c *ConfigCache) get(key string) (*ComputedWorkloadConfig, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cfg, ok := c.data[key]
	return cfg, ok
}

func (c *ConfigCache) set(key string, cfg *ComputedWorkloadConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cfg
}

func (c *ConfigCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *ConfigCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*ComputedWorkloadConfig)
}
