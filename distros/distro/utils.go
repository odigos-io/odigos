package distro

import "github.com/odigos-io/odigos/common"

// IsRestartRequired returns whether the distribution requires application restart in order to be injected.
// it does not specify whether a restart should be initiated or not, just whether it is required.
func IsRestartRequired(d *OtelDistro, config *common.OdigosConfiguration) bool {
	if d == nil {
		return false
	}
	if d.RuntimeAgent == nil {
		return false
	}
	// currently if wasp is enabled and supported by the distribution, restart is required
	if config.WaspEnabled != nil && *config.WaspEnabled && d.RuntimeAgent.WaspSupported {
		return true
	}
	return !d.RuntimeAgent.NoRestartRequired
}
