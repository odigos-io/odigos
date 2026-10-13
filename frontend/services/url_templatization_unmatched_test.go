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

func TestFilterPathsNotMatchingRules(t *testing.T) {
	rules := parseTemplatizationPathRules([]string{"/users/{id}", "/health"})
	cn := "app"
	paths := []*model.UnmatchedURLPath{
		{Path: "/users/123", Count: 10, ContainerName: &cn},
		{Path: "/users/456/orders", Count: 5, ContainerName: &cn},
		{Path: "/health", Count: 3, ContainerName: &cn},
		{Path: "/api/v1/items", Count: 2, ContainerName: &cn},
	}

	filtered := filterPathsNotMatchingRules(paths, rules)
	require.Len(t, filtered, 2)
	require.Equal(t, "/users/456/orders", filtered[0].Path)
	require.Equal(t, "/api/v1/items", filtered[1].Path)
}

func TestFilterPathsNotMatchingRules_NoRules(t *testing.T) {
	paths := []*model.UnmatchedURLPath{{Path: "/users/123", Count: 1}}
	require.Equal(t, paths, filterPathsNotMatchingRules(paths, nil))
}

func TestUrlTemplatizationConfigToExistingModel(t *testing.T) {
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

	got := urlTemplatizationConfigToExistingModel("app", cfg, resolveExistingTemplates(nil, k8sconsts.PodWorkload{}, common.UnknownProgrammingLanguage, cfg))
	require.Equal(t, "app", got.ContainerName)
	require.Equal(t, []string{"/health", "/users/{id}"}, templateStrings(got.Templates))
	require.NotNil(t, got.Default)
	require.False(t, got.Default.Disabled)
	require.NotNil(t, got.Default.SkipPolicy)
	require.NotNil(t, got.Default.SkipPolicy.SkipForNonSuccessCodes)
	require.True(t, *got.Default.SkipPolicy.SkipForNonSuccessCodes)
	require.Equal(t, []int{404}, got.Default.SkipPolicy.SkipHTTPStatusCodes)
}

func TestUrlTemplatizationConfigToExistingModel_NilDefault(t *testing.T) {
	cfg := &actions.UrlTemplatizationConfig{Templates: []string{"/users/{id}"}}

	got := urlTemplatizationConfigToExistingModel("app", cfg, resolveExistingTemplates(nil, k8sconsts.PodWorkload{}, common.UnknownProgrammingLanguage, cfg))
	require.Equal(t, []string{"/users/{id}"}, templateStrings(got.Templates))
	require.Nil(t, got.Default)
}

func templateStrings(templates []*model.URLTemplatizationExistingTemplate) []string {
	out := make([]string, 0, len(templates))
	for _, template := range templates {
		out = append(out, template.Template)
	}
	return out
}

func TestResolveExistingTemplates_FromMatchingActionRules(t *testing.T) {
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

	got := resolveExistingTemplates([]odigosv1.Action{action}, pw, common.GoProgrammingLanguage, cfg)
	require.Equal(t, []string{"/legacy/{id}", "/users/{id}"}, templateStrings(got))

	// documented rule keeps its learning context and owning action
	require.Equal(t, []string{"/users/123"}, got[1].Examples)
	require.Equal(t, "learned from live traffic", *got[1].Notes)
	require.Equal(t, "action-abc", *got[1].ActionID)
	require.Equal(t, model.ManagedByOdigosUI, got[1].ManagedBy)

	// template only present in the resolved config is kept without an owning action
	require.Nil(t, got[0].ActionID)
	require.Equal(t, model.ManagedByUnknown, got[0].ManagedBy)
}
