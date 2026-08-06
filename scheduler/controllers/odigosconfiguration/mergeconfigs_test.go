package odigosconfiguration

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func boolPtr(b bool) *bool { return &b }

// TestMergeConfigs_ProfilingUi verifies that profiling.ui cache limits set via the
// remote-config overlay (the settings-page path, updateRemoteConfig) propagate into
// the effective config. Without the Ui merge, the settings-page edits are silently
// dropped here and never reach the frontend watcher / live Reconfigure.
func TestMergeConfigs_ProfilingUi(t *testing.T) {
	base := &common.OdigosConfiguration{
		Profiling: &common.ProfilingConfiguration{
			Enabled: boolPtr(true),
			Ui:      &common.ProfilingUiConfiguration{MaxSlots: 24, SlotMaxBytes: 8 << 20, SlotTTLSeconds: 120},
		},
	}
	overlay := &common.OdigosConfiguration{
		Profiling: &common.ProfilingConfiguration{
			// SlotMaxBytes left 0 => base value must be kept.
			Ui: &common.ProfilingUiConfiguration{MaxSlots: 50, SlotTTLSeconds: 60},
		},
	}

	mergeConfigs(base, overlay)

	if base.Profiling == nil || base.Profiling.Ui == nil {
		t.Fatalf("profiling.ui was dropped by mergeConfigs")
	}
	ui := base.Profiling.Ui
	if ui.MaxSlots != 50 {
		t.Errorf("MaxSlots = %d, want 50 (overlay override)", ui.MaxSlots)
	}
	if ui.SlotTTLSeconds != 60 {
		t.Errorf("SlotTTLSeconds = %d, want 60 (overlay override)", ui.SlotTTLSeconds)
	}
	if ui.SlotMaxBytes != 8<<20 {
		t.Errorf("SlotMaxBytes = %d, want %d (base kept; overlay field was 0)", ui.SlotMaxBytes, 8<<20)
	}
}

// TestMergeConfigs_ProfilingUi_EmptyBase verifies the overlay creates profiling.ui
// when the base has none.
func TestMergeConfigs_ProfilingUi_EmptyBase(t *testing.T) {
	base := &common.OdigosConfiguration{Profiling: &common.ProfilingConfiguration{Enabled: boolPtr(true)}}
	overlay := &common.OdigosConfiguration{
		Profiling: &common.ProfilingConfiguration{Ui: &common.ProfilingUiConfiguration{MaxSlots: 30}},
	}

	mergeConfigs(base, overlay)

	if base.Profiling.Ui == nil || base.Profiling.Ui.MaxSlots != 30 {
		t.Fatalf("overlay profiling.ui not applied to empty base: %+v", base.Profiling.Ui)
	}
}
