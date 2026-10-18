package sampling

import (
	"reflect"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

// Each merge function is a run of near-identical `if !input.F.IsSet() { rule.F = existing.F }`
// guards, one per field. A guard wired to the wrong field (or a missing guard on a newly added
// one) is invisible to a test that sets several fields at once: it only shows when exactly one
// field moves and everything else is asserted to stay put. These tables drive one field at a
// time, in both the "client sent a value" and the "client sent an explicit null" directions, and
// compare the whole merged rule so a cross-wired guard cannot hide.

// omMergeCase isolates one field: it builds an input where only that field is present and the
// matching expectation from the stored rule.
type omMergeCase[R any, I any] struct {
	field string
	// value fills in the one field with something different from what the rule already stores.
	value     func(input *I)
	wantValue func(want *R)
	// null sends the field explicitly as null, which clears it.
	null     func(input *I)
	wantNull func(want *R)
}

// omRunMergeMatrix drives every case against one stored rule. notMerged names the fields that are
// deliberately outside the omitted/null contract, so the coverage gate below still accounts for
// them.
func omRunMergeMatrix[R any, I any](
	t *testing.T,
	existing R,
	baseInput func() I,
	merge func(R, I) R,
	cases []omMergeCase[R, I],
	notMerged ...string,
) {
	t.Helper()

	covered := make(map[string]bool, len(cases)+len(notMerged))
	for _, name := range notMerged {
		covered[name] = true
	}
	ruleType := reflect.TypeOf(existing)
	for _, testCase := range cases {
		_, found := ruleType.FieldByName(testCase.field)
		require.True(t, found, "%s has no field %s", ruleType.Name(), testCase.field)
		require.False(t, covered[testCase.field], "duplicate case for %s.%s", ruleType.Name(), testCase.field)
		covered[testCase.field] = true

		t.Run(testCase.field+"/value", func(t *testing.T) {
			input := baseInput()
			testCase.value(&input)
			want := existing
			testCase.wantValue(&want)
			require.Equal(t, want, merge(existing, input),
				"sending only %s must change %s and leave every other field alone", testCase.field, testCase.field)
		})

		t.Run(testCase.field+"/explicitNull", func(t *testing.T) {
			input := baseInput()
			testCase.null(&input)
			want := existing
			testCase.wantNull(&want)
			require.Equal(t, want, merge(existing, input),
				"a null %s must clear %s and leave every other field alone", testCase.field, testCase.field)
		})
	}

	t.Run("everythingOmitted", func(t *testing.T) {
		require.Equal(t, existing, merge(existing, baseInput()),
			"an update that omits every field (a bulk toggle re-sending nothing) must be a no-op")
	})

	// A field added to the CRD rule without a merge guard would silently reset on every partial
	// update; make that a test failure rather than a support ticket.
	require.Equal(t, ruleType.NumField(), len(covered),
		"%s has %d fields but only %d are classified by this table", ruleType.Name(), ruleType.NumField(), len(covered))
	for i := range ruleType.NumField() {
		require.True(t, covered[ruleType.Field(i).Name],
			"%s.%s is not covered by the partial-update matrix", ruleType.Name(), ruleType.Field(i).Name)
	}
}

func omStoredNoisyOperation() v1alpha1.NoisyOperation {
	return v1alpha1.NoisyOperation{
		Name:             "payments healthz",
		Disabled:         true,
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Operation:        &commonapisampling.HeadSamplingOperationMatcher{HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/healthz"}},
		PercentageAtMost: float64Ptr(5),
		Notes:            "stored note",
	}
}

func TestMergeNoisyOperationUpdateMergesEachFieldIndependently(t *testing.T) {
	omRunMergeMatrix(t, omStoredNoisyOperation(),
		func() model.NoisyOperationRuleInput { return model.NoisyOperationRuleInput{} },
		mergeNoisyOperationUpdate,
		[]omMergeCase[v1alpha1.NoisyOperation, model.NoisyOperationRuleInput]{
			{
				field:     "Name",
				value:     func(in *model.NoisyOperationRuleInput) { in.Name = graphql.OmittableOf(stringPtr("renamed")) },
				wantValue: func(want *v1alpha1.NoisyOperation) { want.Name = "renamed" },
				null:      func(in *model.NoisyOperationRuleInput) { in.Name = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.NoisyOperation) { want.Name = "" },
			},
			{
				field:     "Disabled",
				value:     func(in *model.NoisyOperationRuleInput) { in.Disabled = graphql.OmittableOf(boolPtr(false)) },
				wantValue: func(want *v1alpha1.NoisyOperation) { want.Disabled = false },
				null:      func(in *model.NoisyOperationRuleInput) { in.Disabled = graphql.OmittableOf[*bool](nil) },
				wantNull:  func(want *v1alpha1.NoisyOperation) { want.Disabled = false },
			},
			{
				field: "SourceScopes",
				value: func(in *model.NoisyOperationRuleInput) {
					in.SourceScopes = graphql.OmittableOf(&model.SourcesScopesInput{Namespaces: []string{"checkout"}})
				},
				wantValue: func(want *v1alpha1.NoisyOperation) {
					want.SourceScopes = &k8sconsts.SourcesScopes{Namespaces: []string{"checkout"}}
				},
				null: func(in *model.NoisyOperationRuleInput) {
					in.SourceScopes = graphql.OmittableOf[*model.SourcesScopesInput](nil)
				},
				wantNull: func(want *v1alpha1.NoisyOperation) { want.SourceScopes = nil },
			},
			{
				field: "Operation",
				value: func(in *model.NoisyOperationRuleInput) {
					in.Operation = graphql.OmittableOf(&model.HeadSamplingOperationMatcherInput{
						HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{Route: stringPtr("/metrics")},
					})
				},
				wantValue: func(want *v1alpha1.NoisyOperation) {
					want.Operation = &commonapisampling.HeadSamplingOperationMatcher{
						HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/metrics"},
					}
				},
				null: func(in *model.NoisyOperationRuleInput) {
					in.Operation = graphql.OmittableOf[*model.HeadSamplingOperationMatcherInput](nil)
				},
				wantNull: func(want *v1alpha1.NoisyOperation) { want.Operation = nil },
			},
			{
				field: "PercentageAtMost",
				value: func(in *model.NoisyOperationRuleInput) {
					in.PercentageAtMost = graphql.OmittableOf(float64Ptr(42))
				},
				wantValue: func(want *v1alpha1.NoisyOperation) { want.PercentageAtMost = float64Ptr(42) },
				null:      func(in *model.NoisyOperationRuleInput) { in.PercentageAtMost = graphql.OmittableOf[*float64](nil) },
				wantNull:  func(want *v1alpha1.NoisyOperation) { want.PercentageAtMost = nil },
			},
			{
				field:     "Notes",
				value:     func(in *model.NoisyOperationRuleInput) { in.Notes = graphql.OmittableOf(stringPtr("new note")) },
				wantValue: func(want *v1alpha1.NoisyOperation) { want.Notes = "new note" },
				null:      func(in *model.NoisyOperationRuleInput) { in.Notes = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.NoisyOperation) { want.Notes = "" },
			},
		})
}

func omStoredHighlyRelevantOperation() v1alpha1.HighlyRelevantOperation {
	return v1alpha1.HighlyRelevantOperation{
		Name:              "slow charges",
		Disabled:          true,
		SourceScopes:      &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Error:             true,
		DurationAtLeastMs: intPtr(500),
		Operation:         &commonapisampling.TailSamplingOperationMatcher{HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/charge"}},
		PercentageAtLeast: float64Ptr(50),
		Notes:             "stored note",
	}
}

func TestMergeHighlyRelevantOperationUpdateMergesEachFieldIndependently(t *testing.T) {
	omRunMergeMatrix(t, omStoredHighlyRelevantOperation(),
		func() model.HighlyRelevantOperationRuleInput { return model.HighlyRelevantOperationRuleInput{} },
		mergeHighlyRelevantOperationUpdate,
		[]omMergeCase[v1alpha1.HighlyRelevantOperation, model.HighlyRelevantOperationRuleInput]{
			{
				field:     "Name",
				value:     func(in *model.HighlyRelevantOperationRuleInput) { in.Name = graphql.OmittableOf(stringPtr("renamed")) },
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.Name = "renamed" },
				null:      func(in *model.HighlyRelevantOperationRuleInput) { in.Name = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.HighlyRelevantOperation) { want.Name = "" },
			},
			{
				field:     "Disabled",
				value:     func(in *model.HighlyRelevantOperationRuleInput) { in.Disabled = graphql.OmittableOf(boolPtr(false)) },
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.Disabled = false },
				null:      func(in *model.HighlyRelevantOperationRuleInput) { in.Disabled = graphql.OmittableOf[*bool](nil) },
				wantNull:  func(want *v1alpha1.HighlyRelevantOperation) { want.Disabled = false },
			},
			{
				field: "SourceScopes",
				value: func(in *model.HighlyRelevantOperationRuleInput) {
					in.SourceScopes = graphql.OmittableOf(&model.SourcesScopesInput{Namespaces: []string{"checkout"}})
				},
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) {
					want.SourceScopes = &k8sconsts.SourcesScopes{Namespaces: []string{"checkout"}}
				},
				null: func(in *model.HighlyRelevantOperationRuleInput) {
					in.SourceScopes = graphql.OmittableOf[*model.SourcesScopesInput](nil)
				},
				wantNull: func(want *v1alpha1.HighlyRelevantOperation) { want.SourceScopes = nil },
			},
			{
				field:     "Error",
				value:     func(in *model.HighlyRelevantOperationRuleInput) { in.Error = graphql.OmittableOf(boolPtr(false)) },
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.Error = false },
				null:      func(in *model.HighlyRelevantOperationRuleInput) { in.Error = graphql.OmittableOf[*bool](nil) },
				wantNull:  func(want *v1alpha1.HighlyRelevantOperation) { want.Error = false },
			},
			{
				field: "DurationAtLeastMs",
				value: func(in *model.HighlyRelevantOperationRuleInput) {
					in.DurationAtLeastMs = graphql.OmittableOf(intPtr(1500))
				},
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.DurationAtLeastMs = intPtr(1500) },
				null: func(in *model.HighlyRelevantOperationRuleInput) {
					in.DurationAtLeastMs = graphql.OmittableOf[*int](nil)
				},
				wantNull: func(want *v1alpha1.HighlyRelevantOperation) { want.DurationAtLeastMs = nil },
			},
			{
				field: "Operation",
				value: func(in *model.HighlyRelevantOperationRuleInput) {
					in.Operation = graphql.OmittableOf(&model.TailSamplingOperationMatcherInput{
						KafkaConsumer: &model.TailSamplingKafkaMatcherInput{KafkaTopic: stringPtr("orders")},
					})
				},
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) {
					want.Operation = &commonapisampling.TailSamplingOperationMatcher{
						KafkaConsumer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "orders"},
					}
				},
				null: func(in *model.HighlyRelevantOperationRuleInput) {
					in.Operation = graphql.OmittableOf[*model.TailSamplingOperationMatcherInput](nil)
				},
				wantNull: func(want *v1alpha1.HighlyRelevantOperation) { want.Operation = nil },
			},
			{
				field: "PercentageAtLeast",
				value: func(in *model.HighlyRelevantOperationRuleInput) {
					in.PercentageAtLeast = graphql.OmittableOf(float64Ptr(100))
				},
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.PercentageAtLeast = float64Ptr(100) },
				null: func(in *model.HighlyRelevantOperationRuleInput) {
					in.PercentageAtLeast = graphql.OmittableOf[*float64](nil)
				},
				wantNull: func(want *v1alpha1.HighlyRelevantOperation) { want.PercentageAtLeast = nil },
			},
			{
				field: "Notes",
				value: func(in *model.HighlyRelevantOperationRuleInput) {
					in.Notes = graphql.OmittableOf(stringPtr("new note"))
				},
				wantValue: func(want *v1alpha1.HighlyRelevantOperation) { want.Notes = "new note" },
				null:      func(in *model.HighlyRelevantOperationRuleInput) { in.Notes = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.HighlyRelevantOperation) { want.Notes = "" },
			},
		})
}

func omStoredCostReductionRule() v1alpha1.CostReductionRule {
	return v1alpha1.CostReductionRule{
		Name:             "checkout drop",
		Disabled:         true,
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"checkout"}},
		Operation:        &commonapisampling.TailSamplingOperationMatcher{KafkaConsumer: &commonapisampling.TailSamplingKafkaOperationMatcher{KafkaTopic: "orders"}},
		PercentageAtMost: 10,
		Notes:            "stored note",
	}
}

func TestMergeCostReductionRuleUpdateMergesEachFieldIndependently(t *testing.T) {
	stored := omStoredCostReductionRule()
	// percentageAtMost is non-nullable in the schema, so every update carries it and it has no
	// merge guard; echoing the stored value keeps the other cases isolated.
	baseInput := func() model.CostReductionRuleInput {
		return model.CostReductionRuleInput{PercentageAtMost: stored.PercentageAtMost}
	}

	omRunMergeMatrix(t, stored, baseInput, mergeCostReductionRuleUpdate,
		[]omMergeCase[v1alpha1.CostReductionRule, model.CostReductionRuleInput]{
			{
				field:     "Name",
				value:     func(in *model.CostReductionRuleInput) { in.Name = graphql.OmittableOf(stringPtr("renamed")) },
				wantValue: func(want *v1alpha1.CostReductionRule) { want.Name = "renamed" },
				null:      func(in *model.CostReductionRuleInput) { in.Name = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.CostReductionRule) { want.Name = "" },
			},
			{
				field:     "Disabled",
				value:     func(in *model.CostReductionRuleInput) { in.Disabled = graphql.OmittableOf(boolPtr(false)) },
				wantValue: func(want *v1alpha1.CostReductionRule) { want.Disabled = false },
				null:      func(in *model.CostReductionRuleInput) { in.Disabled = graphql.OmittableOf[*bool](nil) },
				wantNull:  func(want *v1alpha1.CostReductionRule) { want.Disabled = false },
			},
			{
				field: "SourceScopes",
				value: func(in *model.CostReductionRuleInput) {
					in.SourceScopes = graphql.OmittableOf(&model.SourcesScopesInput{Namespaces: []string{"payments"}})
				},
				wantValue: func(want *v1alpha1.CostReductionRule) {
					want.SourceScopes = &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}}
				},
				null: func(in *model.CostReductionRuleInput) {
					in.SourceScopes = graphql.OmittableOf[*model.SourcesScopesInput](nil)
				},
				wantNull: func(want *v1alpha1.CostReductionRule) { want.SourceScopes = nil },
			},
			{
				field: "Operation",
				value: func(in *model.CostReductionRuleInput) {
					in.Operation = graphql.OmittableOf(&model.TailSamplingOperationMatcherInput{
						HTTPServer: &model.TailSamplingHTTPServerMatcherInput{Route: stringPtr("/api/checkout")},
					})
				},
				wantValue: func(want *v1alpha1.CostReductionRule) {
					want.Operation = &commonapisampling.TailSamplingOperationMatcher{
						HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{Route: "/api/checkout"},
					}
				},
				null: func(in *model.CostReductionRuleInput) {
					in.Operation = graphql.OmittableOf[*model.TailSamplingOperationMatcherInput](nil)
				},
				wantNull: func(want *v1alpha1.CostReductionRule) { want.Operation = nil },
			},
			{
				field:     "Notes",
				value:     func(in *model.CostReductionRuleInput) { in.Notes = graphql.OmittableOf(stringPtr("new note")) },
				wantValue: func(want *v1alpha1.CostReductionRule) { want.Notes = "new note" },
				null:      func(in *model.CostReductionRuleInput) { in.Notes = graphql.OmittableOf[*string](nil) },
				wantNull:  func(want *v1alpha1.CostReductionRule) { want.Notes = "" },
			},
		},
		"PercentageAtMost")
}

// percentageAtMost is the one sampling rule field the client must always send. Unlike every other
// field it is taken from the input even when the rest of the update is omitted, so a client that
// forgets it rewrites the rule to 0% — which for a cost reduction rule means dropping everything
// that matches rather than leaving the stored percentage alone.
func TestMergeCostReductionRuleUpdateAlwaysTakesPercentageFromTheInput(t *testing.T) {
	stored := omStoredCostReductionRule()
	require.NotZero(t, stored.PercentageAtMost)

	got := mergeCostReductionRuleUpdate(stored, model.CostReductionRuleInput{PercentageAtMost: 75})
	require.Equal(t, 75.0, got.PercentageAtMost)

	unset := mergeCostReductionRuleUpdate(stored, model.CostReductionRuleInput{})
	require.Zero(t, unset.PercentageAtMost,
		"a missing percentageAtMost resets the rule to 0%%; the schema marks it non-null so clients must send it")
}
