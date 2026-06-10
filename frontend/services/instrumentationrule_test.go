package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func setFakeOdigosClient(t *testing.T, objects ...*v1alpha1.InstrumentationRule) {
	t.Helper()

	runtimeObjects := make([]runtime.Object, 0, len(objects))
	for _, obj := range objects {
		runtimeObjects = append(runtimeObjects, obj)
	}

	clientset := odigosfake.NewSimpleClientset(runtimeObjects...)
	previousClient := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() {
		kube.SetDefaultClient(previousClient)
	})
}

func TestUpdateInstrumentationRulePreservesOmittedScopesAndLibraries(t *testing.T) {
	ctx := context.Background()
	ruleID := "scoped-rule"
	ruleName := "renamed rule"
	notes := "updated notes"
	disabled := false

	scopes := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{{
			Name:      "checkout",
			Namespace: "prod",
			Kind:      k8sconsts.WorkloadKindDeployment,
		}},
		Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}
	libraries := []v1alpha1.InstrumentationLibraryGlobalId{{
		Name:     "spring-webmvc",
		SpanKind: common.ServerSpanKind,
		Language: common.JavaProgrammingLanguage,
	}}

	setFakeOdigosClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:                 "original rule",
			Notes:                    "original notes",
			Scopes:                   scopes,
			InstrumentationLibraries: &libraries,
			PayloadCollection:        &instrumentationrules.PayloadCollection{HttpRequest: &instrumentationrules.HttpPayloadCollection{}},
		},
	})

	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		// Older UI clients omit sourcesScopes and instrumentationLibraries on edit.
		PayloadCollection: &model.PayloadCollectionInput{HTTPRequest: &model.HTTPPayloadCollectionInput{}},
	})
	require.NoError(t, err)

	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, scopes, updatedRule.Spec.Scopes)
	require.Equal(t, &libraries, updatedRule.Spec.InstrumentationLibraries)
	require.Equal(t, ruleName, updatedRule.Spec.RuleName)
	require.Equal(t, notes, updatedRule.Spec.Notes)
}

func TestUpdateInstrumentationRuleAllowsExplicitClearingScopesAndLibraries(t *testing.T) {
	ctx := context.Background()
	ruleID := "scoped-rule"
	ruleName := "cluster wide rule"
	notes := "clear selectors"
	disabled := false

	libraries := []v1alpha1.InstrumentationLibraryGlobalId{{
		Name:     "spring-webmvc",
		SpanKind: common.ServerSpanKind,
		Language: common.JavaProgrammingLanguage,
	}}

	setFakeOdigosClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName: "original rule",
			Scopes: &k8sconsts.SourcesScopes{
				Namespaces: []string{"prod"},
			},
			InstrumentationLibraries: &libraries,
		},
	})

	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName:                 &ruleName,
		Notes:                    &notes,
		Disabled:                 &disabled,
		SourcesScopes:            []*model.InstrumentationRuleSourcesScopeInput{},
		InstrumentationLibraries: []*model.InstrumentationLibraryGlobalIDInput{},
	})
	require.NoError(t, err)

	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, updatedRule.Spec.Scopes)
	require.NotNil(t, updatedRule.Spec.InstrumentationLibraries)
	require.Empty(t, *updatedRule.Spec.InstrumentationLibraries)
}
