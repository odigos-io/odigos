package collectorconfig

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
)

// The odigos_capabilities extension raises the collector process's effective capabilities to its
// permitted set on startup, which the node collector needs to keep its eBPF capabilities when
// running as a non-root user. An extension only starts if it is listed in service.extensions, and
// listing one that is not defined fails the collector's config validation, so unlike the
// enterprise-only extensions this one must be both defined and enabled at every tier.
func TestCommonConfig_CapabilitiesExtensionEnabledAtEveryTier(t *testing.T) {
	for _, tier := range []common.OdigosTier{common.CommunityOdigosTier, common.OnPremOdigosTier, common.CloudOdigosTier} {
		t.Run(string(tier), func(t *testing.T) {
			cfg := CommonConfig(tier)

			_, defined := cfg.Extensions[consts.OdigosCapabilitiesExtensionType]
			assert.True(t, defined, "extensions map is missing %q", consts.OdigosCapabilitiesExtensionType)
			assert.True(t, contains(cfg.Service.Extensions, consts.OdigosCapabilitiesExtensionType),
				"service.extensions %v does not enable %q", cfg.Service.Extensions, consts.OdigosCapabilitiesExtensionType)
		})
	}
}
