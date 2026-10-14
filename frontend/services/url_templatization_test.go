package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	actionsv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUrlTemplatizationConfigToModel(t *testing.T) {
	cfg := &actions.UrlTemplatizationConfig{
		Templates: []string{"/users/{id}", "/health"},
		Default: &actions.DefaultTemplatizationConfig{
			Disabled: false,
			SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{
				SkipForNonSuccessCodes: true,
				SkipHttpStatusCodes:    []int{404},
			},
		},
	}

	got := UrlTemplatizationConfigToModel(cfg, ResolveExistingUrlTemplatizationTemplates(nil, k8sconsts.PodWorkload{}, common.UnknownProgrammingLanguage, cfg))
	require.Equal(t, []string{"/health", "/users/{id}"}, templateStrings(got.Templates))
	require.NotNil(t, got.PatternBasedTemplatization)
	require.False(t, got.PatternBasedTemplatization.Disabled)
	require.NotNil(t, got.PatternBasedTemplatization.SkipPolicy)
	require.NotNil(t, got.PatternBasedTemplatization.SkipPolicy.SkipForNonSuccessCodes)
	require.True(t, *got.PatternBasedTemplatization.SkipPolicy.SkipForNonSuccessCodes)
	require.Equal(t, []int{404}, got.PatternBasedTemplatization.SkipPolicy.SkipHTTPStatusCodes)
}

func TestResolveExistingUrlTemplatizationTemplates_FromMatchingActionRules(t *testing.T) {
	action := odigosv1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "action-abc",
			Labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue},
		},
		Spec: odigosv1.ActionSpec{
			ActionName: "URLTemplatization",
			URLTemplatization: &actionsv1.URLTemplatizationConfig{
				Rules: []actionsv1.UrlTemplatizationRule{
					{
						Scopes: &k8sconsts.SourcesScopes{Namespaces: []string{"default"}},
						DocumentedTemplates: []actionsv1.UrlTemplatizationDocumentedTemplate{
							{Template: "/users/{id}", Examples: []string{"/users/123"}, Notes: "learned from live traffic"},
						},
					},
					{
						Scopes:    &k8sconsts.SourcesScopes{Namespaces: []string{"other"}},
						Templates: []string{"/not-applied/{id}"},
					},
				},
			},
		},
	}

	pw := k8sconsts.PodWorkload{Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment, Name: "api"}
	cfg := &actions.UrlTemplatizationConfig{Templates: []string{"/users/{id}", "/legacy/{id}"}}

	got := ResolveExistingUrlTemplatizationTemplates([]odigosv1.Action{action}, pw, common.GoProgrammingLanguage, cfg)
	require.Equal(t, []string{"/legacy/{id}", "/users/{id}"}, templateStrings(got))

	require.Equal(t, []string{"/users/123"}, got[1].Examples)
	require.Equal(t, "learned from live traffic", *got[1].Notes)
	require.Equal(t, "action-abc", *got[1].ActionID)
	require.Equal(t, model.ManagedByOdigosUI, got[1].ManagedBy)

	require.Nil(t, got[0].ActionID)
	require.Equal(t, model.ManagedByUnknown, got[0].ManagedBy)
}

func templateStrings(templates []*model.LiveTrafficLearningTemplate) []string {
	out := make([]string, 0, len(templates))
	for _, template := range templates {
		out = append(out, template.Template)
	}
	return out
}
