package odigosworkloadconfigextension

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"k8s.io/client-go/dynamic/dynamicinformer"
)

// OdigosWorkloadConfig is an extension that runs a dynamic informer for InstrumentationConfigs
// and maintains a cache of workload sampling config keyed by WorkloadKey (namespace, kind, name).
type OdigosWorkloadConfig struct {
	cache           *Cache
	logger          *zap.Logger
	cancel          context.CancelFunc
	informerFactory dynamicinformer.DynamicSharedInformerFactory // set when in-cluster; nil otherwise
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
func (o *OdigosWorkloadConfig) GetWorkloadSamplingConfig(key WorkloadKey) (*WorkloadConfig, bool) {
	return o.cache.Get(key)
}

// GetWorkloadUrlTemplatizationRules returns the URL templatization rules for the workload identified
// by the given resource attributes and whether the workload is opted in to templatization.
// optedIn is true if the workload has at least one container with UrlTemplatization configured.
// When optedIn is false, the processor should leave the span untouched.
// When optedIn is true and rules is nil/empty, heuristic templatization should be applied.
// When optedIn is true and rules is non-empty, explicit rules (plus heuristic fallback) should be applied.
func (o *OdigosWorkloadConfig) GetWorkloadUrlTemplatizationRules(attrs pcommon.Map) (rules []string, optedIn bool) {
	key := WorkloadKeyFromResourceAttributes(attrs)
	cfg, found := o.cache.Get(key)
	if !found {
		return nil, false
	}
	// Collect templatization rules from all containers; optedIn only if at least one container has UrlTemplatization.
	for _, container := range cfg.WorkloadCollectorConfig {
		if container.UrlTemplatization != nil {
			optedIn = true
			rules = append(rules, container.UrlTemplatization.TemplatizationRules...)
		}
	}
	return rules, optedIn
}
