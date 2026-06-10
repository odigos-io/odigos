package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urlactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	commonactions "github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertUrlTemplatizationFromInputPreservesExistingDefaultConfig(t *testing.T) {
	existingDefault := []urlactions.URLTemplatizationDefaultTemplatizationGroup{{
		Scopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"public"},
		},
		DefaultTemplatizationConfig: commonactions.DefaultTemplatizationConfig{
			SkipPolicy: &commonactions.DefaultTemplatizationSkipPolicyConfig{
				SkipForNonSuccessCodes: true,
			},
		},
	}}
	existingAction := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			URLTemplatization: &urlactions.URLTemplatizationConfig{
				Default: existingDefault,
			},
		},
	}

	cfg := convertUrlTemplatizationFromInput(&model.ActionFieldsInput{
		URLTemplatizationRulesGroups: []*model.URLTemplatizationRulesGroupInput{{
			TemplatizationRules: []*model.URLTemplatizationRuleInput{{
				Template: "/users/{id}",
			}},
		}},
	}, existingAction)

	require.NotNil(t, cfg)
	require.Equal(t, existingDefault, cfg.Default)
	require.Len(t, cfg.Rules, 1)
	require.Equal(t, []string{"/users/{id}"}, cfg.Rules[0].Templates)
}
