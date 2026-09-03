package clustercollector

import (
	"slices"

	"github.com/odigos-io/odigos/common/config"
)

// odigos_enterprise_auth is only compiled into the enterprise collector image, and the collector
// rejects unknown component types at config load even when nothing references them, so callers
// must gate this on tier.
const odigosEnterpriseAuthExtensionName = "odigos_enterprise_auth"

func addEnterpriseAuthExtension(c *config.Config) {
	if c.Extensions == nil {
		c.Extensions = config.GenericMap{}
	}
	c.Extensions[odigosEnterpriseAuthExtensionName] = config.GenericMap{}

	if !slices.Contains(c.Service.Extensions, odigosEnterpriseAuthExtensionName) {
		c.Service.Extensions = append(c.Service.Extensions, odigosEnterpriseAuthExtensionName)
	}
}
