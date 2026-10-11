package services

import (
	"testing"

	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestConvertK8sAttributesPreservesOmittedSiblingFields(t *testing.T) {
	trueVal := true
	existing := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			K8sAttributes: &actionsv1.K8sAttributesConfig{
				CollectContainerAttributes: true,
				CollectReplicaSetAttributes: true,
				LabelsAttributes: []actionsv1.K8sLabelAttribute{{
					LabelKey:     "app",
					AttributeKey: "k8s.app",
				}},
			},
		},
	}

	cfg := convertK8sAttributesFromInput(&model.ActionFieldsInput{
		CollectContainerAttributes: &trueVal,
	}, existing)

	require.NotNil(t, cfg)
	require.True(t, cfg.CollectContainerAttributes)
	require.True(t, cfg.CollectReplicaSetAttributes, "omitted collectReplicaSetAttributes must be preserved")
	require.Equal(t, []actionsv1.K8sLabelAttribute{{
		LabelKey:     "app",
		AttributeKey: "k8s.app",
	}}, cfg.LabelsAttributes, "omitted labelsAttributes must be preserved")
}

func TestConvertK8sAttributesClearsLabelsOnExplicitEmpty(t *testing.T) {
	existing := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			K8sAttributes: &actionsv1.K8sAttributesConfig{
				LabelsAttributes: []actionsv1.K8sLabelAttribute{{
					LabelKey:     "app",
					AttributeKey: "k8s.app",
				}},
			},
		},
	}

	cfg := convertK8sAttributesFromInput(&model.ActionFieldsInput{
		LabelsAttributes: []*model.K8sLabelAttributeInput{},
	}, existing)

	require.NotNil(t, cfg)
	require.Empty(t, cfg.LabelsAttributes)
}

func TestConvertAddClusterInfoPreservesAttributesWhenOnlyOverwriteSet(t *testing.T) {
	overwrite := true
	attrValue := "prod"
	existing := &v1alpha1.Action{
		Spec: v1alpha1.ActionSpec{
			AddClusterInfo: &actionsv1.AddClusterInfoConfig{
				ClusterAttributes: []actionsv1.OtelAttributeWithValue{{
					AttributeName:        "deployment.environment",
					AttributeStringValue: &attrValue,
				}},
				OverwriteExistingValues: false,
			},
		},
	}

	cfg := convertAddClusterInfoFromInput(&model.ActionFieldsInput{
		OverwriteExistingValues: &overwrite,
	}, existing)

	require.NotNil(t, cfg)
	require.True(t, cfg.OverwriteExistingValues)
	require.Len(t, cfg.ClusterAttributes, 1)
	require.Equal(t, "deployment.environment", cfg.ClusterAttributes[0].AttributeName)
	require.Equal(t, "prod", *cfg.ClusterAttributes[0].AttributeStringValue)
}
