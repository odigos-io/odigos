package pro

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

func TestUsesOdigosRegistry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config common.OdigosConfiguration
		want   bool
	}{
		{
			name:   "default",
			config: common.OdigosConfiguration{},
			want:   true,
		},
		{
			name:   "openshift",
			config: common.OdigosConfiguration{OpenshiftEnabled: true},
			want:   false,
		},
		{
			name:   "custom prefix",
			config: common.OdigosConfiguration{ImagePrefix: "myregistry.io/odigos"},
			want:   false,
		},
		{
			name:   "explicit odigos prefix",
			config: common.OdigosConfiguration{ImagePrefix: k8sconsts.OdigosImagePrefix},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := UsesOdigosRegistry(&tt.config); got != tt.want {
				t.Fatalf("UsesOdigosRegistry() = %v, want %v", got, tt.want)
			}
		})
	}
}
