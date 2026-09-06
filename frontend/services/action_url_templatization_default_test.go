package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urlactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertUrlTemplatizationDefaultToModel(t *testing.T) {
	skipNonSuccess := true
	groups := convertUrlTemplatizationDefaultToModel(&urlactions.URLTemplatizationConfig{
		Default: []urlactions.URLTemplatizationDefaultTemplatizationGroup{
			{
				DefaultTemplatizationConfig: actionsapi.DefaultTemplatizationConfig{
					SkipPolicy: &actionsapi.DefaultTemplatizationSkipPolicyConfig{
						SkipForNonSuccessCodes: true,
					},
				},
			},
		},
	})

	require.Len(t, groups, 1)
	require.NotNil(t, groups[0].SkipPolicy)
	require.NotNil(t, groups[0].SkipPolicy.SkipForNonSuccessCodes)
	require.True(t, *groups[0].SkipPolicy.SkipForNonSuccessCodes)
	require.Equal(t, skipNonSuccess, *groups[0].SkipPolicy.SkipForNonSuccessCodes)
}

func TestConvertActionToModelIncludesUrlTemplatizationDefaultGroups(t *testing.T) {
	action := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			Signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Default: []urlactions.URLTemplatizationDefaultTemplatizationGroup{
					{
						DefaultTemplatizationConfig: actionsapi.DefaultTemplatizationConfig{
							Disabled: true,
						},
					},
				},
			},
		},
	}

	modelAction, err := convertActionToModel(action)
	require.NoError(t, err)
	require.Equal(t, model.ActionTypeURLTemplatization, modelAction.Type)
	require.Len(t, modelAction.Fields.URLTemplatizationDefaultGroups, 1)
	require.NotNil(t, modelAction.Fields.URLTemplatizationDefaultGroups[0].Disabled)
	require.True(t, *modelAction.Fields.URLTemplatizationDefaultGroups[0].Disabled)
}

func TestConvertUrlTemplatizationFromInputAcceptsDefaultGroups(t *testing.T) {
	skipNonSuccess := true
	existingAction := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Rules: []urlactions.UrlTemplatizationRule{
					{Templates: []string{"/existing/{id}"}},
				},
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(&model.ActionFieldsInput{
		URLTemplatizationDefaultGroups: []*model.URLTemplatizationDefaultGroupInput{
			{
				Scopes: &model.SourcesScopesInput{
					Namespaces: []string{"prod"},
				},
				SkipPolicy: &model.URLTemplatizationDefaultSkipPolicyInput{
					SkipForNonSuccessCodes: &skipNonSuccess,
				},
			},
		},
	}, existingAction)

	require.NotNil(t, cfg)
	require.Len(t, cfg.Rules, 1)
	require.Equal(t, []string{"/existing/{id}"}, cfg.Rules[0].Templates)
	require.Len(t, cfg.Default, 1)
	require.Equal(t, []string{"prod"}, cfg.Default[0].Scopes.Namespaces)
	require.NotNil(t, cfg.Default[0].SkipPolicy)
	require.True(t, cfg.Default[0].SkipPolicy.SkipForNonSuccessCodes)
}
