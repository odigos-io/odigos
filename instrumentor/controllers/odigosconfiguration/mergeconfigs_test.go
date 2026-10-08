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

func intPtr(i int) *int { return &i }

// intValue keeps assertion failures readable: the fields under test are *int, and printing
// the pointer shows an address instead of the value that was actually merged.
func intValue(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func liveTrafficLearning(cfg *common.OdigosConfiguration) *common.LiveTrafficLearningConfiguration {
	if cfg.CardinalityControl == nil || cfg.CardinalityControl.UrlTemplatization == nil {
		return nil
	}
	return cfg.CardinalityControl.UrlTemplatization.LiveTrafficLearning
}

func ltlConfig(ltl *common.LiveTrafficLearningConfiguration) *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		CardinalityControl: &common.CardinalityControlConfiguration{
			UrlTemplatization: &common.UrlTemplatizationCardinalityControlConfiguration{
				LiveTrafficLearning: ltl,
			},
		},
	}
}

// The live-traffic learning tunables are isHelmOnly: false in the config catalog, so the UI
// writes them into odigos-local-ui-config. Without this merge they never reach the effective
// config, and the gateway keeps exporting with the helm defaults while the UI reports the
// user's value as applied.
func TestMergeConfigs_LiveTrafficLearningOverlay(t *testing.T) {
	base := ltlConfig(&common.LiveTrafficLearningConfiguration{
		Enabled:                    boolPtr(true),
		MaxExamplePathsPerWorkload: intPtr(5000),
		PathExampleIdleTTL:         "48h",
		LearningInterval:           "30s",
		RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
			MinObservationsForRule:      intPtr(100),
			MinCardinalityForTemplating: intPtr(20),
		},
	})
	overlay := ltlConfig(&common.LiveTrafficLearningConfiguration{
		MaxExamplePathsPerWorkload: intPtr(20000),
		PathExampleIdleTTL:         "12h",
		// LearningInterval left empty => base value must be kept.
		RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
			MinObservationsForRule: intPtr(250),
		},
		AutomaticRules: &common.LiveTrafficLearningAutomaticRulesConfiguration{Enabled: boolPtr(true)},
	})

	mergeConfigs(base, overlay)

	ltl := liveTrafficLearning(base)
	if ltl == nil {
		t.Fatalf("cardinalityControl.urlTemplatization.liveTrafficLearning was dropped by mergeConfigs")
	}
	if got := intValue(ltl.MaxExamplePathsPerWorkload); got != 20000 {
		t.Errorf("MaxExamplePathsPerWorkload = %d, want 20000 (overlay override)", got)
	}
	if ltl.PathExampleIdleTTL != "12h" {
		t.Errorf("PathExampleIdleTTL = %q, want \"12h\" (overlay override)", ltl.PathExampleIdleTTL)
	}
	if ltl.LearningInterval != "30s" {
		t.Errorf("LearningInterval = %q, want \"30s\" (base kept; overlay field was empty)", ltl.LearningInterval)
	}
	if ltl.RuleComputation == nil {
		t.Fatalf("ruleComputation was dropped by mergeConfigs")
	}
	if got := intValue(ltl.RuleComputation.MinObservationsForRule); got != 250 {
		t.Errorf("MinObservationsForRule = %d, want 250 (overlay override)", got)
	}
	if got := intValue(ltl.RuleComputation.MinCardinalityForTemplating); got != 20 {
		t.Errorf("MinCardinalityForTemplating = %d, want 20 (base kept; overlay field was nil)", got)
	}
	if ltl.AutomaticRules == nil || ltl.AutomaticRules.Enabled == nil || !*ltl.AutomaticRules.Enabled {
		t.Errorf("AutomaticRules.Enabled not set to true by the overlay: %+v", ltl.AutomaticRules)
	}
	if ltl.Enabled == nil || !*ltl.Enabled {
		t.Errorf("Enabled = %v, want the helm value true to survive the merge", ltl.Enabled)
	}
}

// The overlay must be able to tune a feature helm turned on even when helm wrote no
// tunables of its own, which is the default install shape (only `enabled: true` is rendered).
func TestMergeConfigs_LiveTrafficLearningOverlay_EmptyBase(t *testing.T) {
	base := ltlConfig(&common.LiveTrafficLearningConfiguration{Enabled: boolPtr(true)})
	overlay := ltlConfig(&common.LiveTrafficLearningConfiguration{
		MaxExamplePathsPerWorkload: intPtr(20000),
		RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
			MinCardinalityForTemplating: intPtr(50),
		},
	})

	mergeConfigs(base, overlay)

	ltl := liveTrafficLearning(base)
	if ltl == nil || intValue(ltl.MaxExamplePathsPerWorkload) != 20000 {
		t.Fatalf("overlay maxExamplePathsPerWorkload not applied to empty base: %+v", ltl)
	}
	if ltl.RuleComputation == nil || intValue(ltl.RuleComputation.MinCardinalityForTemplating) != 50 {
		t.Fatalf("overlay ruleComputation not applied to empty base: %+v", ltl.RuleComputation)
	}
}

// liveTrafficLearning.enabled is isHelmOnly: true because turning it on renders the
// odigos-cache Deployment and Service. An overlay must not be able to enable the feature,
// or the gateway exporter would target a Service that was never created.
func TestMergeConfigs_LiveTrafficLearningEnabledIsHelmOnly(t *testing.T) {
	base := &common.OdigosConfiguration{}
	overlay := ltlConfig(&common.LiveTrafficLearningConfiguration{
		Enabled:                    boolPtr(true),
		MaxExamplePathsPerWorkload: intPtr(20000),
	})

	mergeConfigs(base, overlay)

	ltl := liveTrafficLearning(base)
	if ltl == nil {
		t.Fatalf("overlay tunables were dropped entirely")
	}
	if ltl.Enabled != nil {
		t.Errorf("Enabled = %v, want nil: an overlay must not turn on a helm-only feature", *ltl.Enabled)
	}
	if common.UrlTemplatizationLiveTrafficLearningActive(base.CardinalityControl) {
		t.Errorf("overlay must not make live traffic learning active")
	}
}

// A base config with no cardinalityControl at all (live traffic learning off, the default)
// must not grow one just because an unrelated overlay was merged.
func TestMergeConfigs_NoCardinalityControlOverlayLeavesBaseUntouched(t *testing.T) {
	base := &common.OdigosConfiguration{}
	overlay := &common.OdigosConfiguration{ClusterName: "prod"}

	mergeConfigs(base, overlay)

	if base.CardinalityControl != nil {
		t.Errorf("CardinalityControl = %+v, want nil when the overlay sets none", base.CardinalityControl)
	}
}
