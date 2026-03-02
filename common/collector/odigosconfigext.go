package collector

import (
	commonapi "github.com/odigos-io/odigos/common/api"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OdigosConfigExtension interface {
	GetConfigFromResourceAttributes(res resource.Resource) (*commonapi.ContainerCollectorConfig, bool)
}
