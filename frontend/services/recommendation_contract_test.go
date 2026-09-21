package services

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The GraphQL schema used to carry the recommendation type as an enum, which made the
// conversions on both sides of the UI mechanical. It is a plain String now, so nothing but this
// test links the type the query emits to the type the apply mutation accepts: a rename on
// either side silently turns every apply into "not found in catalog".
func TestTheRecommendationTypeTheQueryEmitsIsTheOneTheApplyMutationAccepts(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, entry := range loadRecommendationCatalog(t) {
		t.Run(entry.K8sObjectName, func(t *testing.T) {
			setFakeRecommendationClient(t)

			converted, err := convertRecommendationToModel(recommendationCR(entry.K8sObjectName, entry.Type))
			require.NoError(t, err)

			// Ask for a remediation that cannot exist: reaching the remediation lookup at all
			// proves the recommendation itself was resolved from the emitted type.
			err = ApplyRecommendationRemediation(context.Background(), recommendationsConfigClient(t),
				converted.Type, "NoSuchRemediation")

			require.Error(t, err)
			assert.NotContains(t, err.Error(), "not found in catalog",
				"the type the query emits must resolve back to a catalog entry")
			assert.Contains(t, err.Error(), `remediation type "NoSuchRemediation" not found`)
		})
	}
}

// canApplyViaUi is the only thing that decides whether the UI shows an apply button or a
// copy-paste snippet. If it says yes and the backend then refuses, the button is dead; if it
// says no for a remediation the backend would have applied, the feature is unreachable.
func TestCanApplyViaUiPredictsWhetherTheBackendAcceptsTheRemediation(t *testing.T) {
	useRecommendationsNamespace(t)
	catalog := loadRecommendationCatalog(t)

	applicable := 0
	snippetOnly := 0
	for _, entry := range catalog {
		converted, err := convertRecommendationToModel(recommendationCR(entry.K8sObjectName, entry.Type))
		require.NoError(t, err)
		require.Len(t, converted.Remediations, len(entry.Remediations))

		for _, remediation := range converted.Remediations {
			t.Run(entry.K8sObjectName+"/"+remediation.Type, func(t *testing.T) {
				setFakeRecommendationClient(t)

				err := ApplyRecommendationRemediation(context.Background(), recommendationsConfigClient(t),
					converted.Type, remediation.Type)

				if remediation.CanApplyViaUI {
					require.NoError(t, err, "canApplyViaUi is true, so the backend must accept it")
					return
				}
				require.Error(t, err, "canApplyViaUi is false, so the backend must refuse it")
			})

			if remediation.CanApplyViaUI {
				applicable++
			} else {
				snippetOnly++
			}
		}
	}

	// Guard against the assertions above passing vacuously: the shipped catalog has to keep
	// exercising both sides of the gate.
	assert.Positive(t, applicable, "no remediation in the catalog is applicable from the UI")
	assert.Positive(t, snippetOnly, "no remediation in the catalog is snippet-only")
}

// The autoscaler marks an ActionExists recommendation as applied by reflecting the catalog's
// actionType off ActionSpec (see actionHasType in autoscaler/controllers/recommendations). The
// Action this package creates from the apply example therefore has to populate exactly that
// field, or the recommendation stays "not applied" after the user enables it.
func TestTheActionCreatedFromTheCatalogPopulatesTheFieldAppliedWhenLooksFor(t *testing.T) {
	useRecommendationsNamespace(t)

	checked := 0
	for _, entry := range loadRecommendationCatalog(t) {
		for _, remediation := range entry.Remediations {
			if len(recommendations.ActionSteps(remediation.Steps)) == 0 {
				continue
			}

			t.Run(entry.K8sObjectName+"/"+remediation.Type, func(t *testing.T) {
				actionTypes := actionExistsActionTypes(entry)
				require.NotEmpty(t, actionTypes,
					"a remediation that creates an Action needs an ActionExists appliedWhen check")

				setFakeRecommendationClient(t)
				require.NoError(t, ApplyRecommendationRemediation(context.Background(),
					recommendationsConfigClient(t), string(entry.Type), remediation.Type))

				actions := storedActions(t)
				require.Len(t, actions, 1)
				for _, actionType := range actionTypes {
					field := reflect.ValueOf(actions[0].Spec).FieldByName(actionType)
					require.True(t, field.IsValid(),
						"ActionSpec has no field %q named by the appliedWhen check", actionType)
					require.Equal(t, reflect.Ptr, field.Kind())
					assert.False(t, field.IsNil(),
						"the created action must set ActionSpec.%s", actionType)
				}
			})
			checked++
		}
	}

	assert.Positive(t, checked, "no remediation in the catalog creates an Action any more")
}

func actionExistsActionTypes(entry recommendations.Recommendation) []string {
	var result []string
	for _, check := range entry.AppliedWhen {
		if check.Type == recommendations.AppliedWhenTypeActionExists && check.ActionType != "" {
			result = append(result, check.ActionType)
		}
	}
	return result
}

// Adding a recommendation means editing three independent places: the constant in common, a
// manifest under recommendations/manifests, and the Enum marker on RecommendationSpec.Type that
// the CRD is generated from. The list below is a fourth witness, so a manifest whose type has no
// constant (the CR is then rejected by the API server) cannot land unnoticed.
func TestTheShippedCatalogCoversExactlyTheKnownRecommendationTypes(t *testing.T) {
	knownTypes := []common.RecommendationType{
		common.RecommendationTypeInferDBAttributes,
		common.RecommendationTypeAutoGoOffsetUpdater,
		common.RecommendationTypeEnableOwnMetrics,
		common.RecommendationTypeSampleHealthProbes,
		common.RecommendationTypeUrlTemplatization,
	}

	catalog := loadRecommendationCatalog(t)

	catalogTypes := make([]string, 0, len(catalog))
	for _, entry := range catalog {
		catalogTypes = append(catalogTypes, string(entry.Type))
	}
	expected := make([]string, 0, len(knownTypes))
	for _, recType := range knownTypes {
		expected = append(expected, string(recType))
	}

	assert.ElementsMatch(t, expected, catalogTypes,
		"every recommendation type constant needs a manifest and vice versa")
}

// RemediationByType returns the first match, so a duplicated remediation type in one manifest
// would make the second one unreachable from the UI without any load-time complaint.
func TestEveryRemediationTypeIsUniqueWithinItsRecommendation(t *testing.T) {
	for _, entry := range loadRecommendationCatalog(t) {
		t.Run(entry.K8sObjectName, func(t *testing.T) {
			seen := map[string]bool{}
			for _, remediation := range entry.Remediations {
				assert.NotEmpty(t, strings.TrimSpace(remediation.Type), "a remediation needs a type")
				assert.False(t, seen[remediation.Type], "duplicate remediation type %q", remediation.Type)
				seen[remediation.Type] = true
			}
		})
	}
}

// Every Action the catalog can create is created by this package, so it must also be one the UI
// keeps treating as its own afterwards.
func TestEveryActionTheCatalogCreatesStaysEditableFromTheUi(t *testing.T) {
	useRecommendationsNamespace(t)

	for _, entry := range loadRecommendationCatalog(t) {
		for _, remediation := range entry.Remediations {
			if _, ok := remediation.OdigosActionExample(); !ok {
				continue
			}

			t.Run(entry.K8sObjectName+"/"+remediation.Type, func(t *testing.T) {
				setFakeRecommendationClient(t)
				require.NoError(t, applyOdigosActionStep(context.Background(), remediation))

				actions := storedActions(t)
				require.Len(t, actions, 1)
				assert.True(t, isActionUiGenerated(&actions[0]))
				// The scheduler garbage collects anything marked as profile-managed that the
				// current profile hash does not account for, so this action must not claim to be.
				assert.NotEqual(t, k8sconsts.OdigosProfilesManagedByValue,
					actions[0].Labels[k8sconsts.OdigosProfilesManagedByLabel])
				assertActionCarriesSomeConfig(t, &actions[0])
			})
		}
	}
}

// An Action with no config block at all is rejected by the Action admission webhook, which is
// the failure the empty-config converter fix was about.
func assertActionCarriesSomeConfig(t *testing.T, action *v1alpha1.Action) {
	t.Helper()

	spec := reflect.ValueOf(action.Spec)
	for i := 0; i < spec.NumField(); i++ {
		if spec.Field(i).Kind() == reflect.Ptr && !spec.Field(i).IsNil() {
			return
		}
	}
	t.Fatalf("action %q has no action config set", action.Name)
}
