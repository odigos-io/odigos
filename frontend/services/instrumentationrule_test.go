package services

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUpdateInstrumentationRulePreservesOmittedSourcesScopes(t *testing.T) {
	storedScopes := &k8sconsts.SourcesScopes{
		Namespaces: []string{"payments"},
		Languages:  []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}
	existingRule := &odigosv1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "scoped-rule",
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: odigosv1.InstrumentationRuleSpec{
			RuleName:      "Collect payment payloads",
			Notes:         "Scoped to payments services",
			SourcesScopes: storedScopes,
		},
	}

	previousClient := kube.DefaultClient
	fakeClient := fake.NewSimpleClientset(existingRule)
	kube.DefaultClient = &kube.Client{OdigosClient: fakeClient.OdigosV1alpha1()}
	t.Cleanup(func() {
		kube.DefaultClient = previousClient
	})

	updatedName := "Updated rule name"
	updatedNotes := "Only metadata changed"
	disabled := false

	_, err := UpdateInstrumentationRule(context.Background(), existingRule.Name, model.InstrumentationRuleInput{
		RuleName: &updatedName,
		Notes:    &updatedNotes,
		Disabled: &disabled,
	})
	require.NoError(t, err)

	updatedRule, err := fakeClient.OdigosV1alpha1().InstrumentationRules(consts.DefaultOdigosNamespace).Get(context.Background(), existingRule.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, storedScopes, updatedRule.Spec.SourcesScopes)
	require.Equal(t, updatedName, updatedRule.Spec.RuleName)
	require.Equal(t, updatedNotes, updatedRule.Spec.Notes)
}
