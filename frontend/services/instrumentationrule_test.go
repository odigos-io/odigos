package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	apirules "github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func setFakeOdigosInstrumentationRuleClient(t *testing.T, objects ...*v1alpha1.InstrumentationRule) {
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

func TestMergePayloadCollectionUpdatePreservesOmittedAdvancedOptions(t *testing.T) {
	maxHTTP := int64(2048)
	dropHTTP := true
	maxDb := int64(512)
	dropDb := false
	maxMessaging := int64(1024)
	dropMessaging := true
	mimeTypes := []string{"application/json", "text/plain"}

	existing := &apirules.PayloadCollection{
		HttpRequest: &apirules.HttpPayloadCollection{
			MimeTypes:           &mimeTypes,
			MaxPayloadLength:    &maxHTTP,
			DropPartialPayloads: &dropHTTP,
		},
		DbQuery: &apirules.DbQueryPayloadCollection{
			MaxPayloadLength:    &maxDb,
			DropPartialPayloads: &dropDb,
		},
		Messaging: &apirules.MessagingPayloadCollection{
			MaxPayloadLength:    &maxMessaging,
			DropPartialPayloads: &dropMessaging,
		},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{},
		DbQuery:     &model.DbQueryPayloadCollectionInput{},
		Messaging:   &model.MessagingPayloadCollectionInput{},
	})

	require.NotNil(t, out.HttpRequest)
	require.Equal(t, []string{"application/json", "text/plain"}, *out.HttpRequest.MimeTypes)
	require.Equal(t, int64(2048), *out.HttpRequest.MaxPayloadLength)
	require.True(t, *out.HttpRequest.DropPartialPayloads)
	require.NotSame(t, existing.HttpRequest.MimeTypes, out.HttpRequest.MimeTypes)

	require.NotNil(t, out.DbQuery)
	require.Equal(t, int64(512), *out.DbQuery.MaxPayloadLength)
	require.False(t, *out.DbQuery.DropPartialPayloads)

	require.NotNil(t, out.Messaging)
	require.Equal(t, int64(1024), *out.Messaging.MaxPayloadLength)
	require.True(t, *out.Messaging.DropPartialPayloads)
}

func TestMergePayloadCollectionUpdateReplacesExplicitAdvancedOptions(t *testing.T) {
	oldMax := int64(2048)
	oldDrop := true
	oldMimeTypes := []string{"application/json"}
	newMax := 4096
	newDrop := false

	existing := &apirules.PayloadCollection{
		HttpRequest: &apirules.HttpPayloadCollection{
			MimeTypes:           &oldMimeTypes,
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
		HttpResponse: &apirules.HttpPayloadCollection{
			MimeTypes:           &oldMimeTypes,
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{
			MimeTypes:           []*string{},
			MaxPayloadLength:    &newMax,
			DropPartialPayloads: &newDrop,
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Empty(t, *out.HttpRequest.MimeTypes)
	require.Equal(t, int64(4096), *out.HttpRequest.MaxPayloadLength)
	require.False(t, *out.HttpRequest.DropPartialPayloads)
	require.Nil(t, out.HttpResponse, "omitted payload sections should still be disabled")
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

	setFakeOdigosInstrumentationRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:                 "original rule",
			Notes:                    "original notes",
			Scopes:                   scopes,
			InstrumentationLibraries: &libraries,
			PayloadCollection:        &apirules.PayloadCollection{HttpRequest: &apirules.HttpPayloadCollection{}},
		},
	})

	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		// Older UI clients omit SourcesScopes and InstrumentationLibraries on edit.
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

	setFakeOdigosInstrumentationRuleClient(t, &v1alpha1.InstrumentationRule{
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
