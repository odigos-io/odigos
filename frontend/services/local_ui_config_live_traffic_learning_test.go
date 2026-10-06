package services

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ltlBoolPtr(b bool) *bool    { return &b }
func ltlIntPtr(i int) *int       { return &i }
func ltlStrPtr(s string) *string { return &s }

// ltlInput wraps a live traffic learning input at the nesting depth
// LocalUIConfigInput uses, so the tests drive the real applyLocalUiConfigInput
// entry point rather than the inner helper.
func ltlInput(in *model.LocalUIConfigLiveTrafficLearningInput) model.LocalUIConfigInput {
	return model.LocalUIConfigInput{
		CardinalityControl: &model.LocalUIConfigCardinalityControlInput{
			URLTemplatization: &model.LocalUIConfigURLTemplatizationCardinalityControlInput{
				LiveTrafficLearning: in,
			},
		},
	}
}

// ltlStoredConfig is an overlay document that already carries every live traffic
// learning value, so a partial edit can be checked for collateral damage.
// Enabled is set the way helm would set it: the GraphQL input has no field for it,
// so it must survive every UI edit.
func ltlStoredConfig() *common.OdigosConfiguration {
	return &common.OdigosConfiguration{
		CardinalityControl: &common.CardinalityControlConfiguration{
			UrlTemplatization: &common.UrlTemplatizationCardinalityControlConfiguration{
				LiveTrafficLearning: &common.LiveTrafficLearningConfiguration{
					Enabled:                    ltlBoolPtr(true),
					MaxExamplePathsPerWorkload: ltlIntPtr(100),
					PathExampleIdleTTL:         "1h",
					LearningInterval:           "10s",
					RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
						MinObservationsForRule:      ltlIntPtr(11),
						MinCardinalityForTemplating: ltlIntPtr(12),
					},
					AutomaticRules: &common.LiveTrafficLearningAutomaticRulesConfiguration{
						Enabled: ltlBoolPtr(false),
					},
				},
			},
		},
	}
}

func ltlStored(t *testing.T, cfg *common.OdigosConfiguration) *common.LiveTrafficLearningConfiguration {
	t.Helper()
	require.NotNil(t, cfg.CardinalityControl)
	require.NotNil(t, cfg.CardinalityControl.UrlTemplatization)
	require.NotNil(t, cfg.CardinalityControl.UrlTemplatization.LiveTrafficLearning)
	return cfg.CardinalityControl.UrlTemplatization.LiveTrafficLearning
}

// ltlSingleFieldEdits sets exactly one leaf of the GraphQL input. Each leaf has
// its own guard in applyCardinalityControlInput, and the "nothing to do" check in
// front of them enumerates all five top-level leaves by hand, so a leaf missing
// from that check is dropped on the floor whenever it is edited on its own -
// which is precisely what the settings screen sends.
var ltlSingleFieldEdits = map[string]struct {
	edit *model.LocalUIConfigLiveTrafficLearningInput
	want *common.LiveTrafficLearningConfiguration
}{
	"MaxExamplePathsPerWorkload": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{MaxExamplePathsPerWorkload: ltlIntPtr(9000)},
		want: &common.LiveTrafficLearningConfiguration{MaxExamplePathsPerWorkload: ltlIntPtr(9000)},
	},
	"PathExampleIdleTTL": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{PathExampleIdleTTL: ltlStrPtr("72h")},
		want: &common.LiveTrafficLearningConfiguration{PathExampleIdleTTL: "72h"},
	},
	"LearningInterval": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{LearningInterval: ltlStrPtr("45s")},
		want: &common.LiveTrafficLearningConfiguration{LearningInterval: "45s"},
	},
	"RuleComputation.MinObservationsForRule": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{
			RuleComputation: &model.LocalUIConfigLiveTrafficLearningRuleComputationInput{
				MinObservationsForRule: ltlIntPtr(250),
			},
		},
		want: &common.LiveTrafficLearningConfiguration{
			RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
				MinObservationsForRule: ltlIntPtr(250),
			},
		},
	},
	"RuleComputation.MinCardinalityForTemplating": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{
			RuleComputation: &model.LocalUIConfigLiveTrafficLearningRuleComputationInput{
				MinCardinalityForTemplating: ltlIntPtr(35),
			},
		},
		want: &common.LiveTrafficLearningConfiguration{
			RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
				MinCardinalityForTemplating: ltlIntPtr(35),
			},
		},
	},
	"AutomaticRules.Enabled": {
		edit: &model.LocalUIConfigLiveTrafficLearningInput{
			AutomaticRules: &model.LocalUIConfigLiveTrafficLearningAutomaticRulesInput{
				Enabled: ltlBoolPtr(true),
			},
		},
		want: &common.LiveTrafficLearningConfiguration{
			AutomaticRules: &common.LiveTrafficLearningAutomaticRulesConfiguration{
				Enabled: ltlBoolPtr(true),
			},
		},
	},
}

// Editing one setting must build the three nested blocks the overlay document
// needs and write that setting and nothing else. Whole-struct equality rather
// than a single field assertion: these leaves are two ints, two pointers to int
// and two bools, so a guard that writes the wrong destination still type-checks.
func TestApplyCardinalityControlInput_EachFieldAloneReachesTheOverlay(t *testing.T) {
	for name, tc := range ltlSingleFieldEdits {
		t.Run(name, func(t *testing.T) {
			cfg := &common.OdigosConfiguration{}
			applyLocalUiConfigInput(cfg, ltlInput(tc.edit))

			require.NotNil(t, cfg.CardinalityControl, "cardinalityControl block was not created")
			require.NotNil(t, cfg.CardinalityControl.UrlTemplatization, "urlTemplatization block was not created")
			assert.Equal(t, tc.want, cfg.CardinalityControl.UrlTemplatization.LiveTrafficLearning)
		})
	}
}

// The same edits applied to a document that already holds every value must move
// exactly one field. Enabled is the row that matters most: the GraphQL input
// cannot express it, so a writer that rebuilds the block instead of patching it
// would silently turn live traffic learning off on the next settings save.
func TestApplyCardinalityControlInput_PartialEditKeepsEveryStoredSibling(t *testing.T) {
	for name, tc := range ltlSingleFieldEdits {
		t.Run(name, func(t *testing.T) {
			cfg := ltlStoredConfig()
			applyLocalUiConfigInput(cfg, ltlInput(tc.edit))

			want := ltlStoredConfig()
			ltlApplyToCommon(t, name, want)

			require.NotEqual(t, ltlStoredConfig(), want, "the fixture for %s changes nothing", name)
			assert.Equal(t, want, cfg)
			assert.Equal(t, ltlBoolPtr(true), ltlStored(t, cfg).Enabled,
				"enabled is helm-only and must survive a UI edit of a sibling")
		})
	}
}

// ltlApplyToCommon mirrors one ltlSingleFieldEdits row directly onto the common
// config, so the expectation above is derived rather than hand-written.
func ltlApplyToCommon(t *testing.T, name string, cfg *common.OdigosConfiguration) {
	t.Helper()
	ltl := ltlStored(t, cfg)
	switch name {
	case "MaxExamplePathsPerWorkload":
		ltl.MaxExamplePathsPerWorkload = ltlIntPtr(9000)
	case "PathExampleIdleTTL":
		ltl.PathExampleIdleTTL = "72h"
	case "LearningInterval":
		ltl.LearningInterval = "45s"
	case "RuleComputation.MinObservationsForRule":
		ltl.RuleComputation.MinObservationsForRule = ltlIntPtr(250)
	case "RuleComputation.MinCardinalityForTemplating":
		ltl.RuleComputation.MinCardinalityForTemplating = ltlIntPtr(35)
	case "AutomaticRules.Enabled":
		ltl.AutomaticRules.Enabled = ltlBoolPtr(true)
	default:
		t.Fatalf("no common-side mirror for %q", name)
	}
}

// applyCardinalityControlInput returns early unless at least one of the five
// top-level input leaves is non-nil. That check is a hand-written list with no
// link to the input type, so a sixth field added to the GraphQL input would be
// accepted by the mutation, pass through every per-field guard below the check,
// and still be dropped - because the function returns before reaching them.
func TestApplyCardinalityControlInput_EveryInputFieldIsInTheNothingToDoCheck(t *testing.T) {
	ty := reflect.TypeOf(model.LocalUIConfigLiveTrafficLearningInput{})
	require.Positive(t, ty.NumField())

	covered := map[string]bool{}
	for name := range ltlSingleFieldEdits {
		for i := 0; i < ty.NumField(); i++ {
			field := ty.Field(i).Name
			if name == field || len(name) > len(field) && name[:len(field)+1] == field+"." {
				covered[field] = true
			}
		}
	}

	for i := 0; i < ty.NumField(); i++ {
		field := ty.Field(i).Name
		assert.True(t, covered[field],
			"LocalUIConfigLiveTrafficLearningInput.%s has no row in ltlSingleFieldEdits, so nothing "+
				"proves an edit that sets only this field survives the nothing-to-do check", field)
	}

	// Enabled is deliberately absent from the GraphQL input: deploying
	// odigos-cache is a helm decision. If it ever appears here, the overlay
	// writer and the merge both need a branch for it.
	_, hasEnabled := ty.FieldByName("Enabled")
	assert.False(t, hasEnabled,
		"enabled became settable from the UI; applyCardinalityControlInput now has to handle it")
}

// An input that carries the wrapper objects but no value must not allocate the
// overlay blocks. A cardinalityControl block that exists but is empty is what the
// settings screen sends whenever the user saves an unrelated page, and writing it
// would add a meaningless key to odigos-local-ui-config on every save.
func TestApplyCardinalityControlInput_NoValuesIsAnExactNoop(t *testing.T) {
	for name, input := range map[string]model.LocalUIConfigInput{
		"no cardinalityControl":     {},
		"no urlTemplatization":      {CardinalityControl: &model.LocalUIConfigCardinalityControlInput{}},
		"no liveTrafficLearning":    ltlInput(nil),
		"empty liveTrafficLearning": ltlInput(&model.LocalUIConfigLiveTrafficLearningInput{}),
	} {
		t.Run(name, func(t *testing.T) {
			fresh := &common.OdigosConfiguration{}
			applyLocalUiConfigInput(fresh, input)
			assert.Nil(t, fresh.CardinalityControl,
				"no value was supplied, so no cardinalityControl key may be written to the overlay")
		})
	}

	// An allocated-but-empty nested object does pass the nothing-to-do check, so
	// these shapes build the blocks. What matters is that they still write no
	// leaf, and in particular do not clear a stored one.
	for name, input := range map[string]model.LocalUIConfigInput{
		"empty rule computation": ltlInput(&model.LocalUIConfigLiveTrafficLearningInput{
			RuleComputation: &model.LocalUIConfigLiveTrafficLearningRuleComputationInput{},
		}),
		"empty automatic rules": ltlInput(&model.LocalUIConfigLiveTrafficLearningInput{
			AutomaticRules: &model.LocalUIConfigLiveTrafficLearningAutomaticRulesInput{},
		}),
	} {
		t.Run("stored/"+name, func(t *testing.T) {
			cfg := ltlStoredConfig()
			applyLocalUiConfigInput(cfg, input)
			assert.Equal(t, ltlStoredConfig(), cfg,
				"an empty nested input must not clear the stored values")
		})

		t.Run("fresh/"+name, func(t *testing.T) {
			fresh := &common.OdigosConfiguration{}
			applyLocalUiConfigInput(fresh, input)
			assert.Equal(t, &common.LiveTrafficLearningConfiguration{
				RuleComputation: ltlEmptyRuleComputation(input),
				AutomaticRules:  ltlEmptyAutomaticRules(input),
			}, ltlStored(t, fresh),
				"the block is created but no value may be invented for it")
		})
	}
}

// The two helpers below mirror which empty nested object the input carried, so
// the expectation above stays derived from the input rather than duplicated.
func ltlEmptyRuleComputation(in model.LocalUIConfigInput) *common.LiveTrafficLearningRuleComputationConfiguration {
	if in.CardinalityControl.URLTemplatization.LiveTrafficLearning.RuleComputation == nil {
		return nil
	}
	return &common.LiveTrafficLearningRuleComputationConfiguration{}
}

func ltlEmptyAutomaticRules(in model.LocalUIConfigInput) *common.LiveTrafficLearningAutomaticRulesConfiguration {
	if in.CardinalityControl.URLTemplatization.LiveTrafficLearning.AutomaticRules == nil {
		return nil
	}
	return &common.LiveTrafficLearningAutomaticRulesConfiguration{}
}

// Editing every setting at once has to land every one of them. This is the
// anti-vacuity pair for the per-field table: without it, a writer that ignores
// the input entirely would still satisfy the "keeps siblings" assertions.
func TestApplyCardinalityControlInput_WritesEveryFieldAtOnce(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	applyLocalUiConfigInput(cfg, ltlInput(&model.LocalUIConfigLiveTrafficLearningInput{
		MaxExamplePathsPerWorkload: ltlIntPtr(9000),
		PathExampleIdleTTL:         ltlStrPtr("72h"),
		LearningInterval:           ltlStrPtr("45s"),
		RuleComputation: &model.LocalUIConfigLiveTrafficLearningRuleComputationInput{
			MinObservationsForRule:      ltlIntPtr(250),
			MinCardinalityForTemplating: ltlIntPtr(35),
		},
		AutomaticRules: &model.LocalUIConfigLiveTrafficLearningAutomaticRulesInput{
			Enabled: ltlBoolPtr(true),
		},
	}))

	assert.Equal(t, &common.LiveTrafficLearningConfiguration{
		MaxExamplePathsPerWorkload: ltlIntPtr(9000),
		PathExampleIdleTTL:         "72h",
		LearningInterval:           "45s",
		RuleComputation: &common.LiveTrafficLearningRuleComputationConfiguration{
			MinObservationsForRule:      ltlIntPtr(250),
			MinCardinalityForTemplating: ltlIntPtr(35),
		},
		AutomaticRules: &common.LiveTrafficLearningAutomaticRulesConfiguration{
			Enabled: ltlBoolPtr(true),
		},
	}, ltlStored(t, cfg))
}

// The overlay writer must stay inside its own block. applyLocalUiConfigInput
// dispatches on a dozen sibling inputs, and cardinalityControl was appended to
// the end of that chain, where an early return would silently skip nothing but
// itself - or, written the other way round, where it could swallow a sibling.
func TestApplyCardinalityControlInput_DoesNotDisturbSiblingInputs(t *testing.T) {
	cfg := &common.OdigosConfiguration{}
	input := ltlInput(&model.LocalUIConfigLiveTrafficLearningInput{LearningInterval: ltlStrPtr("45s")})
	input.ClusterName = ltlStrPtr("from-ui")
	input.TraceCorrelations = &model.LocalUIConfigTraceCorrelationsInput{
		ServiceIo: &model.LocalUIConfigTraceCorrelationsServiceIOInput{
			MetricsFlushInterval: ltlStrPtr("21s"),
		},
	}

	applyLocalUiConfigInput(cfg, input)

	assert.Equal(t, "45s", ltlStored(t, cfg).LearningInterval)
	assert.Equal(t, "from-ui", cfg.ClusterName, "a scalar sibling must still be applied")
	require.NotNil(t, cfg.TraceCorrelations)
	require.NotNil(t, cfg.TraceCorrelations.ServiceIO)
	assert.Equal(t, "21s", cfg.TraceCorrelations.ServiceIO.MetricsFlushInterval,
		"the sibling block applied just before cardinalityControl must still be applied")
}
