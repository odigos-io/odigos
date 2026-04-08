package odigosconfiguration

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestHasTopLevelField(t *testing.T) {
	t.Run("detects explicit top-level field", func(t *testing.T) {
		yamlBytes := []byte("telemetryEnabled: false\nclusterName: test\n")
		if !hasTopLevelField(yamlBytes, "telemetryEnabled") {
			t.Fatalf("expected telemetryEnabled to be detected as top-level field")
		}
	})

	t.Run("returns false when field is absent", func(t *testing.T) {
		yamlBytes := []byte("clusterName: test\n")
		if hasTopLevelField(yamlBytes, "telemetryEnabled") {
			t.Fatalf("expected telemetryEnabled to be absent")
		}
	})

	t.Run("ignores nested field with same name", func(t *testing.T) {
		yamlBytes := []byte("instrumentor:\n  telemetryEnabled: true\n")
		if hasTopLevelField(yamlBytes, "telemetryEnabled") {
			t.Fatalf("expected only top-level telemetryEnabled to match")
		}
	})
}

func TestMergeConfigsTelemetryEnabled(t *testing.T) {
	t.Run("applies explicit false override", func(t *testing.T) {
		base := &common.OdigosConfiguration{TelemetryEnabled: true}
		overlay := &common.OdigosConfiguration{TelemetryEnabled: false}

		mergeConfigs(base, overlay, true)

		if base.TelemetryEnabled {
			t.Fatalf("expected telemetryEnabled=false after explicit override")
		}
	})

	t.Run("does not override when field not explicitly set", func(t *testing.T) {
		base := &common.OdigosConfiguration{TelemetryEnabled: true}
		overlay := &common.OdigosConfiguration{TelemetryEnabled: false}

		mergeConfigs(base, overlay, false)

		if !base.TelemetryEnabled {
			t.Fatalf("expected telemetryEnabled to remain true when overlay field is absent")
		}
	})
}
