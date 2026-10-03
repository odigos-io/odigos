package services

import (
	"fmt"
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"
)

func TestConvertRecommendationToModel_CarriesTheCatalogContent(t *testing.T) {
	useRecommendationsNamespace(t)
	catalog := recommendationCatalogEntry(t, common.RecommendationTypeSampleHealthProbes)

	converted, err := convertRecommendationToModel(recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes))

	require.NoError(t, err)
	assert.Equal(t, "sample-health-probes", converted.Name)
	assert.Equal(t, catalog.Title, converted.Title)
	assert.Equal(t, catalog.Summary, converted.Summary)
	assert.Equal(t, catalog.Description, converted.Description)
	assert.Equal(t, catalog.Categories, converted.Categories)
	assert.Equal(t, catalog.Pros, converted.Pros)
	assert.Equal(t, catalog.Cons, converted.Cons)
	require.NotNil(t, converted.DocsURL)
	assert.Equal(t, catalog.Docs, *converted.DocsURL)
	assert.Len(t, converted.Remediations, len(catalog.Remediations))
	assert.Len(t, converted.AppliedWhen, len(catalog.AppliedWhen))
}

// The catalog's oss / requireOdigosDeployment pair only takes three of its four possible
// combinations across the shipped manifests, so the table below is what distinguishes the
// two fields from each other; a fully populated single fixture could not.
func TestConvertRecommendationToModel_CarriesTheCatalogEditionFlags(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, tc := range []struct {
		recType                 common.RecommendationType
		oss                     bool
		requireOdigosDeployment bool
	}{
		{common.RecommendationTypeInferDBAttributes, true, false},
		{common.RecommendationTypeAutoGoOffsetUpdater, false, false},
		{common.RecommendationTypeEnableOwnMetrics, true, true},
	} {
		t.Run(string(tc.recType), func(t *testing.T) {
			catalog := recommendationCatalogEntry(t, tc.recType)
			require.Equal(t, tc.oss, catalog.OSS, "the shipped catalog changed, update this table")
			require.Equal(t, tc.requireOdigosDeployment, catalog.RequireOdigosDeployment,
				"the shipped catalog changed, update this table")

			converted, err := convertRecommendationToModel(recommendationCR("rec", tc.recType))

			require.NoError(t, err)
			assert.Equal(t, tc.oss, converted.Oss)
			assert.Equal(t, tc.requireOdigosDeployment, converted.RequireOdigosDeployment)
		})
	}
}

// Applied, ConditionsMet and Dismissed are three booleans read from three different places on
// the CR, so each case starts from an all-false recommendation and flips exactly one of them.
// Asserting the other two stayed false is what catches a converter reading the wrong source.
func TestConvertRecommendationToModel_EachCrStateFlagComesFromItsOwnSource(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)

	for _, tc := range []struct {
		name          string
		mutate        func(*v1alpha1.Recommendation)
		applied       bool
		conditionsMet bool
		dismissed     bool
	}{
		{name: "nothing set"},
		{
			name:    "spec.applied",
			mutate:  func(rec *v1alpha1.Recommendation) { rec.Spec.Applied = true },
			applied: true,
		},
		{
			name:          "spec.conditionsMet",
			mutate:        func(rec *v1alpha1.Recommendation) { rec.Spec.ConditionsMet = true },
			conditionsMet: true,
		},
		{
			name:      "dismissed label",
			mutate:    func(rec *v1alpha1.Recommendation) { dismissed(rec) },
			dismissed: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes)
			rec.Spec.Applied = false
			rec.Spec.ConditionsMet = false
			if tc.mutate != nil {
				tc.mutate(rec)
			}

			converted, err := convertRecommendationToModel(rec)

			require.NoError(t, err)
			assert.Equal(t, tc.applied, converted.Applied)
			assert.Equal(t, tc.conditionsMet, converted.ConditionsMet)
			assert.Equal(t, tc.dismissed, converted.Dismissed)
		})
	}
}

func TestConvertRecommendationToModel_FallsBackToTheCatalogEntryNamedLikeTheCr(t *testing.T) {
	useRecommendationsNamespace(t)
	// go-offset-updater is the k8sObjectName of the AutoGoOffsetUpdater entry, and differs
	// from both its type and its manifest file name.
	catalog := recommendationCatalogEntry(t, common.RecommendationTypeAutoGoOffsetUpdater)
	require.Equal(t, "go-offset-updater", catalog.K8sObjectName, "the shipped catalog changed, update this test")

	converted, err := convertRecommendationToModel(recommendationCR("go-offset-updater", "TypeTheCatalogDoesNotKnow"))

	require.NoError(t, err)
	assert.Equal(t, catalog.Title, converted.Title, "catalog content must be resolved by the CR name")
}

func TestConvertRecommendationToModel_UnknownTypeAndNameIsAnError(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)

	_, err := convertRecommendationToModel(recommendationCR("no-such-recommendation", "TypeTheCatalogDoesNotKnow"))

	// Asserted exactly rather than with Contains: the message interpolates the type and the
	// name, and Contains cannot tell the two apart if they are ever swapped.
	require.EqualError(t, err,
		`recommendation catalog entry not found for type "TypeTheCatalogDoesNotKnow" (name "no-such-recommendation")`)
}

func TestIsRecommendationDismissed(t *testing.T) {
	for _, tc := range []struct {
		name   string
		labels map[string]string
		want   bool
	}{
		{name: "no labels"},
		{name: "empty labels", labels: map[string]string{}},
		{name: "unrelated label", labels: map[string]string{"odigos.io/other": "true"}},
		{name: "dismissed false", labels: map[string]string{"odigos.io/recommendation-dismissed": "false"}},
		{name: "dismissed empty", labels: map[string]string{"odigos.io/recommendation-dismissed": ""}},
		{name: "dismissed true", labels: map[string]string{"odigos.io/recommendation-dismissed": "true"}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := recommendationCR("rec", common.RecommendationTypeSampleHealthProbes)
			rec.Labels = tc.labels

			assert.Equal(t, tc.want, isRecommendationDismissed(rec))
		})
	}
}

func TestToCatalogConditions(t *testing.T) {
	converted := toCatalogConditions([]recommendations.Condition{
		{Type: recommendations.ConditionTypeGoEnterpriseSources},
		{Type: "SecondCondition"},
	})

	require.Len(t, converted, 2)
	assert.Equal(t, recommendations.ConditionTypeGoEnterpriseSources, converted[0].Type)
	assert.Equal(t, "SecondCondition", converted[1].Type)
}

// catalogConditions is a GraphQL non-null list, so an absent conditions block has to render as
// [] rather than null.
func TestToCatalogConditionsRendersNoConditionsAsAnEmptyList(t *testing.T) {
	assert.Equal(t, []*model.RecommendationCatalogCondition{}, toCatalogConditions(nil))
}

func TestToAppliedWhen_OnlyPopulatesTheFieldTheCheckKindUses(t *testing.T) {
	converted := toAppliedWhen([]recommendations.AppliedWhenCheck{
		{Type: recommendations.AppliedWhenTypeEffectiveConfig, Expression: "goAutoOffsetsMode != 'off'"},
		{Type: recommendations.AppliedWhenTypeActionExists, ActionType: "InferDbAttributes"},
		{Type: "CheckWithNeitherField"},
	})

	require.Len(t, converted, 3)

	assert.Equal(t, recommendations.AppliedWhenTypeEffectiveConfig, converted[0].Type)
	require.NotNil(t, converted[0].Expression)
	assert.Equal(t, "goAutoOffsetsMode != 'off'", *converted[0].Expression)
	assert.Nil(t, converted[0].ActionType)

	assert.Equal(t, recommendations.AppliedWhenTypeActionExists, converted[1].Type)
	require.NotNil(t, converted[1].ActionType)
	assert.Equal(t, "InferDbAttributes", *converted[1].ActionType)
	assert.Nil(t, converted[1].Expression)

	assert.Nil(t, converted[2].Expression)
	assert.Nil(t, converted[2].ActionType)
}

// Each entry must carry its own copy of the optional strings; sharing one address across the
// loop would give every check the last entry's expression.
func TestToAppliedWhenGivesEachCheckItsOwnStrings(t *testing.T) {
	converted := toAppliedWhen([]recommendations.AppliedWhenCheck{
		{Type: recommendations.AppliedWhenTypeEffectiveConfig, Expression: "first"},
		{Type: recommendations.AppliedWhenTypeEffectiveConfig, Expression: "second"},
	})

	require.Len(t, converted, 2)
	require.NotNil(t, converted[0].Expression)
	require.NotNil(t, converted[1].Expression)
	assert.Equal(t, "first", *converted[0].Expression)
	assert.Equal(t, "second", *converted[1].Expression)
}

func TestToAppliedWhenRendersNoChecksAsAnEmptyList(t *testing.T) {
	assert.Equal(t, []*model.RecommendationAppliedWhen{}, toAppliedWhen(nil))
}

// canApplyViaUi is what makes the UI offer the apply button instead of a copy-paste snippet,
// so it must follow the presence of catalog steps and nothing else.
func TestToCatalogRemediations_CanApplyViaUiFollowsTheCatalogSteps(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, tc := range []struct {
		name  string
		steps []recommendations.RemediationStep
		want  bool
	}{
		{name: "no steps"},
		{name: "empty step list", steps: []recommendations.RemediationStep{}},
		{
			name:  "an edit config step",
			steps: []recommendations.RemediationStep{{Type: recommendations.RemediationStepTypeEditConfig, Path: "clusterName", Value: "x"}},
			want:  true,
		},
		{
			name:  "an apply action step",
			steps: []recommendations.RemediationStep{{Type: recommendations.RemediationStepTypeApplyOdigosAction}},
			want:  true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			converted, err := toCatalogRemediations([]recommendations.Remediation{{
				Type:       "SomeRemediation",
				ButtonText: "Do it",
				Tooltip:    "Tooltip text",
				Steps:      tc.steps,
			}})

			require.NoError(t, err)
			require.Len(t, converted, 1)
			assert.Equal(t, tc.want, converted[0].CanApplyViaUI)
			assert.Equal(t, "SomeRemediation", converted[0].Type)
			assert.Equal(t, "Do it", converted[0].ButtonText)
			assert.Equal(t, "Tooltip text", converted[0].Tooltip)
		})
	}
}

func TestToCatalogRemediations_PropagatesABrokenApplyExample(t *testing.T) {
	useRecommendationsNamespace(t)

	_, err := toCatalogRemediations([]recommendations.Remediation{{
		Type: "SomeRemediation",
		ApplyExamples: []recommendations.ApplyExample{{
			Type:    recommendations.ApplyExampleTypeOdigosAction,
			Content: "this: is: not: valid: yaml",
		}},
	}})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse OdigosAction applyExamples")
}

func TestToCatalogApplyExamples_InjectsTheNamespaceOnlyIntoActionExamples(t *testing.T) {
	useRecommendationsNamespace(t)

	helmValues := "sampling:\n  k8sHealthProbesSampling:\n    enabled: true\n"
	converted, err := toCatalogApplyExamples([]recommendations.ApplyExample{
		{Type: recommendations.ApplyExampleTypeHelmValues, Content: helmValues},
		{
			Type: recommendations.ApplyExampleTypeOdigosAction,
			Content: "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: url-templatization\n" +
				"spec:\n  actionName: URL Templatization\n  signals:\n  - TRACES\n  urlTemplatization: {}\n",
		},
	})

	require.NoError(t, err)
	require.Len(t, converted, 2)

	// A Helm values snippet is not an Action manifest. Parsing it as one would silently drop
	// every key and hand the user an empty snippet, so it has to be passed through verbatim.
	assert.Equal(t, recommendations.ApplyExampleTypeHelmValues, converted[0].Type)
	assert.Equal(t, helmValues, converted[0].Content)

	assert.Equal(t, recommendations.ApplyExampleTypeOdigosAction, converted[1].Type)
	var action v1alpha1.Action
	require.NoError(t, yaml.Unmarshal([]byte(converted[1].Content), &action))
	assert.Equal(t, recommendationsTestNamespace, action.Namespace)
	assert.Equal(t, "url-templatization", action.Name)
	assert.Equal(t, "URL Templatization", action.Spec.ActionName)
}

func TestToCatalogApplyExamplesRendersNoExamplesAsAnEmptyList(t *testing.T) {
	useRecommendationsNamespace(t)

	converted, err := toCatalogApplyExamples(nil)

	require.NoError(t, err)
	assert.Equal(t, []*model.RecommendationCatalogApplyExample{}, converted)
}

// The apply example is marshalled back out through the typed Action struct, so an empty config
// block has to survive as `{}`. A nil pointer would render the snippet as an Action with no
// config at all, which the Action admission webhook rejects.
func TestInjectOdigosActionNamespaceKeepsAnEmptyActionConfig(t *testing.T) {
	content := "apiVersion: odigos.io/v1alpha1\nkind: Action\nmetadata:\n  name: url-templatization\n" +
		"spec:\n  actionName: URL Templatization\n  signals:\n  - TRACES\n  urlTemplatization: {}\n"

	out, err := injectOdigosActionNamespace(content, recommendationsTestNamespace)

	require.NoError(t, err)
	assert.Contains(t, out, "urlTemplatization: {}")

	var action v1alpha1.Action
	require.NoError(t, yaml.Unmarshal([]byte(out), &action))
	require.NotNil(t, action.Spec.URLTemplatization, "the empty config block must survive the round trip")
	assert.Equal(t, recommendationsTestNamespace, action.Namespace)
	assert.Equal(t, []common.ObservabilitySignal{common.TracesObservabilitySignal}, action.Spec.Signals)
}

func TestInjectOdigosActionNamespaceRejectsContentThatIsNotAnAction(t *testing.T) {
	_, err := injectOdigosActionNamespace("\tnot yaml at all", recommendationsTestNamespace)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse OdigosAction applyExamples")
}

func TestOptionalString(t *testing.T) {
	assert.Nil(t, optionalString(""))

	got := optionalString("https://docs.odigos.io")
	require.NotNil(t, got)
	assert.Equal(t, "https://docs.odigos.io", *got)
}

// categories, pros and cons are GraphQL non-null lists; returning a nil slice would serialise
// as null and fail the UI's query validation.
func TestNonNilStrings(t *testing.T) {
	assert.Equal(t, []string{}, nonNilStrings(nil))
	assert.NotNil(t, nonNilStrings(nil))
	assert.Equal(t, []string{"a"}, nonNilStrings([]string{"a"}))
}

func TestEveryShippedRecommendationRendersWithoutNullLists(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, entry := range loadRecommendationCatalog(t) {
		t.Run(entry.K8sObjectName, func(t *testing.T) {
			converted, err := convertRecommendationToModel(recommendationCR(entry.K8sObjectName, entry.Type))

			require.NoError(t, err)
			assert.NotNil(t, converted.Categories, "categories is a non-null GraphQL list")
			assert.NotNil(t, converted.Pros, "pros is a non-null GraphQL list")
			assert.NotNil(t, converted.Cons, "cons is a non-null GraphQL list")
			assert.NotNil(t, converted.CatalogConditions, "catalogConditions is a non-null GraphQL list")
			assert.NotNil(t, converted.AppliedWhen, "appliedWhen is a non-null GraphQL list")
			assert.NotNil(t, converted.Remediations, "remediations is a non-null GraphQL list")
			for i, remediation := range converted.Remediations {
				assert.NotNil(t, remediation.ApplyExamples,
					fmt.Sprintf("remediations[%d].applyExamples is a non-null GraphQL list", i))
			}
		})
	}
}
