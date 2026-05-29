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
				DefaultTemplatizations: []urlactions.URLTemplatizationDefaultTemplatizationGroup{
					{
						Config: actionsapi.DefaultTemplatizationConfig{
							SkipPolicy: &actionsapi.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404},
							},
						},
					},
				},
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(&model.ActionFieldsInput{
		URLTemplatizationRulesGroups: []*model.URLTemplatizationRulesGroupInput{
			{
				TemplatizationRules: []*model.URLTemplatizationRuleInput{
					{Template: "/users/{id}"},
				},
			},
		},
	}, existingAction)

	require.NotNil(t, cfg)
	require.Len(t, cfg.TemplatizationRulesGroups, 1)
	require.Equal(t, "/users/{id}", cfg.TemplatizationRulesGroups[0].TemplatizationRules[0].Template)
	require.Equal(t, existingAction.Spec.URLTemplatization.DefaultTemplatizations, cfg.DefaultTemplatizations)
}
