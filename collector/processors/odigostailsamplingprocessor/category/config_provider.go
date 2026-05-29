package category

import (
	"go.opentelemetry.io/collector/pdata/pcommon"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
)

// TailSamplingConfigProvider resolves tail sampling configuration for a trace resource.
type TailSamplingConfigProvider interface {
	GetTailSamplingConfig(resource pcommon.Resource) (*commonapisampling.TailSamplingSourceConfig, bool)
}
