package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsInstrumentationRuleUiGenerated(t *testing.T) {
	require.False(t, isInstrumentationRuleUiGenerated(nil))
	require.False(t, isInstrumentationRuleUiGenerated(&v1alpha1.InstrumentationRule{}))
	require.False(t, isInstrumentationRuleUiGenerated(&v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{k8sconsts.OdigosProfilesManagedByLabel: "helm"},
		},
	}))
	require.True(t, isInstrumentationRuleUiGenerated(&v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue,
			},
		},
	}))
}

func TestCreateInstrumentationRuleMarksUiGenerated(t *testing.T) {
	ctx := context.Background()
	ruleName := "ui rule"
	notes := ""
	disabled := false
	enabled := true

	useFakeRuleClient(t)

	created, err := CreateInstrumentationRule(ctx, model.InstrumentationRuleInput{
		RuleName:       &ruleName,
		Notes:          &notes,
		Disabled:       &disabled,
		NetworkMetrics: &enabled,
	})
	require.NoError(t, err)
	require.True(t, created.UIGenerated)
	require.Equal(t, model.ManagedByOdigosUI, created.ManagedBy)

	stored, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, created.RuleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, k8sconsts.OdigosUIManagedByValue, stored.Labels[k8sconsts.OdigosProfilesManagedByLabel])
}

func TestGetInstrumentationRuleReportsUiGenerated(t *testing.T) {
	ctx := context.Background()

	useFakeRuleClient(t,
		&v1alpha1.InstrumentationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule-ui",
				Namespace: consts.DefaultOdigosNamespace,
				Labels: map[string]string{
					k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosUIManagedByValue,
				},
			},
			Spec: v1alpha1.InstrumentationRuleSpec{RuleName: "ui"},
		},
		&v1alpha1.InstrumentationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule-yaml",
				Namespace: consts.DefaultOdigosNamespace,
			},
			Spec: v1alpha1.InstrumentationRuleSpec{RuleName: "yaml"},
		},
		&v1alpha1.InstrumentationRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule-interrogation",
				Namespace: consts.DefaultOdigosNamespace,
				Labels: map[string]string{
					k8sconsts.OdigosProfilesManagedByLabel: k8sconsts.OdigosInterrogationLoopManagedByValue,
				},
			},
			Spec: v1alpha1.InstrumentationRuleSpec{RuleName: "interrogation"},
		},
	)

	uiRule, err := GetInstrumentationRule(ctx, "rule-ui")
	require.NoError(t, err)
	require.True(t, uiRule.UIGenerated)
	require.Equal(t, model.ManagedByOdigosUI, uiRule.ManagedBy)

	yamlRule, err := GetInstrumentationRule(ctx, "rule-yaml")
	require.NoError(t, err)
	require.False(t, yamlRule.UIGenerated)
	require.Equal(t, model.ManagedByUnknown, yamlRule.ManagedBy)

	interrogationRule, err := GetInstrumentationRule(ctx, "rule-interrogation")
	require.NoError(t, err)
	require.False(t, interrogationRule.UIGenerated)
	require.Equal(t, model.ManagedByInterrogationLoop, interrogationRule.ManagedBy)

	rules, err := GetInstrumentationRules(ctx)
	require.NoError(t, err)
	require.Len(t, rules, 3)
	byID := map[string]model.ManagedBy{}
	for _, r := range rules {
		byID[r.RuleID] = r.ManagedBy
	}
	require.Equal(t, model.ManagedByOdigosUI, byID["rule-ui"])
	require.Equal(t, model.ManagedByUnknown, byID["rule-yaml"])
	require.Equal(t, model.ManagedByInterrogationLoop, byID["rule-interrogation"])
}
