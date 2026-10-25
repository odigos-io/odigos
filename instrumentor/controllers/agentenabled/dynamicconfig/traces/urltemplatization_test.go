package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	urltemplatizationactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/stretchr/testify/require"
)

func TestCalculateUrlTemplatizationConfig_dedupesStatusCodes(t *testing.T) {
	agentLevelActions := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &urltemplatizationactions.URLTemplatizationConfig{
				Default: []urltemplatizationactions.URLTemplatizationDefaultTemplatizationGroup{
					{
						DefaultTemplatizationConfig: actions.DefaultTemplatizationConfig{
							SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404, 401},
							},
						},
					},
					{
						DefaultTemplatizationConfig: actions.DefaultTemplatizationConfig{
							SkipPolicy: &actions.DefaultTemplatizationSkipPolicyConfig{
								SkipHttpStatusCodes: []int{404, 500},
							},
						},
					},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateUrlTemplatizationConfig(&agentLevelActions, "container", common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.NotNil(t, got.Default)
	require.NotNil(t, got.Default.SkipPolicy)
	require.Equal(t, []int{401, 404, 500}, got.Default.SkipPolicy.SkipHttpStatusCodes)
}

func TestCalculateUrlTemplatizationConfig_appliesDocumentedTemplates(t *testing.T) {
	agentLevelActions := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &urltemplatizationactions.URLTemplatizationConfig{
				Rules: []urltemplatizationactions.UrlTemplatizationRule{
					{
						DocumentedTemplates: []urltemplatizationactions.UrlTemplatizationDocumentedTemplate{
							{
								Template: "/users/{id}/orders",
								Examples: []string{"/users/42/orders"},
								Notes:    "learned from live traffic",
							},
						},
					},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateUrlTemplatizationConfig(&agentLevelActions, "container", common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []string{"/users/{id}/orders"}, got.Templates)
}

func TestCalculateUrlTemplatizationConfig_mergesDocumentedAndPlainTemplates(t *testing.T) {
	agentLevelActions := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &urltemplatizationactions.URLTemplatizationConfig{
				Rules: []urltemplatizationactions.UrlTemplatizationRule{
					{
						Templates: []string{"/carts/{id}", "/users/{id}"},
						DocumentedTemplates: []urltemplatizationactions.UrlTemplatizationDocumentedTemplate{
							// also present in Templates, must not be duplicated
							{Template: "/users/{id}"},
							{Template: "/orders/{id}"},
						},
					},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateUrlTemplatizationConfig(&agentLevelActions, "container", common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []string{"/carts/{id}", "/orders/{id}", "/users/{id}"}, got.Templates)
}

func TestCalculateUrlTemplatizationConfig_skipsDocumentedTemplatesOutOfScope(t *testing.T) {
	agentLevelActions := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			URLTemplatization: &urltemplatizationactions.URLTemplatizationConfig{
				Rules: []urltemplatizationactions.UrlTemplatizationRule{
					{
						Scopes: &k8sconsts.SourcesScopes{Namespaces: []string{"other"}},
						DocumentedTemplates: []urltemplatizationactions.UrlTemplatizationDocumentedTemplate{
							{Template: "/users/{id}"},
						},
					},
					{
						Scopes: &k8sconsts.SourcesScopes{Namespaces: []string{"default"}},
						DocumentedTemplates: []urltemplatizationactions.UrlTemplatizationDocumentedTemplate{
							{Template: "/orders/{id}"},
						},
					},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateUrlTemplatizationConfig(&agentLevelActions, "container", common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []string{"/orders/{id}"}, got.Templates)
}
