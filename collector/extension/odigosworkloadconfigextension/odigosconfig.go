package odigosworkloadconfigextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

// OdigosWorkloadConfig is an extension that runs a dynamic informer for InstrumentationConfigs
// and maintains a cache of workload sampling config keyed by WorkloadKey (namespace, kind, name).
type OdigosWorkloadConfig struct {
	cache  *Cache
	logger *zap.Logger
	cancel context.CancelFunc
}

// NewOdigosConfig creates a new OdigosConfig extension.
func NewOdigosConfig(settings component.TelemetrySettings) (*OdigosWorkloadConfig, error) {
	return &OdigosWorkloadConfig{
		cache:  NewCache(),
		logger: settings.Logger,
	}, nil
}

// Start starts the dynamic informer for InstrumentationConfigs. The informer
// fills the cache with workload sampling configs keyed by WorkloadKey.
func (o *OdigosWorkloadConfig) Start(ctx context.Context, _ component.Host) error {
	ctx, o.cancel = context.WithCancel(ctx)
	return o.startInformer(ctx)
}

// Shutdown stops the informer and clears the cache.
func (o *OdigosWorkloadConfig) Shutdown(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	return nil
}

// GetWorkloadSamplingConfig returns the sampling config for the given workload key, or (nil, false) if not found.
func (o *OdigosWorkloadConfig) GetWorkloadSamplingConfig(key WorkloadKey) (*WorkloadSamplingConfig, bool) {
	return o.cache.Get(key)
}

// Cache returns the underlying cache for advanced use (e.g. iteration).
// Do not modify the cache directly; use GetWorkloadSamplingConfig for reads.
func (o *OdigosWorkloadConfig) Cache() *Cache {
	return o.cache
}
