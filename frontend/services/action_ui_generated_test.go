package services

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsActionUiGenerated(t *testing.T) {
	require.False(t, isActionUiGenerated(nil))
	require.False(t, isActionUiGenerated(&v1alpha1.Action{}))
	require.False(t, isActionUiGenerated(&v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{k8sconsts.OdigosCreatedByLabel: "helm"},
		},
	}))
	require.True(t, isActionUiGenerated(&v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				k8sconsts.OdigosCreatedByLabel: k8sconsts.OdigosCreatedByOdigosUIValue,
			},
		},
	}))
}

func TestConvertActionToModelReportsUiGenerated(t *testing.T) {
	action := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name: "action-ui",
			Labels: map[string]string{
				k8sconsts.OdigosCreatedByLabel: k8sconsts.OdigosCreatedByOdigosUIValue,
			},
		},
	}
	modelAction, err := convertActionToModel(action)
	require.NoError(t, err)
	require.True(t, modelAction.UIGenerated)

	yamlAction := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{Name: "action-yaml"},
	}
	modelYaml, err := convertActionToModel(yamlAction)
	require.NoError(t, err)
	require.False(t, modelYaml.UIGenerated)
}
