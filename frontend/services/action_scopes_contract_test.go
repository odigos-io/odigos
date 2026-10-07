package services

// Three action configs carry a top-level `scopes` selecting which sources they apply to, and each
// one wires that field through convertActionToModel and getSpecFromInput by hand. #5972 is what
// happens when one of them is missed: PII masking had been scopable since #5341, convertActionToModel
// never read spec.piiMasking.scopes, and the UI showed a namespace-scoped masking action as
// applying to every source in the cluster.
//
// These tests classify the scoped configs by reflection so a fourth one cannot be added without a
// read contract, and they pin the write side, which #5972 described ("updates keep ignoring scopes
// for PII masking and preserve the existing value") but did not test.

import (
	"reflect"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigosactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

// scCase pins one ActionSpec config that carries a top-level Scopes field to the Action fixtures
// that exercise it.
type scCase struct {
	actionType model.ActionType
	// withScopes builds an Action of this type whose config is scoped to the given selection.
	withScopes func(scopes *k8sconsts.SourcesScopes) *v1alpha1.Action
	// scopesTakenFromInput records whether getSpecFromInput reads Scopes off the GraphQL input.
	// The PII masking form has no scope picker, so its input is deliberately ignored.
	scopesTakenFromInput bool
	// readSpecScopes returns the scopes stored on the reconciled spec.
	readSpecScopes func(spec *v1alpha1.ActionSpec) *k8sconsts.SourcesScopes
	// editConfigFields is an input that changes the config itself while still omitting scopes.
	// Each converter preserves scopes down two separate paths — an untouched config is returned
	// whole, an edited one is rebuilt from a DeepCopy — and only this input reaches the second.
	// nil when the config has no field besides scopes to edit.
	editConfigFields func() *model.ActionFieldsInput
}

func scSignals() []common.ObservabilitySignal {
	return []common.ObservabilitySignal{common.TracesObservabilitySignal}
}

func scAction(spec v1alpha1.ActionSpec) *v1alpha1.Action {
	spec.Signals = scSignals()
	return &v1alpha1.Action{Spec: spec}
}

func scCases() map[string]scCase {
	return map[string]scCase{
		"PiiMasking": {
			actionType:           model.ActionTypePiiMasking,
			scopesTakenFromInput: false,
			withScopes: func(scopes *k8sconsts.SourcesScopes) *v1alpha1.Action {
				return scAction(v1alpha1.ActionSpec{PiiMasking: &odigosactions.PiiMaskingConfig{
					Scopes: scopes,
					PiiMaskingConfig: actionsapi.PiiMaskingConfig{
						PiiCategories: []actionsapi.PiiCategory{actionsapi.EmailMasking},
					},
				}})
			},
			readSpecScopes: func(s *v1alpha1.ActionSpec) *k8sconsts.SourcesScopes {
				if s.PiiMasking == nil {
					return nil
				}
				return s.PiiMasking.Scopes
			},
			editConfigFields: func() *model.ActionFieldsInput {
				return &model.ActionFieldsInput{PiiCategories: []string{"CREDIT_CARD"}}
			},
		},
		"DbQueryTemplatization": {
			actionType:           model.ActionTypeDbQueryTemplatization,
			scopesTakenFromInput: true,
			withScopes: func(scopes *k8sconsts.SourcesScopes) *v1alpha1.Action {
				return scAction(v1alpha1.ActionSpec{
					DbQueryTemplatization: &odigosactions.DbQueryTemplatizationConfig{Scopes: scopes},
				})
			},
			readSpecScopes: func(s *v1alpha1.ActionSpec) *k8sconsts.SourcesScopes {
				if s.DbQueryTemplatization == nil {
					return nil
				}
				return s.DbQueryTemplatization.Scopes
			},
			editConfigFields: func() *model.ActionFieldsInput {
				templatize := true
				return &model.ActionFieldsInput{TemplatizeLiterals: &templatize}
			},
		},
		"InferDbAttributes": {
			actionType:           model.ActionTypeInferDbAttributes,
			scopesTakenFromInput: true,
			withScopes: func(scopes *k8sconsts.SourcesScopes) *v1alpha1.Action {
				return scAction(v1alpha1.ActionSpec{
					InferDbAttributes: &odigosactions.InferDbAttributesConfig{Scopes: scopes},
				})
			},
			readSpecScopes: func(s *v1alpha1.ActionSpec) *k8sconsts.SourcesScopes {
				if s.InferDbAttributes == nil {
					return nil
				}
				return s.InferDbAttributes.Scopes
			},
		},
	}
}

// scFullScopes exercises all three selector dimensions. #5972's own test asserted only Namespaces,
// which cannot see a conversion that drops sources or languages.
func scFullScopes() *k8sconsts.SourcesScopes {
	return &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{
			{Name: "checkout", Namespace: "shop", Kind: k8sconsts.WorkloadKindDeployment},
			{Name: "cron", Namespace: "ops", Kind: k8sconsts.WorkloadKindCronJob},
		},
		Namespaces: []string{"shop", "ops"},
		Languages:  []common.ProgrammingLanguage{common.JavaProgrammingLanguage, common.GoProgrammingLanguage},
	}
}

func scFullScopesInput() *model.SourcesScopesInput {
	return &model.SourcesScopesInput{
		Sources: []*model.K8sSourceID{
			{Name: "payments", Namespace: "billing", Kind: model.K8sResourceKindDeployment},
		},
		Namespaces: []string{"billing"},
		Languages:  []model.SamplingWorkloadLanguage{model.SamplingWorkloadLanguageJava},
	}
}

// scActionSpecHasTopLevelScopes reports whether an ActionSpec config declares the scopes field the
// GraphQL `fields.scopes` is built from. URL templatization is deliberately excluded: its scopes
// live inside each rule group and are surfaced through those groups instead.
func scActionSpecHasTopLevelScopes(configType reflect.Type) bool {
	f, ok := configType.FieldByName("Scopes")
	return ok && f.Type == reflect.TypeOf(&k8sconsts.SourcesScopes{})
}

// TestEveryTopLevelScopedActionConfigHasAScopeContract is the gate #5972 would have tripped: the
// day an action config grows a top-level `scopes`, it needs a case here or the UI will keep
// reporting it as applying to every source.
func TestEveryTopLevelScopedActionConfigHasAScopeContract(t *testing.T) {
	specType := reflect.TypeOf(v1alpha1.ActionSpec{})
	require.Positive(t, specType.NumField())

	cases := scCases()
	scopedFields := 0
	for i := 0; i < specType.NumField(); i++ {
		f := specType.Field(i)
		if f.Type.Kind() != reflect.Ptr || f.Type.Elem().Kind() != reflect.Struct {
			continue
		}
		_, covered := cases[f.Name]
		if !scActionSpecHasTopLevelScopes(f.Type.Elem()) {
			require.Falsef(t, covered,
				"ActionSpec.%s has a scope contract but no top-level Scopes field", f.Name)
			continue
		}
		scopedFields++
		require.Truef(t, covered,
			"ActionSpec.%s carries a top-level Scopes field with no case in scCases(). "+
				"convertActionToModel has to surface it or the UI reports the action as cluster-wide.",
			f.Name)
	}

	require.Equal(t, len(cases), scopedFields)
	require.GreaterOrEqual(t, scopedFields, 3, "ActionSpec lost a scoped action config unexpectedly")
}

// TestConvertActionToModelSurfacesScopesForEveryScopedConfig checks the whole selector survives,
// not just the dimension that happened to be in the fixture of the fix.
func TestConvertActionToModelSurfacesScopesForEveryScopedConfig(t *testing.T) {
	want := &model.SourcesScopes{
		Sources: []*model.K8sWorkloadID{
			{Name: "checkout", Namespace: "shop", Kind: model.K8sResourceKindDeployment},
			{Name: "cron", Namespace: "ops", Kind: model.K8sResourceKindCronJob},
		},
		Namespaces: []string{"shop", "ops"},
		Languages: []model.SamplingWorkloadLanguage{
			model.SamplingWorkloadLanguage(common.JavaProgrammingLanguage),
			model.SamplingWorkloadLanguage(common.GoProgrammingLanguage),
		},
	}

	for name, tc := range scCases() {
		t.Run(name, func(t *testing.T) {
			out, err := convertActionToModel(tc.withScopes(scFullScopes()))
			require.NoError(t, err)
			require.Equal(t, tc.actionType, out.Type)
			require.Equal(t, want, out.Fields.Scopes)
		})
	}
}

// TestConvertActionToModelReportsAnUnscopedActionAsUnscoped is the other half: "nil scopes" is how
// the UI learns the action really does apply everywhere, so it must not be synthesised.
func TestConvertActionToModelReportsAnUnscopedActionAsUnscoped(t *testing.T) {
	for name, tc := range scCases() {
		t.Run(name, func(t *testing.T) {
			out, err := convertActionToModel(tc.withScopes(nil))
			require.NoError(t, err)
			require.Nil(t, out.Fields.Scopes)
		})
	}

	t.Run("URLTemplatization keeps its scopes on the rule groups", func(t *testing.T) {
		out, err := convertActionToModel(scAction(v1alpha1.ActionSpec{
			URLTemplatization: &odigosactions.URLTemplatizationConfig{
				Rules: []odigosactions.UrlTemplatizationRule{{
					Scopes:    scFullScopes(),
					Templates: []string{"/api/v1/orders/{id}"},
				}},
			},
		}))
		require.NoError(t, err)
		require.Nil(t, out.Fields.Scopes,
			"a per-group scope must not be promoted to the action-wide scopes field")
		require.Len(t, out.Fields.URLTemplatizationRulesGroups, 1)
		require.NotNil(t, out.Fields.URLTemplatizationRulesGroups[0].Scopes)
		require.Equal(t, []string{"shop", "ops"}, out.Fields.URLTemplatizationRulesGroups[0].Scopes.Namespaces)
	})
}

// TestUpdatingAScopedActionFromTheUIPreservesItsScopes pins the claim #5972 made about the write
// side. The UI forms do not all expose a scope picker, so an edit that omits scopes must leave a
// selection authored in YAML alone — widening a PII masking action from one namespace to the whole
// cluster is a privacy regression, not a cosmetic one.
func TestUpdatingAScopedActionFromTheUIPreservesItsScopes(t *testing.T) {
	for name, tc := range scCases() {
		t.Run(name+", config untouched", func(t *testing.T) {
			spec, err := getSpecFromInput(model.ActionInput{
				Type:    tc.actionType,
				Name:    StringPtr("renamed from the UI"),
				Signals: []model.SignalType{model.SignalTypeTraces},
				Fields:  &model.ActionFieldsInput{},
			}, tc.withScopes(scFullScopes()))

			require.NoError(t, err)
			require.Equal(t, "renamed from the UI", spec.ActionName, "the edit must still apply")
			require.Equal(t, scFullScopes(), tc.readSpecScopes(spec),
				"an update that omits scopes must not widen the action")
		})

		if tc.editConfigFields == nil {
			continue
		}
		t.Run(name+", config edited", func(t *testing.T) {
			spec, err := getSpecFromInput(model.ActionInput{
				Type:    tc.actionType,
				Signals: []model.SignalType{model.SignalTypeTraces},
				Fields:  tc.editConfigFields(),
			}, tc.withScopes(scFullScopes()))

			require.NoError(t, err)
			require.Equal(t, scFullScopes(), tc.readSpecScopes(spec),
				"rebuilding the config from the input must carry the scopes over")
		})
	}
}

// TestUpdatingAScopedActionAppliesScopesFromTheInput is the complementary half: for the forms that
// do send scopes, an update has to replace the stored selection rather than keep the old one.
func TestUpdatingAScopedActionAppliesScopesFromTheInput(t *testing.T) {
	wantReplaced := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{
			{Name: "payments", Namespace: "billing", Kind: k8sconsts.WorkloadKindDeployment},
		},
		Namespaces: []string{"billing"},
		Languages:  []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}

	for name, tc := range scCases() {
		t.Run(name, func(t *testing.T) {
			spec, err := getSpecFromInput(model.ActionInput{
				Type:    tc.actionType,
				Signals: []model.SignalType{model.SignalTypeTraces},
				Fields:  &model.ActionFieldsInput{Scopes: scFullScopesInput()},
			}, tc.withScopes(scFullScopes()))
			require.NoError(t, err)

			if !tc.scopesTakenFromInput {
				// The PII masking form has no scope picker, so the resolver ignores the field
				// rather than letting a form that cannot show scopes rewrite them. Remove this
				// branch once the form grows one.
				require.Equal(t, scFullScopes(), tc.readSpecScopes(spec))
				return
			}
			require.Equal(t, wantReplaced, tc.readSpecScopes(spec))
		})
	}
}

// TestUpdatingAScopedActionCanClearItsScopes separates "the input omitted scopes" from "the input
// sent an empty selection", which are the same JSON-ish shape but must not mean the same thing:
// only the second one makes the action cluster-wide.
func TestUpdatingAScopedActionCanClearItsScopes(t *testing.T) {
	for name, tc := range scCases() {
		if !tc.scopesTakenFromInput {
			continue
		}
		t.Run(name, func(t *testing.T) {
			spec, err := getSpecFromInput(model.ActionInput{
				Type:    tc.actionType,
				Signals: []model.SignalType{model.SignalTypeTraces},
				Fields:  &model.ActionFieldsInput{Scopes: &model.SourcesScopesInput{}},
			}, tc.withScopes(scFullScopes()))
			require.NoError(t, err)
			require.Equal(t, &k8sconsts.SourcesScopes{}, tc.readSpecScopes(spec),
				"an explicitly empty scope selection must clear the stored one")
		})
	}
}
