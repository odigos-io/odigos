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
	// Browser distributions are delivered by the odigos-browser-proxy sidecar (plus its iptables init container), which is injected into the pod manifest by the instrumentor webhook at pod creation.
	// Unlike in-pod runtime agents, this cannot be applied to already-running pods, so a restart/rollout is always required for the sidecar to be added.
	if d.BrowserSidecar != nil {
		return true
	}
	// currently if wasp is enabled and supported by the distribution, restart is required
	if config.WaspEnabled != nil && *config.WaspEnabled && d.RuntimeAgent.WaspSupported {
		return true
	}
	return !d.RuntimeAgent.NoRestartRequired
}

// RuntimeAgentOptionValue returns the value of the specified option for the runtime agent, and a boolean indicating whether the option was found or not.
// assuming the option name is unique across the options for the distribution.
// if duplicate option names are present, the first one will be returned.
func RuntimeAgentOptionValue(d *OtelDistro, optionName string) (string, bool) {
	if d == nil || d.RuntimeAgent == nil {
		return "", false
	}
	for _, option := range d.RuntimeAgent.Options {
		if option.Name == optionName {
			return option.Value, true
		}
	}
	return "", false
}
