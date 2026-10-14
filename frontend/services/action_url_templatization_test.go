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
	require.Empty(t, cfg.Rules[0].DocumentedTemplates)
	require.Equal(t, existingAction.Spec.URLTemplatization.Default, cfg.Default)
}

func TestConvertUrlTemplatizationFromInputWritesDocumentedTemplates(t *testing.T) {
	notes := "from live traffic"
	examples := []string{"/users/1", "/users/2", "/users/3", "/users/4", "/users/5", "/users/6"}

	cfg := convertUrlTemplatizationFromInput(model.ActionTypeURLTemplatization, &model.ActionFieldsInput{
		URLTemplatizationRulesGroups: []*model.URLTemplatizationRulesGroupInput{
			{
				TemplatizationRules: []*model.URLTemplatizationRuleInput{
					{Template: "/health"},
					{
						Template: "/users/{id}",
						Notes:    &notes,
						Examples: examples,
					},
					{
						Template: "/orders/{id}",
						Examples: []string{},
					},
				},
			},
		},
	}, nil)

	require.NotNil(t, cfg)
	require.Len(t, cfg.Rules, 1)
	require.Equal(t, []string{"/health"}, cfg.Rules[0].Templates)
	require.Equal(t, []urlactions.UrlTemplatizationDocumentedTemplate{
		{
			Template: "/users/{id}",
			Notes:    notes,
			Examples: []string{"/users/1", "/users/2", "/users/3", "/users/4", "/users/5"},
		},
		{
			Template: "/orders/{id}",
		},
	}, cfg.Rules[0].DocumentedTemplates)
}

func TestConvertUrlTemplatizationToModelReadsDocumentedTemplates(t *testing.T) {
	groups := convertUrlTemplatizationToModel(&urlactions.URLTemplatizationConfig{
		Rules: []urlactions.UrlTemplatizationRule{
			{
				Templates: []string{"/health"},
				DocumentedTemplates: []urlactions.UrlTemplatizationDocumentedTemplate{
					{
						Template: "/users/{id}",
						Notes:    "from live traffic",
						Examples: []string{"/users/1", "/users/2"},
					},
				},
			},
		},
	})

	require.Len(t, groups, 1)
	require.Len(t, groups[0].TemplatizationRules, 2)
	require.Equal(t, "/health", groups[0].TemplatizationRules[0].Template)
	require.Nil(t, groups[0].TemplatizationRules[0].Notes)
	require.Nil(t, groups[0].TemplatizationRules[0].Examples)

	require.Equal(t, "/users/{id}", groups[0].TemplatizationRules[1].Template)
	require.NotNil(t, groups[0].TemplatizationRules[1].Notes)
	require.Equal(t, "from live traffic", *groups[0].TemplatizationRules[1].Notes)
	require.Equal(t, []string{"/users/1", "/users/2"}, groups[0].TemplatizationRules[1].Examples)
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
