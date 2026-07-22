package services

import (
	"testing"

	"github.com/odigos-io/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
)

// TestOdigosConfigurationFromConfigMap_ProfilingUi guards the effective-config
// path used for live cache tuning: the scheduler serializes profiling.ui into
// the effective-config ConfigMap and the frontend parses it back here, so the
// watcher can Reconfigure the store without restarting the UI pod.
func TestOdigosConfigurationFromConfigMap_ProfilingUi(t *testing.T) {
	yamlCfg := `
profiling:
  enabled: true
  ui:
    maxSlots: 50
    slotMaxBytes: 1048576
    slotTTLSeconds: 60
`
	cm := &v1.ConfigMap{Data: map[string]string{consts.OdigosConfigurationFileName: yamlCfg}}

	cfg, err := OdigosConfigurationFromConfigMap(cm)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Profiling == nil || cfg.Profiling.Ui == nil {
		t.Fatalf("profiling.ui not parsed from effective-config")
	}
	ui := cfg.Profiling.Ui
	if ui.MaxSlots != 50 || ui.SlotMaxBytes != 1048576 || ui.SlotTTLSeconds != 60 {
		t.Fatalf("profiling.ui values wrong: %+v", ui)
	}
}
