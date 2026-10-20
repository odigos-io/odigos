package services

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestApplyInterrogationInput_ModeAndReset(t *testing.T) {
	enabled := true
	cfg := &common.OdigosConfiguration{
		Interrogation: &common.InterrogationConfiguration{Enabled: &enabled, Mode: common.InterrogationModeAuto},
	}
	off := model.InterrogationModeOff
	applyInterrogationInput(cfg, &model.LocalUIConfigInterrogationInput{Mode: &off})
	if cfg.Interrogation.Mode != common.InterrogationModeOff {
		t.Fatalf("mode = %q, want off", cfg.Interrogation.Mode)
	}
	if cfg.Interrogation.Enabled == nil || !*cfg.Interrogation.Enabled {
		t.Fatal("enabled was cleared")
	}

	applyInterrogationInput(cfg, &model.LocalUIConfigInterrogationInput{ResetPaths: []string{"interrogation.mode"}})
	if cfg.Interrogation.Mode != "" {
		t.Fatalf("mode = %q, want empty after reset", cfg.Interrogation.Mode)
	}
	if cfg.Interrogation.Enabled == nil || !*cfg.Interrogation.Enabled {
		t.Fatal("reset cleared enabled")
	}
}
