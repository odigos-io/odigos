package collector

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	commonapi "github.com/odigos-io/odigos/common/api"
)

// OdigosConfigExtension is the interface that must be implemented by an extension that wants to provide Odigos configuration.
// Every platform (k8s, vm) can implement this interface to provide it's own processor extension to fetch the config from where it is stored.
type OdigosConfigExtension interface {

	// givin a specific resource, return a source collector config if exists.
	GetFromResource(res pcommon.Resource) (*commonapi.ContainerCollectorConfig, bool)
}

// UrlTemplatizationCacheCallback is notified when the extension's workload cache changes.
// Defined here so both the extension and the URL template processor use the same interface type
// (required for the processor's type assertion ext.(UrlTemplatizationCacheNotifier) to succeed).
type UrlTemplatizationCacheCallback interface {
	OnSet(key string, cfg *commonapi.ContainerCollectorConfig)
	OnDeleteKey(key string)
}

// UrlTemplatizationCacheNotifier is implemented by the extension so the processor can register a callback.
// Defined here so both use the same interface type.
type UrlTemplatizationCacheNotifier interface {
	RegisterUrlTemplatizationCacheCallback(cb UrlTemplatizationCacheCallback)
}
