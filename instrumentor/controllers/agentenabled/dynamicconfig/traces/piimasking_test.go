package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	piiactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/stretchr/testify/require"
)

func TestCalculatePiiMaskingConfig_noActions(t *testing.T) {
	actionsList := []odigosv1.Action{}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculatePiiMaskingConfig_globalEnable(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &piiactions.PiiMaskingConfig{
				PiiMaskingConfig: actions.PiiMaskingConfig{
					PiiCategories: []actions.PiiCategory{actions.CreditCardMasking, actions.EmailMasking},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []actions.PiiCategory{actions.CreditCardMasking, actions.EmailMasking}, got.PiiCategories)
}

func TestCalculatePiiMaskingConfig_scopeMismatch(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &piiactions.PiiMaskingConfig{
				Scopes: &k8sconsts.SourcesScopes{
					Namespaces: []string{"other-ns"},
				},
				PiiMaskingConfig: actions.PiiMaskingConfig{
					PiiCategories: []actions.PiiCategory{actions.CreditCardMasking},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculatePiiMaskingConfig_scopeMatch(t *testing.T) {
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &piiactions.PiiMaskingConfig{
				Scopes: &k8sconsts.SourcesScopes{
					Namespaces: []string{pw.Namespace},
				},
				PiiMaskingConfig: actions.PiiMaskingConfig{
					PiiCategories: []actions.PiiCategory{actions.JwtMasking},
					CustomFormatMaskings: []actions.CustomFormatMasking{{
						LookupKey:  "ssn",
						DataFormat: actions.FormatJSON,
					}},
					CustomRegexMaskings: []actions.CustomRegexMasking{{
						Regex: `(secret=)(\w+)`,
					}},
				},
			},
		},
	}}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []actions.PiiCategory{actions.JwtMasking}, got.PiiCategories)
	require.Equal(t, []actions.CustomFormatMasking{{LookupKey: "ssn", DataFormat: actions.FormatJSON}}, got.CustomFormatMaskings)
	require.Equal(t, []actions.CustomRegexMasking{{Regex: `(secret=)(\w+)`}}, got.CustomRegexMaskings)
}

func TestCalculatePiiMaskingConfig_unionsMatchingActions(t *testing.T) {
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	actionsList := []odigosv1.Action{
		{
			Spec: odigosv1.ActionSpec{
				PiiMasking: &piiactions.PiiMaskingConfig{
					PiiMaskingConfig: actions.PiiMaskingConfig{
						PiiCategories: []actions.PiiCategory{actions.EmailMasking, actions.CreditCardMasking},
						CustomRegexMaskings: []actions.CustomRegexMasking{{
							Regex: `(token=)(\w+)`,
						}},
					},
				},
			},
		},
		{
			Spec: odigosv1.ActionSpec{
				PiiMasking: &piiactions.PiiMaskingConfig{
					PiiMaskingConfig: actions.PiiMaskingConfig{
						PiiCategories: []actions.PiiCategory{actions.CreditCardMasking, actions.UuidMasking},
						CustomFormatMaskings: []actions.CustomFormatMasking{{
							LookupKey:  "email",
							DataFormat: actions.FormatJSON,
						}},
						CustomRegexMaskings: []actions.CustomRegexMasking{{
							Regex: `(token=)(\w+)`,
						}},
					},
				},
			},
		},
	}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.Equal(t, []actions.PiiCategory{actions.CreditCardMasking, actions.EmailMasking, actions.UuidMasking}, got.PiiCategories)
	require.Equal(t, []actions.CustomFormatMasking{{LookupKey: "email", DataFormat: actions.FormatJSON}}, got.CustomFormatMaskings)
	require.Equal(t, []actions.CustomRegexMasking{{Regex: `(token=)(\w+)`}}, got.CustomRegexMaskings)
}

func TestCalculatePiiMaskingConfig_emptyRulesReturnsNil(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			PiiMasking: &piiactions.PiiMaskingConfig{},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculatePiiMaskingConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}
