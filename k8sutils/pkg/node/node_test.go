package node

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
)

// withGKEAutopilotEnv sets GKE_AUTOPILOT for one test, or removes it when value is nil.
func withGKEAutopilotEnv(t *testing.T, value *string) {
	t.Helper()

	// t.Setenv captures the original value and restores it on cleanup either way.
	t.Setenv(k8sconsts.GKEAutopilotEnvVar, "")
	if value == nil {
		require.NoError(t, os.Unsetenv(k8sconsts.GKEAutopilotEnvVar))
		return
	}
	t.Setenv(k8sconsts.GKEAutopilotEnvVar, *value)
}

func gkeAutopilotEnv(value string) *string { return &value }

func TestIsGKEManagedNode(t *testing.T) {
	tests := []struct {
		name      string
		autoPilot *string
		nodeName  string
		want      bool
	}{
		{
			name:     "regular node on a standard cluster",
			nodeName: "ip-10-0-1-15.ec2.internal",
			want:     false,
		},
		{
			// GKE Autopilot names its nodes gk3-<cluster>-<pool>-<hash>.
			name:     "autopilot node detected by its name",
			nodeName: "gk3-my-cluster-pool-1-abcd1234",
			want:     true,
		},
		{
			name:      "forced on for a node that does not look managed",
			autoPilot: gkeAutopilotEnv("true"),
			nodeName:  "ip-10-0-1-15.ec2.internal",
			want:      true,
		},
		{
			name:      "forced on with a numeric boolean",
			autoPilot: gkeAutopilotEnv("1"),
			nodeName:  "ip-10-0-1-15.ec2.internal",
			want:      true,
		},
		{
			name:      "forced on with a capitalised boolean",
			autoPilot: gkeAutopilotEnv("TRUE"),
			nodeName:  "ip-10-0-1-15.ec2.internal",
			want:      true,
		},
		{
			name:      "explicitly off does not override the node name",
			autoPilot: gkeAutopilotEnv("false"),
			nodeName:  "gk3-my-cluster-pool-1-abcd1234",
			want:      true,
		},
		{
			name:      "an unparsable override is ignored",
			autoPilot: gkeAutopilotEnv("yes"),
			nodeName:  "ip-10-0-1-15.ec2.internal",
			want:      false,
		},
		{
			name:      "an unparsable override still leaves the node name check",
			autoPilot: gkeAutopilotEnv("yes"),
			nodeName:  "gk3-my-cluster-pool-1-abcd1234",
			want:      true,
		},
		{
			name:      "an empty override is ignored",
			autoPilot: gkeAutopilotEnv(""),
			nodeName:  "gk3-my-cluster-pool-1-abcd1234",
			want:      true,
		},
		{
			name:     "the prefix includes the separator",
			nodeName: "gk3node-1",
			want:     false,
		},
		{
			name:     "the prefix must be at the start of the name",
			nodeName: "node-gk3-1",
			want:     false,
		},
		{
			name:     "the prefix is case sensitive",
			nodeName: "GK3-my-cluster-pool-1",
			want:     false,
		},
		{
			name:     "empty node name",
			nodeName: "",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withGKEAutopilotEnv(t, tt.autoPilot)

			assert.Equal(t, tt.want, IsGKEManagedNode(tt.nodeName))
		})
	}
}

func TestDetermineNodeOdigletInstalledLabelByTier(t *testing.T) {
	tests := []struct {
		name string
		tier string
		want string
	}{
		// The label literals are pinned because the instrumentor selects on them for pod node
		// affinity and the uninstall path looks them up by name, so a rename is a silent break.
		{name: "community", tier: string(common.CommunityOdigosTier), want: "odigos.io/odiglet-oss-installed"},
		{name: "onprem", tier: string(common.OnPremOdigosTier), want: "odigos.io/odiglet-enterprise-installed"},
		{name: "cloud falls back to the community label", tier: string(common.CloudOdigosTier), want: "odigos.io/odiglet-oss-installed"},
		{name: "unset", tier: "", want: "odigos.io/odiglet-oss-installed"},
		{name: "unknown tier", tier: "enterprise", want: "odigos.io/odiglet-oss-installed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(consts.OdigosTierEnvVarName, tt.tier)

			assert.Equal(t, tt.want, DetermineNodeOdigletInstalledLabelByTier())
		})
	}
}

func TestDetermineNodeOdigletInstalledLabelByTierMatchesTheSharedLabelMaps(t *testing.T) {
	t.Setenv(consts.OdigosTierEnvVarName, string(common.CommunityOdigosTier))
	assert.Contains(t, k8sconsts.OdigletOSSInstalled, DetermineNodeOdigletInstalledLabelByTier())

	t.Setenv(consts.OdigosTierEnvVarName, string(common.OnPremOdigosTier))
	assert.Contains(t, k8sconsts.OdigletEnterpriseInstalled, DetermineNodeOdigletInstalledLabelByTier())
}
