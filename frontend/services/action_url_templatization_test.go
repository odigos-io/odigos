package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urlactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertUrlTemplatizationFromInputPreservesDefaultTemplatization(t *testing.T) {
	existingAction := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Default: []urlactions.URLTemplatizationDefaultTemplatizationGroup{
					{
						DefaultTemplatizationConfig: actionsapi.DefaultTemplatizationConfig{
							SkipPolicy: &actionsapi.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404},
							},
						},
					},
				},
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, &model.ActionFieldsInput{
		URLTemplatizationRulesGroups: []*model.URLTemplatizationRulesGroupInput{
			{
				TemplatizationRules: []*model.URLTemplatizationRuleInput{
					{Template: "/users/{id}"},
				},
			},
		},
	}, existingAction)

	require.NotNil(t, cfg)
	require.Len(t, cfg.Rules, 1)
	require.Equal(t, []string{"/users/{id}"}, cfg.Rules[0].Templates)
	require.Equal(t, existingAction.Spec.URLTemplatization.Default, cfg.Default)
}

func TestConvertUrlTemplatizationFromInputEmptyFieldsCreatesConfig(t *testing.T) {
	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, &model.ActionFieldsInput{}, nil)

	require.NotNil(t, cfg)
	require.Empty(t, cfg.Rules)
	require.Empty(t, cfg.Default)
}

func TestConvertUrlTemplatizationFromInputIgnoresNonUrlTemplatizationType(t *testing.T) {
	cfg := convertUrlTemplatizationFromInput(model.ActionTypeRenameAttribute, &model.ActionFieldsInput{
		URLTemplatizationDefaultGroups: []*model.URLTemplatizationDefaultGroupInput{{}},
	}, nil)

	require.Nil(t, cfg)
}

// An update that mentions no URL templatization field must not wipe the rules and default
// groups the action already has.
func TestConvertUrlTemplatizationFromInputEmptyFieldsKeepsTheExistingConfig(t *testing.T) {
	existingAction := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Rules: []urlactions.UrlTemplatizationRule{{Templates: []string{"/users/{id}"}}},
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, &model.ActionFieldsInput{}, existingAction)

	require.Same(t, existingAction.Spec.URLTemplatization, cfg)
}

// An action created without a fields object at all - the shape a recommendation remediation
// sends - still needs the empty config block, or the Action webhook rejects it.
func TestConvertUrlTemplatizationFromInputWithoutFieldsCreatesConfig(t *testing.T) {
	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, nil, nil)

	require.NotNil(t, cfg)
	require.Empty(t, cfg.Rules)
	require.Empty(t, cfg.Default)
}

func TestConvertUrlTemplatizationFromInputWithoutFieldsKeepsTheExistingConfig(t *testing.T) {
	existingAction := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Rules: []urlactions.UrlTemplatizationRule{{Templates: []string{"/users/{id}"}}},
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, nil, existingAction)

	require.Same(t, existingAction.Spec.URLTemplatization, cfg)
}
