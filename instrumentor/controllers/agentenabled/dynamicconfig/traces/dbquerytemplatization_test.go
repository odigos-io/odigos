package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	dbqueryactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/actions"
	"github.com/stretchr/testify/require"
)

func TestCalculateDbQueryTemplatizationConfig_noActions(t *testing.T) {
	actionsList := []odigosv1.Action{}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateDbQueryTemplatizationConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculateDbQueryTemplatizationConfig_globalEnable(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
				DbQueryTemplatizationConfig: actions.DbQueryTemplatizationConfig{
					TemplatizeLiterals: true,
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateDbQueryTemplatizationConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.True(t, got.TemplatizeLiterals)
}

func TestCalculateDbQueryTemplatizationConfig_scopeMismatch(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
				Scopes: &k8sconsts.SourcesScopes{
					Namespaces: []string{"other-ns"},
				},
				DbQueryTemplatizationConfig: actions.DbQueryTemplatizationConfig{
					TemplatizeLiterals: true,
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateDbQueryTemplatizationConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculateDbQueryTemplatizationConfig_orsMatchingActions(t *testing.T) {
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	actionsList := []odigosv1.Action{
		{
			Spec: odigosv1.ActionSpec{
				DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
					Scopes: &k8sconsts.SourcesScopes{
						Namespaces: []string{pw.Namespace},
					},
					DbQueryTemplatizationConfig: actions.DbQueryTemplatizationConfig{
						TemplatizeLiterals: false,
					},
				},
			},
		},
		{
			Spec: odigosv1.ActionSpec{
				DbQueryTemplatization: &dbqueryactions.DbQueryTemplatizationConfig{
					Scopes: &k8sconsts.SourcesScopes{
						Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
					},
					DbQueryTemplatizationConfig: actions.DbQueryTemplatizationConfig{
						TemplatizeLiterals: true,
					},
				},
			},
		},
	}

	got := CalculateDbQueryTemplatizationConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
	require.True(t, got.TemplatizeLiterals)
}
