package odigosconfigk8sextension

import (
	"context"
	"strings"
	"sync"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.uber.org/zap"

	"k8s.io/client-go/dynamic/dynamicinformer"

	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common/collector"
)

// OdigosWorkloadConfig is an extension that runs a dynamic informer for InstrumentationConfigs
// and maintains a cache of workload sampling config keyed by WorkloadKey (namespace, kind, name).
type OdigosWorkloadConfig struct {
	cache           *cache
	logger          *zap.Logger
	cancel          context.CancelFunc
	informerFactory dynamicinformer.DynamicSharedInformerFactory // set when in-cluster; nil otherwise

	// workloadKeysIndex maps key prefix (e.g. "ns/kind/name/") to set of full cache keys for that workload.
	// Used to get keys by prefix without iterating the full cache.
	workloadKeysIndex   map[string]map[string]struct{}
	workloadIndexMu     sync.RWMutex
}

// OdigosConfigExtension is the interface that must be implemented by an extension that wants to provide Odigos configuration.
var _ collector.OdigosConfigExtension = (*OdigosWorkloadConfig)(nil)

// NewOdigosConfig creates a new OdigosConfig extension.
func NewOdigosConfig(settings component.TelemetrySettings) (*OdigosWorkloadConfig, error) {
	return &OdigosWorkloadConfig{
		cache:             newCache(),
		logger:            settings.Logger,
		workloadKeysIndex: make(map[string]map[string]struct{}),
	}, nil
}

// keyPrefixFromKey returns the workload prefix for a full cache key (e.g. "ns/kind/name/container" -> "ns/kind/name/").
func keyPrefixFromKey(key string) string {
	i := strings.LastIndex(key, "/")
	if i < 0 {
		return ""
	}
	return key[:i+1]
}

func (o *OdigosWorkloadConfig) addKeyToIndex(key string) {
	prefix := keyPrefixFromKey(key)
	if prefix == "" {
		return
	}
	o.workloadIndexMu.Lock()
	defer o.workloadIndexMu.Unlock()
	if o.workloadKeysIndex[prefix] == nil {
		o.workloadKeysIndex[prefix] = make(map[string]struct{})
	}
	o.workloadKeysIndex[prefix][key] = struct{}{}
}

func (o *OdigosWorkloadConfig) removeKeyFromIndex(key string) {
	prefix := keyPrefixFromKey(key)
	if prefix == "" {
		return
	}
	o.workloadIndexMu.Lock()
	defer o.workloadIndexMu.Unlock()
	delete(o.workloadKeysIndex[prefix], key)
	if len(o.workloadKeysIndex[prefix]) == 0 {
		delete(o.workloadKeysIndex, prefix)
	}
}

// getKeysForPrefix returns a copy of the full cache keys that have the given prefix. Caller must not modify the result.
func (o *OdigosWorkloadConfig) getKeysForPrefix(prefix string) []string {
	o.workloadIndexMu.RLock()
	defer o.workloadIndexMu.RUnlock()
	set := o.workloadKeysIndex[prefix]
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
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

func (o *OdigosWorkloadConfig) GetFromResource(res pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool) {
	key, err := workloadKeyFromResourceAttributes(res.Attributes())
	if err != nil {
		return nil, false
	}
	return o.cache.Get(key)
}

// GetWorkloadCacheKey returns the cache key for the container identified by the given resource.
// Processors use this to look up their own caches without duplicating key logic.
// Key format: "namespace/kind/name/containerName".
func (o *OdigosWorkloadConfig) GetWorkloadCacheKey(res pcommon.Resource) (string, error) {
	return workloadKeyFromResourceAttributes(res.Attributes())
}

// RegisterWorkloadConfigCacheCallback registers a callback that is invoked by the extension
// cache on every Set/Delete. The extension passes the callback to the cache; the informer
// only calls cache.Set and cache.Delete. Backfill replays current cache state so the
// processor starts in sync.
func (o *OdigosWorkloadConfig) RegisterWorkloadConfigCacheCallback(cb collector.WorkloadConfigCacheCallback) {
	o.cache.addCallback(cb)
	o.logger.Debug("workload config cache callback registered")
	backfillCount := 0
	o.cache.Range(func(key string, cfg *commonapi.ContainerCollectorConfig) {
		cb.OnSet(key, cfg)
		backfillCount++
	})
	if backfillCount > 0 {
		o.logger.Debug("workload config callback backfill replayed", zap.Int("entries", backfillCount))
	}
}
