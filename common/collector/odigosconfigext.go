package collector

import (
	"context"

	"go.opentelemetry.io/collector/pdata/pcommon"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// WorkloadConfigCacheCallback is notified when the extension's workload cache changes.
// The callback receives generic container collector config; any processor that cares about config updates can implement this.
//
// Key semantics: both OnSet and OnDeleteKey use the same full cache key format (e.g. "namespace/kind/name/containerName").
// The extension applies new state first (OnSet for each current container), then calls OnDeleteKey for any key that was
// removed so the consumer never sees a gap where the workload is briefly empty.
type WorkloadConfigCacheCallback interface {
	OnSet(key string, cfg *commonapi.ContainerCollectorConfig)
	OnDeleteKey(key string)
}

// OdigosConfigExtension is the interface that must be implemented by an extension that wants to provide Odigos configuration.
// Every platform (k8s, vm) can implement this interface to provide it's own processor extension to fetch the config from where it is stored.
type OdigosConfigExtension interface {
	// GetFromResource returns the container collector config for the given resource if it exists.
	GetFromResource(res pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool)

	// IsActiveSource reports whether the workload identified on the resource (namespace/kind/name)
	// is an active Odigos Source i.e. the extension currently holds an InstrumentationConfig for it
	IsActiveSource(res pcommon.Resource) bool

	// GetWorkloadCacheKey returns the cache key for the container identified by the given resource.
	// Key format is platform-specific (e.g. "namespace/kind/name/containerName" for K8s). Processors use this
	// to look up their own caches without duplicating key logic.
	GetWorkloadCacheKey(res pcommon.Resource) (string, error)

	// GetWorkloadIdentityFromResource returns the workload cache key and the identifying resource attributes
	// for the container on the given resource. Attribute keys match those on the source resource
	// (e.g. k8s.namespace.name, a workload name attribute, k8s.container.name).
	GetWorkloadIdentityFromResource(res pcommon.Resource) (cacheKey string, attrs pcommon.Map, err error)

	// RegisterWorkloadConfigCacheCallback registers a callback that is invoked when the extension's workload cache is updated.
	// Processors (e.g. URL templatization) use this to keep their caches in sync without polling.
	RegisterWorkloadConfigCacheCallback(cb WorkloadConfigCacheCallback)

	// UnregisterWorkloadConfigCacheCallback removes the callback. Processors should call this in Shutdown so the extension
	// stops invoking the callback and can release references; allows the collector to prune caches when a processor is removed from the pipeline.
	UnregisterWorkloadConfigCacheCallback(cb WorkloadConfigCacheCallback)

	// WaitForCacheSync blocks until the extension's workload cache has synced (e.g. initial list from API) or ctx is done.
	// Returns true if synced successfully, false if context canceled or sync failed. Callers that depend on the cache
	// should call this before processing (e.g. in Start) to avoid missing data on startup.
	WaitForCacheSync(ctx context.Context) bool

	// GetDataStreamsForWorkload returns the data stream names the workload belongs to.
	// Derives workload identity from resource attributes (namespace, kind, name).
	// Returns (nil, false) if the workload is not found in the cache.
	GetDataStreamsForWorkload(res pcommon.Resource) ([]string, bool)
}
