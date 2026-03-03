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
