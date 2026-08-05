package recommendations

import (
	"testing"

	"github.com/odigos-io/odigos/common"
)

func TestSetConfigPath(t *testing.T) {
	root := map[string]any{}
	if err := setConfigPath(root, "sampling.k8sHealthProbesSampling.enabled", true); err != nil {
		t.Fatalf("setConfigPath: %v", err)
	}
	if err := setConfigPath(root, "sampling.k8sHealthProbesSampling.keepPercentage", float64(2)); err != nil {
		t.Fatalf("setConfigPath keepPercentage: %v", err)
	}

	sampling, ok := root["sampling"].(map[string]any)
	if !ok {
		t.Fatalf("sampling type = %T", root["sampling"])
	}
	probes, ok := sampling["k8sHealthProbesSampling"].(map[string]any)
	if !ok {
		t.Fatalf("k8sHealthProbesSampling type = %T", sampling["k8sHealthProbesSampling"])
	}
	if probes["enabled"] != true {
		t.Fatalf("enabled = %#v, want true", probes["enabled"])
	}
	if probes["keepPercentage"] != float64(2) {
		t.Fatalf("keepPercentage = %#v, want 2", probes["keepPercentage"])
	}
}

func TestApplyStepsToConfig(t *testing.T) {
	cfg := common.OdigosConfiguration{}
	steps := []RemediationStep{
		{
			Type:  RemediationStepTypeEditConfig,
			Path:  "sampling.k8sHealthProbesSampling.enabled",
			Value: true,
		},
		{
			Type:  RemediationStepTypeEditConfig,
			Path:  "sampling.k8sHealthProbesSampling.keepPercentage",
			Value: 2,
		},
	}

	if err := ApplyStepsToConfig(&cfg, steps); err != nil {
		t.Fatalf("ApplyStepsToConfig: %v", err)
	}
	if cfg.Sampling == nil || cfg.Sampling.K8sHealthProbesSampling == nil {
		t.Fatal("expected k8sHealthProbesSampling to be set")
	}
	if cfg.Sampling.K8sHealthProbesSampling.Enabled == nil || !*cfg.Sampling.K8sHealthProbesSampling.Enabled {
		t.Fatalf("enabled = %#v, want true", cfg.Sampling.K8sHealthProbesSampling.Enabled)
	}
	if cfg.Sampling.K8sHealthProbesSampling.KeepPercentage == nil || *cfg.Sampling.K8sHealthProbesSampling.KeepPercentage != 2 {
		t.Fatalf("keepPercentage = %#v, want 2", cfg.Sampling.K8sHealthProbesSampling.KeepPercentage)
	}
}
