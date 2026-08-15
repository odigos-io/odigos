package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	apirules "github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

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

// libraryScopedRule mirrors the documented payload collection example: a rule scoped to specific
// instrumentation libraries, which can only be authored through kubectl/GitOps since the UI has no
// control for the field.
func libraryScopedRule(name string) *v1alpha1.InstrumentationRule {
	maxPayloadLength := int64(1024)
	disabledTraces := true

	return &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: env.GetCurrentNamespace(),
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName: "collect db payloads for database/sql only",
			InstrumentationLibraries: &[]v1alpha1.InstrumentationLibraryGlobalId{
				{Name: "database/sql", Language: common.GoProgrammingLanguage, SpanKind: common.ClientSpanKind},
			},
			PayloadCollection: &apirules.PayloadCollection{
				DbQuery: &apirules.DbQueryPayloadCollection{MaxPayloadLength: &maxPayloadLength},
			},
			TraceConfig: &apirules.TraceConfig{Disabled: &disabledTraces},
		},
	}
}

func setFakeOdigosClient(t *testing.T, objects ...runtime.Object) {
	t.Helper()

	previous := kube.DefaultClient
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	fakeClient := odigosfake.NewSimpleClientset(objects...)
	kube.SetDefaultClient(&kube.Client{OdigosClient: fakeClient.OdigosV1alpha1()})
}

// The UI never selects instrumentationLibraries in its queries and never sends it back, so treating
// an omitted value as "clear" silently drops the library scope on the first save from the UI.
func TestUpdateInstrumentationRuleKeepsInstrumentationLibrariesWhenInputOmitsThem(t *testing.T) {
	setFakeOdigosClient(t, libraryScopedRule("library-scoped-rule"))

	ruleName := "collect db payloads for database/sql only"
	notes := "edited from the UI"
	disabled := false
	maxPayloadLength := 2048

	// exactly the fields @odigos/ui-kit loads into the rule form and sends back on save
	_, err := UpdateInstrumentationRule(context.Background(), "library-scoped-rule", model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		PayloadCollection: &model.PayloadCollectionInput{
			DbQuery: &model.DbQueryPayloadCollectionInput{MaxPayloadLength: &maxPayloadLength},
		},
	})
	require.NoError(t, err)

	stored, err := kube.DefaultClient.OdigosClient.InstrumentationRules(env.GetCurrentNamespace()).
		Get(context.Background(), "library-scoped-rule", metav1.GetOptions{})
	require.NoError(t, err)

	require.NotNil(t, stored.Spec.InstrumentationLibraries, "UI save must not drop the library scope")
	require.Equal(t, []v1alpha1.InstrumentationLibraryGlobalId{
		{Name: "database/sql", Language: common.GoProgrammingLanguage, SpanKind: common.ClientSpanKind},
	}, *stored.Spec.InstrumentationLibraries)

	require.Equal(t, int64(2048), *stored.Spec.PayloadCollection.DbQuery.MaxPayloadLength)
	require.True(t, *stored.Spec.TraceConfig.Disabled)
}

func TestUpdateInstrumentationRuleReplacesInstrumentationLibrariesWhenInputSetsThem(t *testing.T) {
	setFakeOdigosClient(t, libraryScopedRule("library-scoped-rule"))

	ruleName := "collect db payloads for net/http only"
	notes := ""
	disabled := false
	spanKind := model.SpanKindServer
	language := model.ProgrammingLanguageGo

	_, err := UpdateInstrumentationRule(context.Background(), "library-scoped-rule", model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		InstrumentationLibraries: []*model.InstrumentationLibraryGlobalIDInput{
			{Name: "net/http", SpanKind: &spanKind, Language: &language},
		},
	})
	require.NoError(t, err)

	stored, err := kube.DefaultClient.OdigosClient.InstrumentationRules(env.GetCurrentNamespace()).
		Get(context.Background(), "library-scoped-rule", metav1.GetOptions{})
	require.NoError(t, err)

	require.Len(t, *stored.Spec.InstrumentationLibraries, 1)
	require.Equal(t, "net/http", (*stored.Spec.InstrumentationLibraries)[0].Name)
}
