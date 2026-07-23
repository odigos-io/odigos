package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	dbqueryactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertDbQueryTemplatizationFromInput(t *testing.T) {
	templatize := true
	cfg := convertDbQueryTemplatizationFromInput(model.ActionTypeDbQueryTemplatization, &model.ActionFieldsInput{
		TemplatizeLiterals: &templatize,
		Scopes: &model.SourcesScopesInput{
			Namespaces: []string{"default"},
			Languages:  []model.SamplingWorkloadLanguage{model.SamplingWorkloadLanguageJava},
		},
	}, nil)

	require.NotNil(t, cfg)
	require.True(t, cfg.TemplatizeLiterals)
	require.Equal(t, []string{"default"}, cfg.Scopes.Namespaces)
	require.Equal(t, []common.ProgrammingLanguage{common.JavaProgrammingLanguage}, cfg.Scopes.Languages)
}

func TestConvertDbQueryTemplatizationFromInputWrongType(t *testing.T) {
	templatize := true
	cfg := convertDbQueryTemplatizationFromInput(model.ActionTypeInferDbAttributes, &model.ActionFieldsInput{
		TemplatizeLiterals: &templatize,
	}, nil)
	require.Nil(t, cfg)
}

func TestConvertDbQueryTemplatizationPreservesTemplatizeLiterals(t *testing.T) {
	existing := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
				DbQueryTemplatizationConfig: actionsapi.DbQueryTemplatizationConfig{
					TemplatizeLiterals: true,
				},
			},
		},
	}

	cfg := convertDbQueryTemplatizationFromInput(model.ActionTypeDbQueryTemplatization, &model.ActionFieldsInput{
		Scopes: &model.SourcesScopesInput{Namespaces: []string{"ns-a"}},
	}, existing)

	require.NotNil(t, cfg)
	require.True(t, cfg.TemplatizeLiterals)
	require.Equal(t, []string{"ns-a"}, cfg.Scopes.Namespaces)
}

func TestConvertInferDbAttributesFromInput(t *testing.T) {
	cfg := convertInferDbAttributesFromInput(model.ActionTypeInferDbAttributes, &model.ActionFieldsInput{
		Scopes: &model.SourcesScopesInput{
			Sources: []*model.K8sSourceID{{
				Namespace: "default",
				Kind:      model.K8sResourceKindDeployment,
				Name:      "api",
			}},
		},
	}, nil)

	require.NotNil(t, cfg)
	require.Equal(t, []k8sconsts.PodWorkload{{
		Namespace: "default",
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      "api",
	}}, cfg.Scopes.Sources)
}

func TestConvertInferDbAttributesEmptyFields(t *testing.T) {
	cfg := convertInferDbAttributesFromInput(model.ActionTypeInferDbAttributes, &model.ActionFieldsInput{}, nil)
	require.NotNil(t, cfg)
	require.Nil(t, cfg.Scopes)
}

func TestConvertDbActionFieldsToModel(t *testing.T) {
	scopes, templatizeLiterals := convertDbActionFieldsToModel(&v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
				Scopes: &k8sconsts.SourcesScopes{Namespaces: []string{"prod"}},
				DbQueryTemplatizationConfig: actionsapi.DbQueryTemplatizationConfig{
					TemplatizeLiterals: true,
				},
			},
		},
	})

	require.NotNil(t, scopes)
	require.Equal(t, []string{"prod"}, scopes.Namespaces)
	require.NotNil(t, templatizeLiterals)
	require.True(t, *templatizeLiterals)
}

func TestDeriveTypeFromDbActions(t *testing.T) {
	dbAction := &model.Action{Fields: &model.ActionFields{}}
	require.Equal(t, model.ActionTypeDbQueryTemplatization, deriveTypeFromAction(dbAction, &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{},
		},
	}))

	inferAction := &model.Action{Fields: &model.ActionFields{}}
	require.Equal(t, model.ActionTypeInferDbAttributes, deriveTypeFromAction(inferAction, &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			InferDbAttributes: &dbqueryactions.InferDbAttributesConfig{},
		},
	}))
}
