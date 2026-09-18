package sampling

import (
	"context"
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A sampling rule has no stored identifier. Its id is a hash of its own content, handed to
// the UI by create/list and sent straight back on the next update or delete, where
// find*ByHash recomputes it for every stored rule to find the slot again. That makes the
// whole round trip a contract between two halves of this package: if the id the UI is
// given is not the id the lookup recomputes, every edit and delete fails with
// "rule not found" while the rule sits there in the CR.
func TestSamplingRuleIDFromCreateResolvesOnUpdateAndThenOnDelete(t *testing.T) {
	t.Run("noisyOperation", func(t *testing.T) {
		h := newSamplingRulesHarness(t)

		created, err := CreateNoisyOperationRule(context.Background(), "group", model.NoisyOperationRuleInput{
			Name:         stringPtr("health checks"),
			SourceScopes: &model.SourcesScopesInput{Namespaces: []string{"payments"}},
			Operation:    &model.HeadSamplingOperationMatcherInput{HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/healthz")}},
		})
		require.NoError(t, err)
		require.NotEmpty(t, created.RuleID)

		h.refreshCache()
		updated, err := UpdateNoisyOperationRule(context.Background(), "group", created.RuleID,
			model.NoisyOperationRuleInput{PercentageAtMost: float64Ptr(2)})
		require.NoError(t, err, "the id create handed to the UI must resolve on update")
		require.NotNil(t, updated)

		h.refreshCache()
		ok, err := DeleteNoisyOperationRule(context.Background(), "group", updated.RuleID)
		require.NoError(t, err, "the id update handed back must resolve on delete")
		assert.True(t, ok)
		assert.Empty(t, h.stored("group").Spec.NoisyOperations)
	})

	t.Run("highlyRelevantOperation", func(t *testing.T) {
		h := newSamplingRulesHarness(t)

		created, err := CreateHighlyRelevantOperationRule(context.Background(), "group", model.HighlyRelevantOperationRuleInput{
			Name:              stringPtr("keep charges"),
			SourceScopes:      &model.SourcesScopesInput{Namespaces: []string{"payments"}},
			Error:             boolPtr(true),
			DurationAtLeastMs: intPtr(500),
			Operation:         &model.TailSamplingOperationMatcherInput{HTTPServer: &model.TailSamplingHTTPServerMatcherInput{Route: stringPtr("/charge")}},
		})
		require.NoError(t, err)
		require.NotEmpty(t, created.RuleID)

		h.refreshCache()
		updated, err := UpdateHighlyRelevantOperationRule(context.Background(), "group", created.RuleID,
			model.HighlyRelevantOperationRuleInput{PercentageAtLeast: float64Ptr(90)})
		require.NoError(t, err)
		require.NotNil(t, updated)

		h.refreshCache()
		ok, err := DeleteHighlyRelevantOperationRule(context.Background(), "group", updated.RuleID)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, h.stored("group").Spec.HighlyRelevantOperations)
	})

	t.Run("costReductionRule", func(t *testing.T) {
		h := newSamplingRulesHarness(t)

		created, err := CreateCostReductionRule(context.Background(), "group", model.CostReductionRuleInput{
			Name:             stringPtr("drop kafka noise"),
			SourceScopes:     &model.SourcesScopesInput{Namespaces: []string{"orders"}},
			Operation:        &model.TailSamplingOperationMatcherInput{KafkaConsumer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("orders")}},
			PercentageAtMost: 5,
		})
		require.NoError(t, err)
		require.NotEmpty(t, created.RuleID)

		h.refreshCache()
		updated, err := UpdateCostReductionRule(context.Background(), "group", created.RuleID,
			model.CostReductionRuleInput{PercentageAtMost: 15})
		require.NoError(t, err)
		require.NotNil(t, updated)

		h.refreshCache()
		ok, err := DeleteCostReductionRule(context.Background(), "group", updated.RuleID)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, h.stored("group").Spec.CostReductionRules)
	})
}

// samplingRuleIdentityFields answers, for every field of every rule struct, whether that
// field feeds the rule id. Both answers are load-bearing:
//
//   - a field that joins the hash silently invalidates every rule id the UI is currently
//     holding, so editing that field from the UI starts failing with "rule not found";
//   - a field that stays out of the hash cannot distinguish two rules, so two rules that
//     differ only in it collide on one id and only the first stays addressable.
//
// The reflection gate below fails for a field missing from this map, which forces a
// deliberate answer whenever a rule struct grows a field.
var samplingRuleIdentityFields = map[string]map[string]bool{
	"NoisyOperation": {
		"Name":             false,
		"Disabled":         false,
		"SourceScopes":     true,
		"Operation":        true,
		"PercentageAtMost": false,
		"Notes":            false,
	},
	"HighlyRelevantOperation": {
		"Name":              false,
		"Disabled":          false,
		"SourceScopes":      true,
		"Error":             true,
		"DurationAtLeastMs": true,
		"Operation":         true,
		"PercentageAtLeast": false,
		"Notes":             false,
	},
	"CostReductionRule": {
		"Name":             false,
		"Disabled":         false,
		"SourceScopes":     true,
		"Operation":        true,
		"PercentageAtMost": false,
		"Notes":            false,
	},
}

// nonZeroSamplingFieldValue produces a distinguishable value for a rule field, so that a
// case can change exactly one field and nothing else.
func nonZeroSamplingFieldValue(t *testing.T, fieldType reflect.Type) reflect.Value {
	t.Helper()
	switch fieldType {
	case reflect.TypeOf(""):
		return reflect.ValueOf("changed")
	case reflect.TypeOf(false):
		return reflect.ValueOf(true)
	case reflect.TypeOf((*float64)(nil)):
		return reflect.ValueOf(float64Ptr(37))
	case reflect.TypeOf((*int)(nil)):
		return reflect.ValueOf(intPtr(37))
	case reflect.TypeOf(float64(0)):
		return reflect.ValueOf(float64(37))
	case reflect.TypeOf((*k8sconsts.SourcesScopes)(nil)):
		return reflect.ValueOf(namespaceScope("scoped-namespace"))
	case reflect.TypeOf((*commonapisampling.HeadSamplingOperationMatcher)(nil)):
		return reflect.ValueOf(headServerMatcher("/changed"))
	case reflect.TypeOf((*commonapisampling.TailSamplingOperationMatcher)(nil)):
		return reflect.ValueOf(tailServerMatcher("/changed"))
	}
	t.Fatalf("no distinguishable value known for rule field type %s", fieldType)
	return reflect.Value{}
}

func TestSamplingRuleIDDependsOnExactlyTheDeclaredIdentityFields(t *testing.T) {
	cases := []struct {
		structName string
		empty      any
		hash       func(any) string
	}{
		{
			structName: "NoisyOperation",
			empty:      v1alpha1.NoisyOperation{},
			hash: func(rule any) string {
				typed := rule.(v1alpha1.NoisyOperation)
				return v1alpha1.ComputeNoisyOperationHash(&typed)
			},
		},
		{
			structName: "HighlyRelevantOperation",
			empty:      v1alpha1.HighlyRelevantOperation{},
			hash: func(rule any) string {
				typed := rule.(v1alpha1.HighlyRelevantOperation)
				return v1alpha1.ComputeHighlyRelevantOperationHash(&typed)
			},
		},
		{
			structName: "CostReductionRule",
			empty:      v1alpha1.CostReductionRule{},
			hash: func(rule any) string {
				typed := rule.(v1alpha1.CostReductionRule)
				return v1alpha1.ComputeCostReductionRuleHash(&typed)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.structName, func(t *testing.T) {
			declared := samplingRuleIdentityFields[tc.structName]
			require.NotEmpty(t, declared, "no identity classification declared for %s", tc.structName)

			ruleType := reflect.TypeOf(tc.empty)
			baseline := tc.hash(tc.empty)
			identityFields := 0

			for i := 0; i < ruleType.NumField(); i++ {
				field := ruleType.Field(i)
				affectsIdentity, classified := declared[field.Name]
				require.True(t, classified,
					"%s.%s is not classified in samplingRuleIdentityFields: decide whether it belongs to the rule id",
					tc.structName, field.Name)

				mutated := reflect.New(ruleType).Elem()
				mutated.Field(i).Set(nonZeroSamplingFieldValue(t, field.Type))
				changed := tc.hash(mutated.Interface()) != baseline

				assert.Equal(t, affectsIdentity, changed,
					"%s.%s: expected affectsRuleId=%v but changing it alone %s the rule id",
					tc.structName, field.Name, affectsIdentity,
					map[bool]string{true: "changed", false: "left"}[changed])
				if affectsIdentity {
					identityFields++
				}
			}

			assert.Equal(t, ruleType.NumField(), len(declared),
				"samplingRuleIdentityFields lists a field %s no longer has", tc.structName)
			assert.Positive(t, identityFields, "a rule id derived from no field at all would collide for every rule")
		})
	}
}

// A rename is the edit users make most often, and the UI has to keep working right after
// it. Because the name is deliberately outside the hash, the id survives - so the same id
// must still resolve on the next mutation.
func TestRenamingASamplingRuleKeepsItsIDUsable(t *testing.T) {
	existing := storedNoisyOperation("original", "/healthz")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	ruleID := v1alpha1.ComputeNoisyOperationHash(&existing)

	renamed, err := UpdateNoisyOperationRule(context.Background(), "group", ruleID,
		model.NoisyOperationRuleInput{Name: stringPtr("renamed")})
	require.NoError(t, err)
	assert.Equal(t, ruleID, renamed.RuleID, "a rename must not invalidate the rule id")

	h.refreshCache()
	ok, err := DeleteNoisyOperationRule(context.Background(), "group", ruleID)
	require.NoError(t, err)
	assert.True(t, ok)
}

// Editing the matcher does change the id, and the new id is only discoverable from the
// mutation response. A UI that kept the old one would address a rule that no longer
// exists, so the response has to carry the new id rather than echo the requested one.
func TestEditingTheMatcherReturnsTheNewSamplingRuleID(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/healthz")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	oldID := v1alpha1.ComputeNoisyOperationHash(&existing)

	updated, err := UpdateNoisyOperationRule(context.Background(), "group", oldID,
		model.NoisyOperationRuleInput{Operation: &model.HeadSamplingOperationMatcherInput{
			HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/livez")},
		}})
	require.NoError(t, err)

	assert.NotEqual(t, oldID, updated.RuleID, "a matcher edit produces a different rule id")
	stored := h.stored("group").Spec.NoisyOperations[0]
	assert.Equal(t, v1alpha1.ComputeNoisyOperationHash(&stored), updated.RuleID,
		"the returned id must be the id of the rule as stored, or the next edit cannot find it")

	h.refreshCache()
	_, err = UpdateNoisyOperationRule(context.Background(), "group", oldID, model.NoisyOperationRuleInput{})
	assert.Error(t, err, "the superseded id must no longer resolve")
}

// find*ByHash walks the whole slice, so a rule id must resolve wherever the rule sits.
// A fixture with the target first would pass even for a lookup that only ever checks
// index 0.
func TestASamplingRuleIDResolvesFromAnyPositionInTheGroup(t *testing.T) {
	target := storedNoisyOperation("target", "/target")
	rules := []v1alpha1.NoisyOperation{
		storedNoisyOperation("first", "/first"),
		storedNoisyOperation("second", "/second"),
		target,
	}

	assert.Equal(t, 2, findNoisyOperationByHash(rules, v1alpha1.ComputeNoisyOperationHash(&target)))
	assert.Equal(t, 0, findNoisyOperationByHash(rules, v1alpha1.ComputeNoisyOperationHash(&rules[0])))
	assert.Equal(t, -1, findNoisyOperationByHash(rules, "deadbeef"),
		"an unknown id must be reported as absent, not resolved to the first rule")
	assert.Equal(t, -1, findNoisyOperationByHash(nil, "deadbeef"))
}

func TestAHighlyRelevantOperationIDResolvesFromAnyPositionInTheGroup(t *testing.T) {
	target := storedHighlyRelevantOperation("target", "/target")
	rules := []v1alpha1.HighlyRelevantOperation{
		storedHighlyRelevantOperation("first", "/first"),
		target,
	}

	assert.Equal(t, 1, findHighlyRelevantOperationByHash(rules, v1alpha1.ComputeHighlyRelevantOperationHash(&target)))
	assert.Equal(t, -1, findHighlyRelevantOperationByHash(rules, "deadbeef"))
	assert.Equal(t, -1, findHighlyRelevantOperationByHash(nil, "deadbeef"))
}

func TestACostReductionRuleIDResolvesFromAnyPositionInTheGroup(t *testing.T) {
	target := storedCostReductionRule("target", "/target")
	rules := []v1alpha1.CostReductionRule{
		storedCostReductionRule("first", "/first"),
		target,
	}

	assert.Equal(t, 1, findCostReductionRuleByHash(rules, v1alpha1.ComputeCostReductionRuleHash(&target)))
	assert.Equal(t, -1, findCostReductionRuleByHash(rules, "deadbeef"))
	assert.Equal(t, -1, findCostReductionRuleByHash(nil, "deadbeef"))
}

// The three rule families are addressed by three separate hash functions over three
// separate slices, and every mutation names both the family and the slice by hand. A
// mutation that reached into a sibling family would corrupt unrelated sampling rules
// without any error, so each entry point is checked against the two slices it must leave
// exactly as it found them.
func TestASamplingRuleMutationNeverTouchesTheOtherRuleFamilies(t *testing.T) {
	noisy := storedNoisyOperation("noisy", "/noisy")
	relevant := storedHighlyRelevantOperation("relevant", "/relevant")
	cost := storedCostReductionRule("cost", "/cost")

	seed := func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{noisy}
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{relevant}
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{cost}
	}

	mutations := map[string]struct {
		family string
		run    func(context.Context) error
	}{
		"createNoisyOperation": {"noisy", func(ctx context.Context) error {
			_, err := CreateNoisyOperationRule(ctx, "group", model.NoisyOperationRuleInput{Name: stringPtr("added")})
			return err
		}},
		"updateNoisyOperation": {"noisy", func(ctx context.Context) error {
			_, err := UpdateNoisyOperationRule(ctx, "group", v1alpha1.ComputeNoisyOperationHash(&noisy),
				model.NoisyOperationRuleInput{Name: stringPtr("renamed")})
			return err
		}},
		"deleteNoisyOperation": {"noisy", func(ctx context.Context) error {
			_, err := DeleteNoisyOperationRule(ctx, "group", v1alpha1.ComputeNoisyOperationHash(&noisy))
			return err
		}},
		"createHighlyRelevantOperation": {"relevant", func(ctx context.Context) error {
			_, err := CreateHighlyRelevantOperationRule(ctx, "group", model.HighlyRelevantOperationRuleInput{Name: stringPtr("added")})
			return err
		}},
		"updateHighlyRelevantOperation": {"relevant", func(ctx context.Context) error {
			_, err := UpdateHighlyRelevantOperationRule(ctx, "group", v1alpha1.ComputeHighlyRelevantOperationHash(&relevant),
				model.HighlyRelevantOperationRuleInput{Name: stringPtr("renamed")})
			return err
		}},
		"deleteHighlyRelevantOperation": {"relevant", func(ctx context.Context) error {
			_, err := DeleteHighlyRelevantOperationRule(ctx, "group", v1alpha1.ComputeHighlyRelevantOperationHash(&relevant))
			return err
		}},
		"createCostReductionRule": {"cost", func(ctx context.Context) error {
			_, err := CreateCostReductionRule(ctx, "group", model.CostReductionRuleInput{Name: stringPtr("added")})
			return err
		}},
		"updateCostReductionRule": {"cost", func(ctx context.Context) error {
			_, err := UpdateCostReductionRule(ctx, "group", v1alpha1.ComputeCostReductionRuleHash(&cost),
				model.CostReductionRuleInput{Name: stringPtr("renamed")})
			return err
		}},
		"deleteCostReductionRule": {"cost", func(ctx context.Context) error {
			_, err := DeleteCostReductionRule(ctx, "group", v1alpha1.ComputeCostReductionRuleHash(&cost))
			return err
		}},
	}

	for name, tc := range mutations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t, samplingCR("group", seed))

			require.NoError(t, tc.run(context.Background()))

			stored := h.stored("group").Spec
			if tc.family != "noisy" {
				assert.Equal(t, []v1alpha1.NoisyOperation{noisy}, stored.NoisyOperations,
					"noisy operations must be untouched by a %s mutation", tc.family)
			}
			if tc.family != "relevant" {
				assert.Equal(t, []v1alpha1.HighlyRelevantOperation{relevant}, stored.HighlyRelevantOperations,
					"highly relevant operations must be untouched by a %s mutation", tc.family)
			}
			if tc.family != "cost" {
				assert.Equal(t, []v1alpha1.CostReductionRule{cost}, stored.CostReductionRules,
					"cost reduction rules must be untouched by a %s mutation", tc.family)
			}
			assert.Equal(t, "group", stored.Name, "the group's own fields must be preserved")
		})
	}
}

// Characterisation of a known defect, reported in the PR that adds this test and not fixed
// here: because the rule id covers only the scope and the matcher, two rules of the same
// family that share both get the same id. find*ByHash returns the first match, so the
// second rule can never be edited or deleted from the UI - a mutation aimed at it silently
// rewrites the first one instead. Nothing rejects the duplicate at create time.
func TestTwoSamplingRulesWithTheSameScopeAndMatcherAreNotIndividuallyAddressable(t *testing.T) {
	first := v1alpha1.NoisyOperation{
		Name:             "first",
		Operation:        headServerMatcher("/healthz"),
		PercentageAtMost: float64Ptr(1),
	}
	second := v1alpha1.NoisyOperation{
		Name:             "second",
		Operation:        headServerMatcher("/healthz"),
		PercentageAtMost: float64Ptr(50),
	}
	require.Equal(t, v1alpha1.ComputeNoisyOperationHash(&first), v1alpha1.ComputeNoisyOperationHash(&second),
		"two rules with the same scope and matcher share an id by construction")

	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{first, second}
	}))

	_, err := UpdateNoisyOperationRule(context.Background(), "group",
		v1alpha1.ComputeNoisyOperationHash(&second), model.NoisyOperationRuleInput{Name: stringPtr("renamed")})
	require.NoError(t, err)

	assert.Equal(t, []string{"renamed", "second"},
		noisyOperationNames(h.stored("group").Spec.NoisyOperations),
		"the edit lands on the first rule sharing the id, leaving the intended rule unreachable")
}

// Notes are documentation, so they stay out of the id: annotating a rule must not break
// the id the UI is holding for it.
func TestAnnotatingASamplingRuleKeepsItsID(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/healthz")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	ruleID := v1alpha1.ComputeNoisyOperationHash(&existing)

	updated, err := UpdateNoisyOperationRule(context.Background(), "group", ruleID,
		model.NoisyOperationRuleInput{Notes: stringPtr("raised by the payments team")})
	require.NoError(t, err)

	assert.Equal(t, ruleID, updated.RuleID)
	assert.Equal(t, "raised by the payments team", h.stored("group").Spec.NoisyOperations[0].Notes)
}
