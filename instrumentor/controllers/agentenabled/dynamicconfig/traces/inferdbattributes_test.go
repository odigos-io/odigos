package traces

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	inferactions "github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestCalculateInferDbAttributesConfig_noActions(t *testing.T) {
	actionsList := []odigosv1.Action{}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateInferDbAttributesConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculateInferDbAttributesConfig_globalEnable(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			InferDbAttributes: &inferactions.InferDbAttributesConfig{},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateInferDbAttributesConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
}

func TestCalculateInferDbAttributesConfig_scopeMismatch(t *testing.T) {
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			InferDbAttributes: &inferactions.InferDbAttributesConfig{
				Scopes: &k8sconsts.SourcesScopes{
					Namespaces: []string{"other-ns"},
				},
			},
		},
	}}
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}

	got := CalculateInferDbAttributesConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.Nil(t, got)
}

func TestCalculateInferDbAttributesConfig_scopeMatch(t *testing.T) {
	pw := k8sconsts.PodWorkload{Name: "app", Namespace: "default", Kind: k8sconsts.WorkloadKindDeployment}
	actionsList := []odigosv1.Action{{
		Spec: odigosv1.ActionSpec{
			InferDbAttributes: &inferactions.InferDbAttributesConfig{
				Scopes: &k8sconsts.SourcesScopes{
					Namespaces: []string{pw.Namespace},
				},
			},
		},
	}}

	got := CalculateInferDbAttributesConfig(&actionsList, common.JavaProgrammingLanguage, pw)

	require.NotNil(t, got)
}
