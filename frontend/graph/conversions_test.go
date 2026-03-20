package graph

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestConvertCollectorGatewayToModelDefaultsServiceGraphDisabled(t *testing.T) {
	got, err := convertCollectorGatewayToModel(&common.CollectorGatewayConfiguration{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ServiceGraphDisabled == nil {
		t.Fatalf("expected ServiceGraphDisabled to be set")
	}
	if *got.ServiceGraphDisabled {
		t.Fatalf("expected ServiceGraphDisabled default to false")
	}
}

func TestConvertCollectorGatewayToModelPreservesServiceGraphDisabled(t *testing.T) {
	disabled := true
	got, err := convertCollectorGatewayToModel(&common.CollectorGatewayConfiguration{
		ServiceGraph: &common.ServiceGraphOptions{
			Disabled: &disabled,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil || got.ServiceGraphDisabled == nil || !*got.ServiceGraphDisabled {
		t.Fatalf("expected ServiceGraphDisabled to remain true")
	}
}
