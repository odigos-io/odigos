package sampling

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

func TestCreateNoisyOperationRuleCreatesTheSamplingCRWhenItDoesNotExist(t *testing.T) {
	h := newSamplingRulesHarness(t)

	rule, err := CreateNoisyOperationRule(context.Background(), "default-sampling", model.NoisyOperationRuleInput{
		Name:             stringPtr("health checks"),
		Operation:        &model.HeadSamplingOperationMatcherInput{HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/healthz")}},
		PercentageAtMost: float64Ptr(1),
	})
	require.NoError(t, err)
	require.NotNil(t, rule)

	assert.Equal(t, []string{"default-sampling"}, h.storedNames())
	stored := h.stored("default-sampling")
	assert.Equal(t, samplingRulesNamespace, stored.Namespace,
		"the sampling CR must be created in the odigos namespace")
	assert.Equal(t, "default-sampling", stored.Spec.Name,
		"a CR created on demand names itself after the sampling id the UI asked for")
	require.Len(t, stored.Spec.NoisyOperations, 1)
	assert.Equal(t, "health checks", stored.Spec.NoisyOperations[0].Name)
	assert.Equal(t, float64Ptr(1), stored.Spec.NoisyOperations[0].PercentageAtMost)
	require.NotNil(t, stored.Spec.NoisyOperations[0].Operation)
	require.NotNil(t, stored.Spec.NoisyOperations[0].Operation.HttpServer)
	assert.Equal(t, "/healthz", stored.Spec.NoisyOperations[0].Operation.HttpServer.Route)
}

func TestCreateNoisyOperationRuleAppendsAfterTheExistingRules(t *testing.T) {
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{
			storedNoisyOperation("first", "/one"),
			storedNoisyOperation("second", "/two"),
		}
	}))

	_, err := CreateNoisyOperationRule(context.Background(), "group", model.NoisyOperationRuleInput{
		Name:      stringPtr("third"),
		Operation: &model.HeadSamplingOperationMatcherInput{HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/three")}},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"first", "second", "third"},
		noisyOperationNames(h.stored("group").Spec.NoisyOperations),
		"a new rule is appended; the existing rules keep their order")
}

func TestCreateHighlyRelevantOperationRuleAppendsAfterTheExistingRules(t *testing.T) {
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{
			storedHighlyRelevantOperation("first", "/one"),
		}
	}))

	rule, err := CreateHighlyRelevantOperationRule(context.Background(), "group", model.HighlyRelevantOperationRuleInput{
		Name:              stringPtr("second"),
		Error:             boolPtr(true),
		DurationAtLeastMs: intPtr(250),
		Operation:         &model.TailSamplingOperationMatcherInput{HTTPServer: &model.TailSamplingHTTPServerMatcherInput{Route: stringPtr("/two")}},
		PercentageAtLeast: float64Ptr(100),
	})
	require.NoError(t, err)
	require.NotNil(t, rule)

	stored := h.stored("group")
	assert.Equal(t, []string{"first", "second"}, highlyRelevantOperationNames(stored.Spec.HighlyRelevantOperations))
	created := stored.Spec.HighlyRelevantOperations[1]
	assert.True(t, created.Error)
	assert.Equal(t, intPtr(250), created.DurationAtLeastMs)
	assert.Equal(t, float64Ptr(100), created.PercentageAtLeast)
}

func TestCreateCostReductionRuleAppendsAfterTheExistingRules(t *testing.T) {
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{
			storedCostReductionRule("first", "/one"),
		}
	}))

	rule, err := CreateCostReductionRule(context.Background(), "group", model.CostReductionRuleInput{
		Name:             stringPtr("second"),
		Operation:        &model.TailSamplingOperationMatcherInput{KafkaConsumer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("orders")}},
		PercentageAtMost: 25,
	})
	require.NoError(t, err)
	require.NotNil(t, rule)

	stored := h.stored("group")
	assert.Equal(t, []string{"first", "second"}, costReductionRuleNames(stored.Spec.CostReductionRules))
	created := stored.Spec.CostReductionRules[1]
	assert.Equal(t, 25.0, created.PercentageAtMost)
	require.NotNil(t, created.Operation)
	require.NotNil(t, created.Operation.KafkaConsumer)
	assert.Equal(t, "orders", created.Operation.KafkaConsumer.KafkaTopic)
}

// TestUpdateNoisyOperationRuleAppliesTheMergeInPlace pins that an update rewrites the
// targeted slot rather than appending a second copy of the rule.
func TestUpdateNoisyOperationRuleAppliesTheMergeInPlace(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/healthz")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{
			storedNoisyOperation("other", "/other"),
			existing,
		}
	}))

	rule, err := UpdateNoisyOperationRule(context.Background(), "group",
		v1alpha1.ComputeNoisyOperationHash(&existing),
		model.NoisyOperationRuleInput{Name: stringPtr("renamed"), PercentageAtMost: float64Ptr(50)})
	require.NoError(t, err)
	require.NotNil(t, rule)

	stored := h.stored("group").Spec.NoisyOperations
	require.Len(t, stored, 2, "an update must not append a rule")
	assert.Equal(t, []string{"other", "renamed"}, noisyOperationNames(stored),
		"the update lands in the slot the rule id resolved to")
	assert.Equal(t, float64Ptr(50), stored[1].PercentageAtMost)
	assert.Equal(t, existing.Operation, stored[1].Operation,
		"an omitted operation is preserved rather than cleared")
}

func TestUpdateHighlyRelevantOperationRuleAppliesTheMergeInPlace(t *testing.T) {
	existing := storedHighlyRelevantOperation("relevant", "/charge")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{
			storedHighlyRelevantOperation("other", "/other"),
			existing,
		}
	}))

	_, err := UpdateHighlyRelevantOperationRule(context.Background(), "group",
		v1alpha1.ComputeHighlyRelevantOperationHash(&existing),
		model.HighlyRelevantOperationRuleInput{Name: stringPtr("renamed")})
	require.NoError(t, err)

	stored := h.stored("group").Spec.HighlyRelevantOperations
	require.Len(t, stored, 2)
	assert.Equal(t, []string{"other", "renamed"}, highlyRelevantOperationNames(stored))
	assert.Equal(t, existing.Operation, stored[1].Operation)
}

func TestUpdateCostReductionRuleAppliesTheMergeInPlace(t *testing.T) {
	existing := storedCostReductionRule("cost", "/checkout")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{
			storedCostReductionRule("other", "/other"),
			existing,
		}
	}))

	_, err := UpdateCostReductionRule(context.Background(), "group",
		v1alpha1.ComputeCostReductionRuleHash(&existing),
		model.CostReductionRuleInput{Name: stringPtr("renamed"), PercentageAtMost: 42})
	require.NoError(t, err)

	stored := h.stored("group").Spec.CostReductionRules
	require.Len(t, stored, 2)
	assert.Equal(t, []string{"other", "renamed"}, costReductionRuleNames(stored))
	assert.Equal(t, 42.0, stored[1].PercentageAtMost)
	assert.Equal(t, existing.Operation, stored[1].Operation)
}

// The three delete paths all splice the target out with
// append(rules[:idx], rules[idx+1:]...). A three-rule fixture with the target in the
// middle is the smallest one that can tell "removed the right element" apart from
// "removed a neighbour" and from "reordered the rest".
func TestDeleteNoisyOperationRuleRemovesOnlyTheTargetedRule(t *testing.T) {
	target := storedNoisyOperation("target", "/target")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{
			storedNoisyOperation("before", "/before"),
			target,
			storedNoisyOperation("after", "/after"),
		}
	}))

	ok, err := DeleteNoisyOperationRule(context.Background(), "group", v1alpha1.ComputeNoisyOperationHash(&target))
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, []string{"before", "after"},
		noisyOperationNames(h.stored("group").Spec.NoisyOperations))
}

func TestDeleteHighlyRelevantOperationRuleRemovesOnlyTheTargetedRule(t *testing.T) {
	target := storedHighlyRelevantOperation("target", "/target")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{
			storedHighlyRelevantOperation("before", "/before"),
			target,
			storedHighlyRelevantOperation("after", "/after"),
		}
	}))

	ok, err := DeleteHighlyRelevantOperationRule(context.Background(), "group",
		v1alpha1.ComputeHighlyRelevantOperationHash(&target))
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, []string{"before", "after"},
		highlyRelevantOperationNames(h.stored("group").Spec.HighlyRelevantOperations))
}

func TestDeleteCostReductionRuleRemovesOnlyTheTargetedRule(t *testing.T) {
	target := storedCostReductionRule("target", "/target")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{
			storedCostReductionRule("before", "/before"),
			target,
			storedCostReductionRule("after", "/after"),
		}
	}))

	ok, err := DeleteCostReductionRule(context.Background(), "group",
		v1alpha1.ComputeCostReductionRuleHash(&target))
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, []string{"before", "after"},
		costReductionRuleNames(h.stored("group").Spec.CostReductionRules))
}

// An unknown rule id must be reported and must not reach the API server: the alternative
// is a silent no-op update, or worse, a write of a half-applied spec.
func TestSamplingRuleMutationsRejectAnUnknownRuleIDWithoutWriting(t *testing.T) {
	seed := func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{storedNoisyOperation("noisy", "/noisy")}
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{storedHighlyRelevantOperation("relevant", "/relevant")}
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{storedCostReductionRule("cost", "/cost")}
	}

	// The message is asserted as one ordered phrase rather than as two separate
	// substrings: an operator reading "rule group not found in sampling deadbeef" would
	// go looking for the wrong object, and two Contains assertions cannot tell the two
	// arguments apart.
	mutations := map[string]struct {
		run         func(context.Context) error
		wantMessage string
	}{
		"updateNoisyOperation": {
			run: func(ctx context.Context) error {
				_, err := UpdateNoisyOperationRule(ctx, "group", "deadbeef", model.NoisyOperationRuleInput{Name: stringPtr("x")})
				return err
			},
			wantMessage: "noisy operation rule deadbeef not found in sampling group",
		},
		"deleteNoisyOperation": {
			run: func(ctx context.Context) error {
				_, err := DeleteNoisyOperationRule(ctx, "group", "deadbeef")
				return err
			},
			wantMessage: "noisy operation rule deadbeef not found in sampling group",
		},
		"updateHighlyRelevantOperation": {
			run: func(ctx context.Context) error {
				_, err := UpdateHighlyRelevantOperationRule(ctx, "group", "deadbeef", model.HighlyRelevantOperationRuleInput{Name: stringPtr("x")})
				return err
			},
			wantMessage: "highly relevant operation rule deadbeef not found in sampling group",
		},
		"deleteHighlyRelevantOperation": {
			run: func(ctx context.Context) error {
				_, err := DeleteHighlyRelevantOperationRule(ctx, "group", "deadbeef")
				return err
			},
			wantMessage: "highly relevant operation rule deadbeef not found in sampling group",
		},
		"updateCostReductionRule": {
			run: func(ctx context.Context) error {
				_, err := UpdateCostReductionRule(ctx, "group", "deadbeef", model.CostReductionRuleInput{Name: stringPtr("x")})
				return err
			},
			wantMessage: "cost reduction rule deadbeef not found in sampling group",
		},
		"deleteCostReductionRule": {
			run: func(ctx context.Context) error {
				_, err := DeleteCostReductionRule(ctx, "group", "deadbeef")
				return err
			},
			wantMessage: "cost reduction rule deadbeef not found in sampling group",
		},
	}

	for name, tc := range mutations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t, samplingCR("group", seed))
			writes := h.countLiveWrites()

			err := tc.run(context.Background())

			require.Error(t, err)
			assert.ErrorContains(t, err, tc.wantMessage)
			assert.Zero(t, *writes, "a rejected mutation must not write to the API server")
		})
	}
}

// Each delete returns a bool alongside its error and the UI drops the rule from its list
// on a true. All three must report false when the delete failed, or the rule disappears
// from the screen while the collector keeps applying it.
func TestEverySamplingRuleDeleteReportsFailureRatherThanSuccess(t *testing.T) {
	seed := func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{storedNoisyOperation("noisy", "/noisy")}
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{storedHighlyRelevantOperation("relevant", "/relevant")}
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{storedCostReductionRule("cost", "/cost")}
	}

	deletes := map[string]func(context.Context) (bool, error){
		"noisyOperation": func(ctx context.Context) (bool, error) {
			return DeleteNoisyOperationRule(ctx, "group", "deadbeef")
		},
		"highlyRelevantOperation": func(ctx context.Context) (bool, error) {
			return DeleteHighlyRelevantOperationRule(ctx, "group", "deadbeef")
		},
		"costReductionRule": func(ctx context.Context) (bool, error) {
			return DeleteCostReductionRule(ctx, "group", "deadbeef")
		},
	}

	for name, remove := range deletes {
		t.Run(name, func(t *testing.T) {
			newSamplingRulesHarness(t, samplingCR("group", seed))

			ok, err := remove(context.Background())

			require.Error(t, err)
			assert.False(t, ok, "the boolean result must not claim success when the delete failed")
		})
	}
}

// A delete that the API server rejects must report failure too, not just one whose rule
// id does not resolve.
func TestEverySamplingRuleDeleteReportsAFailedWriteAsFailure(t *testing.T) {
	noisy := storedNoisyOperation("noisy", "/noisy")
	relevant := storedHighlyRelevantOperation("relevant", "/relevant")
	cost := storedCostReductionRule("cost", "/cost")
	seed := func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{noisy}
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{relevant}
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{cost}
	}

	deletes := map[string]func(context.Context) (bool, error){
		"noisyOperation": func(ctx context.Context) (bool, error) {
			return DeleteNoisyOperationRule(ctx, "group", v1alpha1.ComputeNoisyOperationHash(&noisy))
		},
		"highlyRelevantOperation": func(ctx context.Context) (bool, error) {
			return DeleteHighlyRelevantOperationRule(ctx, "group", v1alpha1.ComputeHighlyRelevantOperationHash(&relevant))
		},
		"costReductionRule": func(ctx context.Context) (bool, error) {
			return DeleteCostReductionRule(ctx, "group", v1alpha1.ComputeCostReductionRuleHash(&cost))
		},
	}

	for name, remove := range deletes {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t, samplingCR("group", seed))
			h.live.PrependReactor("update", "samplings", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewInternalError(assert.AnError)
			})

			ok, err := remove(context.Background())

			require.Error(t, err)
			assert.False(t, ok, "a rejected write must not be reported as a successful delete")
		})
	}
}

// Every mutation resolves its sampling group first; a missing group has to surface as an
// error naming the id, not as a silently created empty group.
func TestSamplingRuleMutationsFailWhenTheSamplingGroupIsMissing(t *testing.T) {
	mutations := map[string]func(context.Context) error{
		"updateNoisyOperation": func(ctx context.Context) error {
			_, err := UpdateNoisyOperationRule(ctx, "missing", "id", model.NoisyOperationRuleInput{})
			return err
		},
		"deleteNoisyOperation": func(ctx context.Context) error {
			_, err := DeleteNoisyOperationRule(ctx, "missing", "id")
			return err
		},
		"updateHighlyRelevantOperation": func(ctx context.Context) error {
			_, err := UpdateHighlyRelevantOperationRule(ctx, "missing", "id", model.HighlyRelevantOperationRuleInput{})
			return err
		},
		"deleteHighlyRelevantOperation": func(ctx context.Context) error {
			_, err := DeleteHighlyRelevantOperationRule(ctx, "missing", "id")
			return err
		},
		"updateCostReductionRule": func(ctx context.Context) error {
			_, err := UpdateCostReductionRule(ctx, "missing", "id", model.CostReductionRuleInput{})
			return err
		},
		"deleteCostReductionRule": func(ctx context.Context) error {
			_, err := DeleteCostReductionRule(ctx, "missing", "id")
			return err
		},
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t, samplingCR("other-group"))
			writes := h.countLiveWrites()

			err := mutate(context.Background())

			require.Error(t, err)
			assert.ErrorContains(t, err, `sampling CR "missing" not found`)
			assert.Zero(t, *writes)
			assert.Equal(t, []string{"other-group"}, h.storedNames(),
				"resolving a missing group must not create one")
		})
	}
}

// Creating into a group that does not exist yet is the one path allowed to create it.
func TestCreateSamplingRuleCreatesTheGroupForEveryRuleFamily(t *testing.T) {
	creations := map[string]func(context.Context) error{
		"noisyOperation": func(ctx context.Context) error {
			_, err := CreateNoisyOperationRule(ctx, "fresh", model.NoisyOperationRuleInput{Name: stringPtr("rule")})
			return err
		},
		"highlyRelevantOperation": func(ctx context.Context) error {
			_, err := CreateHighlyRelevantOperationRule(ctx, "fresh", model.HighlyRelevantOperationRuleInput{Name: stringPtr("rule")})
			return err
		},
		"costReductionRule": func(ctx context.Context) error {
			_, err := CreateCostReductionRule(ctx, "fresh", model.CostReductionRuleInput{Name: stringPtr("rule")})
			return err
		},
	}

	for name, create := range creations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t)
			require.NoError(t, create(context.Background()))
			assert.Equal(t, []string{"fresh"}, h.storedNames())
		})
	}
}

// TestCreateSamplingRuleRecoversWhenTheInformerCacheLagsALiveCreate covers the comment in
// getOrCreateSamplingCR: several mutations fired back-to-back (the onboarding flow) race
// the informer, so the cache-backed read reports NotFound for a group that already
// exists. The live Create then fails with AlreadyExists and the code has to recover by
// reading the group from the API server instead of failing the mutation.
func TestCreateSamplingRuleRecoversWhenTheInformerCacheLagsALiveCreate(t *testing.T) {
	alreadyCreated := samplingCR("racing", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{storedNoisyOperation("first", "/first")}
	})
	h := newSamplingRules(t, samplingRulesFixture{
		cached: []*v1alpha1.Sampling{},
		live:   []*v1alpha1.Sampling{alreadyCreated},
	})

	_, err := CreateNoisyOperationRule(context.Background(), "racing", model.NoisyOperationRuleInput{
		Name:      stringPtr("second"),
		Operation: &model.HeadSamplingOperationMatcherInput{HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/second")}},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"racing"}, h.storedNames(), "the group must not be duplicated")
	assert.Equal(t, []string{"first", "second"},
		noisyOperationNames(h.stored("racing").Spec.NoisyOperations),
		"the rule already stored on the API server must survive the recovery")
}

func TestCreateSamplingRuleSurfacesAFailedGroupCreate(t *testing.T) {
	h := newSamplingRulesHarness(t)
	h.live.PrependReactor("create", "samplings", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(v1alpha1.Resource("samplings"), "blocked", assert.AnError)
	})

	_, err := CreateNoisyOperationRule(context.Background(), "blocked", model.NoisyOperationRuleInput{})

	require.Error(t, err)
	assert.ErrorContains(t, err, `failed to create sampling CR "blocked"`)
}

func TestCreateSamplingRuleSurfacesAFailedReadAfterAlreadyExists(t *testing.T) {
	h := newSamplingRules(t, samplingRulesFixture{
		cached: []*v1alpha1.Sampling{},
		live:   []*v1alpha1.Sampling{samplingCR("racing")},
	})
	h.live.PrependReactor("get", "samplings", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(assert.AnError)
	})

	_, err := CreateNoisyOperationRule(context.Background(), "racing", model.NoisyOperationRuleInput{})

	require.Error(t, err)
	assert.ErrorContains(t, err, `failed to fetch existing sampling CR "racing" after AlreadyExists`)
}

// A cache read that fails for a reason other than NotFound must be surfaced as-is. Trying
// to create the group instead would mask an RBAC or cache-sync problem behind a confusing
// AlreadyExists path.
func TestCreateSamplingRulePropagatesANonNotFoundCacheError(t *testing.T) {
	h := newSamplingRules(t, samplingRulesFixture{
		cacheFuncs: interceptor.Funcs{
			Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
				return apierrors.NewForbidden(v1alpha1.Resource("samplings"), "nope", assert.AnError)
			},
		},
	})
	writes := h.countLiveWrites()

	_, err := CreateNoisyOperationRule(context.Background(), "group", model.NoisyOperationRuleInput{})

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err), "the original cache error must reach the caller: %v", err)
	assert.NotContains(t, err.Error(), "failed to create sampling CR",
		"a forbidden read must not be retried as a create")
	assert.Zero(t, *writes)
}

func TestUpdateSamplingRuleSurfacesAFailedWrite(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/noisy")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	h.live.PrependReactor("update", "samplings", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(assert.AnError)
	})

	rule, err := UpdateNoisyOperationRule(context.Background(), "group",
		v1alpha1.ComputeNoisyOperationHash(&existing),
		model.NoisyOperationRuleInput{Name: stringPtr("renamed")})

	require.Error(t, err)
	assert.Nil(t, rule, "no rule may be returned when the write failed")
}

// Two UI tabs editing the same sampling group collide on resourceVersion. The mutation has
// to absorb that and re-apply rather than bubbling a Conflict up to the user.
func TestUpdateSamplingRuleRetriesAConflictingWrite(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/noisy")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	attempts := h.conflictOnFirstUpdates(1)

	_, err := UpdateNoisyOperationRule(context.Background(), "group",
		v1alpha1.ComputeNoisyOperationHash(&existing),
		model.NoisyOperationRuleInput{Name: stringPtr("renamed")})
	require.NoError(t, err)

	assert.Equal(t, 2, *attempts, "the conflicting write must be retried")
	assert.Equal(t, []string{"renamed"}, noisyOperationNames(h.stored("group").Spec.NoisyOperations))
}

func TestUpdateSamplingRuleSurfacesAPersistentConflict(t *testing.T) {
	existing := storedNoisyOperation("noisy", "/noisy")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{existing}
	}))
	attempts := h.conflictOnFirstUpdates(100)

	_, err := UpdateNoisyOperationRule(context.Background(), "group",
		v1alpha1.ComputeNoisyOperationHash(&existing),
		model.NoisyOperationRuleInput{Name: stringPtr("renamed")})

	require.Error(t, err)
	assert.True(t, apierrors.IsConflict(err), "a conflict that outlives the retries must reach the caller: %v", err)
	assert.Greater(t, *attempts, 1, "the retry budget must actually be spent")
	assert.Equal(t, []string{"noisy"}, noisyOperationNames(h.stored("group").Spec.NoisyOperations),
		"nothing may be persisted when every attempt conflicts")
}

func TestDeleteSamplingRuleRetriesAConflictingWrite(t *testing.T) {
	target := storedNoisyOperation("target", "/target")
	h := newSamplingRulesHarness(t, samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{target, storedNoisyOperation("keep", "/keep")}
	}))
	attempts := h.conflictOnFirstUpdates(1)

	ok, err := DeleteNoisyOperationRule(context.Background(), "group", v1alpha1.ComputeNoisyOperationHash(&target))
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, 2, *attempts)
	assert.Equal(t, []string{"keep"}, noisyOperationNames(h.stored("group").Spec.NoisyOperations),
		"the retry must not delete a second rule")
}

func TestGetSamplingCRByIDReadsFromTheOdigosNamespaceOnly(t *testing.T) {
	sameNameElsewhere := samplingCR("group", func(cr *v1alpha1.Sampling) {
		cr.Namespace = "somewhere-else"
		cr.Spec.Notes = "must not be read"
	})
	h := newSamplingRules(t, samplingRulesFixture{
		cached: []*v1alpha1.Sampling{samplingCR("group"), sameNameElsewhere},
	})

	cr, err := getSamplingCRByID(context.Background(), "group")
	require.NoError(t, err)
	assert.Equal(t, h.ns, cr.Namespace)
	assert.Empty(t, cr.Spec.Notes, "a same-named group in another namespace must not be read")
}

func TestGetSamplingCRByIDNamesTheMissingGroupAndKeepsTheNotFoundReason(t *testing.T) {
	newSamplingRulesHarness(t)

	_, err := getSamplingCRByID(context.Background(), "nope")

	require.Error(t, err)
	assert.ErrorContains(t, err, `sampling CR "nope" not found`)
	assert.True(t, apierrors.IsNotFound(err),
		"getOrCreateSamplingCR branches on IsNotFound, so wrapping must keep the reason readable")
}

func TestUpdateSamplingCRWritesToTheOdigosNamespace(t *testing.T) {
	h := newSamplingRulesHarness(t, samplingCR("group"))

	cr := h.stored("group")
	cr.Spec.Notes = "written"
	updated, err := updateSamplingCR(context.Background(), cr)
	require.NoError(t, err)
	assert.Equal(t, "written", updated.Spec.Notes)
	assert.Equal(t, "written", h.stored("group").Spec.Notes)
}

func TestUpdateSamplingCRSurfacesTheWriteError(t *testing.T) {
	h := newSamplingRulesHarness(t)

	_, err := updateSamplingCR(context.Background(), samplingCR("missing"))

	require.Error(t, err)
	assert.Empty(t, h.storedNames(), "a failed update must not create the group")
}

// Every mutation persists through the same updateSamplingCR call. A write error has to
// propagate out of each of them: swallowing it would report success to the UI for a rule
// that was never stored, and the UI would then show a rule the collector never applies.
func TestEverySamplingRuleMutationPropagatesAWriteError(t *testing.T) {
	noisy := storedNoisyOperation("noisy", "/noisy")
	relevant := storedHighlyRelevantOperation("relevant", "/relevant")
	cost := storedCostReductionRule("cost", "/cost")

	seed := func(cr *v1alpha1.Sampling) {
		cr.Spec.NoisyOperations = []v1alpha1.NoisyOperation{noisy}
		cr.Spec.HighlyRelevantOperations = []v1alpha1.HighlyRelevantOperation{relevant}
		cr.Spec.CostReductionRules = []v1alpha1.CostReductionRule{cost}
	}

	mutations := map[string]func(context.Context) error{
		"createNoisyOperation": func(ctx context.Context) error {
			_, err := CreateNoisyOperationRule(ctx, "group", model.NoisyOperationRuleInput{Name: stringPtr("added")})
			return err
		},
		"updateNoisyOperation": func(ctx context.Context) error {
			_, err := UpdateNoisyOperationRule(ctx, "group", v1alpha1.ComputeNoisyOperationHash(&noisy),
				model.NoisyOperationRuleInput{Name: stringPtr("renamed")})
			return err
		},
		"deleteNoisyOperation": func(ctx context.Context) error {
			_, err := DeleteNoisyOperationRule(ctx, "group", v1alpha1.ComputeNoisyOperationHash(&noisy))
			return err
		},
		"createHighlyRelevantOperation": func(ctx context.Context) error {
			_, err := CreateHighlyRelevantOperationRule(ctx, "group", model.HighlyRelevantOperationRuleInput{Name: stringPtr("added")})
			return err
		},
		"updateHighlyRelevantOperation": func(ctx context.Context) error {
			_, err := UpdateHighlyRelevantOperationRule(ctx, "group", v1alpha1.ComputeHighlyRelevantOperationHash(&relevant),
				model.HighlyRelevantOperationRuleInput{Name: stringPtr("renamed")})
			return err
		},
		"deleteHighlyRelevantOperation": func(ctx context.Context) error {
			_, err := DeleteHighlyRelevantOperationRule(ctx, "group", v1alpha1.ComputeHighlyRelevantOperationHash(&relevant))
			return err
		},
		"createCostReductionRule": func(ctx context.Context) error {
			_, err := CreateCostReductionRule(ctx, "group", model.CostReductionRuleInput{Name: stringPtr("added")})
			return err
		},
		"updateCostReductionRule": func(ctx context.Context) error {
			_, err := UpdateCostReductionRule(ctx, "group", v1alpha1.ComputeCostReductionRuleHash(&cost),
				model.CostReductionRuleInput{Name: stringPtr("renamed")})
			return err
		},
		"deleteCostReductionRule": func(ctx context.Context) error {
			_, err := DeleteCostReductionRule(ctx, "group", v1alpha1.ComputeCostReductionRuleHash(&cost))
			return err
		},
	}

	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRulesHarness(t, samplingCR("group", seed))
			h.live.PrependReactor("update", "samplings", func(k8stesting.Action) (bool, runtime.Object, error) {
				return true, nil, apierrors.NewInternalError(assert.AnError)
			})

			err := mutate(context.Background())

			require.Error(t, err)
			assert.Equal(t, v1alpha1.SamplingSpec{
				Name:                     "group",
				NoisyOperations:          []v1alpha1.NoisyOperation{noisy},
				HighlyRelevantOperations: []v1alpha1.HighlyRelevantOperation{relevant},
				CostReductionRules:       []v1alpha1.CostReductionRule{cost},
			}, h.stored("group").Spec, "a failed write must leave the stored rules untouched")
		})
	}
}

// Creating a rule into a group the caller cannot read has to fail before the rule is
// built into any spec, for each of the three families.
func TestEverySamplingRuleCreatePropagatesAGroupResolutionError(t *testing.T) {
	creations := map[string]func(context.Context) error{
		"noisyOperation": func(ctx context.Context) error {
			_, err := CreateNoisyOperationRule(ctx, "group", model.NoisyOperationRuleInput{Name: stringPtr("added")})
			return err
		},
		"highlyRelevantOperation": func(ctx context.Context) error {
			_, err := CreateHighlyRelevantOperationRule(ctx, "group", model.HighlyRelevantOperationRuleInput{Name: stringPtr("added")})
			return err
		},
		"costReductionRule": func(ctx context.Context) error {
			_, err := CreateCostReductionRule(ctx, "group", model.CostReductionRuleInput{Name: stringPtr("added")})
			return err
		},
	}

	for name, create := range creations {
		t.Run(name, func(t *testing.T) {
			h := newSamplingRules(t, samplingRulesFixture{
				cacheFuncs: interceptor.Funcs{
					Get: func(context.Context, client.WithWatch, client.ObjectKey, client.Object, ...client.GetOption) error {
						return apierrors.NewForbidden(v1alpha1.Resource("samplings"), "nope", assert.AnError)
					},
				},
			})
			writes := h.countLiveWrites()

			err := create(context.Background())

			require.Error(t, err)
			assert.True(t, apierrors.IsForbidden(err), "got %v", err)
			assert.Zero(t, *writes)
		})
	}
}

func TestSamplingRuleMutationsUseTheConfiguredOdigosNamespace(t *testing.T) {
	h := newSamplingRules(t, samplingRulesFixture{namespace: "custom-odigos-ns"})

	_, err := CreateNoisyOperationRule(context.Background(), "group", model.NoisyOperationRuleInput{Name: stringPtr("rule")})
	require.NoError(t, err)

	assert.Equal(t, "custom-odigos-ns", h.stored("group").Namespace,
		"the namespace is resolved from the environment, not from a hardcoded default")
}
