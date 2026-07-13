package clustercollectorsgroup

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestResolveServiceGraphOptionsDefaultsWhenGatewayConfigMissing(t *testing.T) {
	options := resolveServiceGraphOptions(nil)

	if options.Disabled == nil {
		t.Fatalf("expected Disabled to be defaulted")
	}
	if *options.Disabled {
		t.Fatalf("expected Disabled default to be false")
	}
}

func TestResolveServiceGraphOptionsDefaultsWhenServiceGraphMissing(t *testing.T) {
	options := resolveServiceGraphOptions(&common.CollectorGatewayConfiguration{})

	if options.Disabled == nil {
		t.Fatalf("expected Disabled to be defaulted")
	}
	if *options.Disabled {
		t.Fatalf("expected Disabled default to be false")
	}
}

func TestResolveServiceGraphOptionsPreservesConfiguredValues(t *testing.T) {
	disabled := true
	options := resolveServiceGraphOptions(&common.CollectorGatewayConfiguration{
		ServiceGraph: &common.ServiceGraphOptions{
			Disabled:                  &disabled,
			ExtraDimensions:           []string{"k8s.namespace.name"},
			VirtualNodePeerAttributes: []string{"peer.service"},
		},
	})

	if options.Disabled == nil || !*options.Disabled {
		t.Fatalf("expected Disabled to remain true")
	}
	if len(options.ExtraDimensions) != 1 || options.ExtraDimensions[0] != "k8s.namespace.name" {
		t.Fatalf("expected ExtraDimensions to be preserved, got %v", options.ExtraDimensions)
	}
	if len(options.VirtualNodePeerAttributes) != 1 || options.VirtualNodePeerAttributes[0] != "peer.service" {
		t.Fatalf("expected VirtualNodePeerAttributes to be preserved, got %v", options.VirtualNodePeerAttributes)
	}
}
