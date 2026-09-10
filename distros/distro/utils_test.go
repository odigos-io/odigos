package distro

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func boolPtr(b bool) *bool { return &b }

func TestIsRestartRequired(t *testing.T) {
	emptyConfig := &common.OdigosConfiguration{}

	tests := []struct {
		name   string
		distro *OtelDistro
		config *common.OdigosConfiguration
		want   bool
	}{
		{
			name:   "nil distro does not require restart",
			distro: nil,
			config: emptyConfig,
			want:   false,
		},
		{
			// regression: browser distros have no in-pod RuntimeAgent, so the BrowserSidecar
			// check must run before the RuntimeAgent nil-check, otherwise this returns false.
			name:   "browser distro requires restart despite nil runtime agent",
			distro: &OtelDistro{BrowserSidecar: &BrowserSidecar{AgentDirectory: "{{ODIGOS_AGENTS_DIR}}/browser", AgentFileName: "agent.js"}},
			config: emptyConfig,
			want:   true,
		},
		{
			name:   "distro with no runtime agent and no browser sidecar does not require restart",
			distro: &OtelDistro{},
			config: emptyConfig,
			want:   false,
		},
		{
			name:   "runtime agent without NoRestartRequired requires restart",
			distro: &OtelDistro{RuntimeAgent: &RuntimeAgent{}},
			config: emptyConfig,
			want:   true,
		},
		{
			name:   "runtime agent with NoRestartRequired does not require restart",
			distro: &OtelDistro{RuntimeAgent: &RuntimeAgent{NoRestartRequired: true}},
			config: emptyConfig,
			want:   false,
		},
		{
			name:   "wasp enabled and supported requires restart even when NoRestartRequired",
			distro: &OtelDistro{RuntimeAgent: &RuntimeAgent{NoRestartRequired: true, WaspSupported: true}},
			config: &common.OdigosConfiguration{WaspEnabled: boolPtr(true)},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRestartRequired(tt.distro, tt.config); got != tt.want {
				t.Errorf("IsRestartRequired() = %v, want %v", got, tt.want)
			}
		})
	}
}
