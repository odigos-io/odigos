package services

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
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
			MimeTypes:           graphql.OmittableOf([]*string{}),
			MaxPayloadLength:    graphql.OmittableOf(&newMax),
			DropPartialPayloads: graphql.OmittableOf(&newDrop),
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Empty(t, *out.HttpRequest.MimeTypes)
	require.Equal(t, int64(4096), *out.HttpRequest.MaxPayloadLength)
	require.False(t, *out.HttpRequest.DropPartialPayloads)
	require.Nil(t, out.HttpResponse, "omitted payload sections should still be disabled")
}

func TestMergePayloadCollectionUpdateExplicitNullClearsAdvancedOptions(t *testing.T) {
	maxHTTP := int64(2048)
	dropHTTP := true
	maxDb := int64(512)
	dropDb := true
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
		HTTPRequest: &model.HTTPPayloadCollectionInput{
			MimeTypes:           graphql.OmittableOf[[]*string](nil),
			MaxPayloadLength:    graphql.OmittableOf[*int](nil),
			DropPartialPayloads: graphql.OmittableOf[*bool](nil),
		},
		DbQuery: &model.DbQueryPayloadCollectionInput{
			MaxPayloadLength:    graphql.OmittableOf[*int](nil),
			DropPartialPayloads: graphql.OmittableOf[*bool](nil),
		},
		Messaging: &model.MessagingPayloadCollectionInput{
			MaxPayloadLength:    graphql.OmittableOf[*int](nil),
			DropPartialPayloads: graphql.OmittableOf[*bool](nil),
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Nil(t, out.HttpRequest.MimeTypes, "null mimeTypes means all MIME types")
	require.Nil(t, out.HttpRequest.MaxPayloadLength, "null maxPayloadLength means no limit")
	require.Nil(t, out.HttpRequest.DropPartialPayloads)

	require.NotNil(t, out.DbQuery)
	require.Nil(t, out.DbQuery.MaxPayloadLength)
	require.Nil(t, out.DbQuery.DropPartialPayloads)

	require.NotNil(t, out.Messaging)
	require.Nil(t, out.Messaging.MaxPayloadLength)
	require.Nil(t, out.Messaging.DropPartialPayloads)
}

func TestMergePayloadCollectionUpdateMixesOmittedNullAndValues(t *testing.T) {
	oldMax := int64(2048)
	oldDrop := true
	oldMimeTypes := []string{"application/json"}
	newDbMax := 256

	existing := &apirules.PayloadCollection{
		HttpResponse: &apirules.HttpPayloadCollection{
			MimeTypes:           &oldMimeTypes,
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
		DbQuery: &apirules.DbQueryPayloadCollection{
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
		Messaging: &apirules.MessagingPayloadCollection{
			MaxPayloadLength:    &oldMax,
			DropPartialPayloads: &oldDrop,
		},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		// mimeTypes omitted (kept), limit cleared, drop-partial omitted (kept).
		HTTPResponse: &model.HTTPPayloadCollectionInput{
			MaxPayloadLength: graphql.OmittableOf[*int](nil),
		},
		// limit replaced, drop-partial cleared.
		DbQuery: &model.DbQueryPayloadCollectionInput{
			MaxPayloadLength:    graphql.OmittableOf(&newDbMax),
			DropPartialPayloads: graphql.OmittableOf[*bool](nil),
		},
		// limit cleared, drop-partial omitted (kept).
		Messaging: &model.MessagingPayloadCollectionInput{
			MaxPayloadLength: graphql.OmittableOf[*int](nil),
		},
	})

	require.NotNil(t, out.HttpResponse)
	require.Equal(t, []string{"application/json"}, *out.HttpResponse.MimeTypes)
	require.Nil(t, out.HttpResponse.MaxPayloadLength)
	require.True(t, *out.HttpResponse.DropPartialPayloads)

	require.NotNil(t, out.DbQuery)
	require.Equal(t, int64(256), *out.DbQuery.MaxPayloadLength)
	require.Nil(t, out.DbQuery.DropPartialPayloads)

	require.NotNil(t, out.Messaging)
	require.Nil(t, out.Messaging.MaxPayloadLength)
	require.True(t, *out.Messaging.DropPartialPayloads)
}

func TestMergePayloadCollectionUpdateNewSectionMapsOmittedToNil(t *testing.T) {
	maxPayload := int64(2048)
	existing := &apirules.PayloadCollection{
		DbQuery: &apirules.DbQueryPayloadCollection{MaxPayloadLength: &maxPayload},
	}

	// A section that wasn't enabled before has nothing to keep, so omitted and
	// null both leave the option unset.
	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{
			MaxPayloadLength: graphql.OmittableOf[*int](nil),
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Nil(t, out.HttpRequest.MimeTypes)
	require.Nil(t, out.HttpRequest.MaxPayloadLength)
	require.Nil(t, out.HttpRequest.DropPartialPayloads)
	require.Nil(t, out.DbQuery, "omitted payload sections should still be disabled")
}

func TestFromHTTPPayloadInputMimeTypes(t *testing.T) {
	tests := []struct {
		name      string
		mimeTypes graphql.Omittable[[]*string]
		want      *[]string
	}{
		{
			name: "omitted means all MIME types",
			want: nil,
		},
		{
			name:      "null means all MIME types",
			mimeTypes: graphql.OmittableOf[[]*string](nil),
			want:      nil,
		},
		{
			name:      "explicit empty list is kept",
			mimeTypes: graphql.OmittableOf([]*string{}),
			want:      &[]string{},
		},
		{
			name:      "values are kept",
			mimeTypes: graphql.OmittableOf([]*string{StringPtr("application/json"), StringPtr("text/plain")}),
			want:      &[]string{"application/json", "text/plain"},
		},
		{
			name:      "blank and null entries are dropped",
			mimeTypes: graphql.OmittableOf([]*string{StringPtr(""), StringPtr("application/json"), StringPtr("  "), nil}),
			want:      &[]string{"application/json"},
		},
		{
			name:      "only blank entries means all MIME types",
			mimeTypes: graphql.OmittableOf([]*string{StringPtr(""), StringPtr("")}),
			want:      nil,
		},
		{
			name:      "only whitespace and null entries means all MIME types",
			mimeTypes: graphql.OmittableOf([]*string{StringPtr(" \t"), nil}),
			want:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := fromHTTPPayloadInput(&model.HTTPPayloadCollectionInput{MimeTypes: tt.mimeTypes})
			require.Equal(t, tt.want, cfg.MimeTypes)
		})
	}
}

func TestMergePayloadCollectionUpdateBlankMimeTypesClearFilter(t *testing.T) {
	oldMimeTypes := []string{"application/json"}
	existing := &apirules.PayloadCollection{
		HttpRequest: &apirules.HttpPayloadCollection{MimeTypes: &oldMimeTypes},
	}

	out := mergePayloadCollectionUpdate(existing, &model.PayloadCollectionInput{
		HTTPRequest: &model.HTTPPayloadCollectionInput{
			MimeTypes: graphql.OmittableOf([]*string{StringPtr(""), StringPtr("")}),
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Nil(t, out.HttpRequest.MimeTypes, "a list of blank entries must not stop HTTP payload collection")
}

func TestGetPayloadCollectionInputMapsOmittedAndNullToNil(t *testing.T) {
	maxPayload := 1024

	out := getPayloadCollectionInput(model.InstrumentationRuleInput{
		PayloadCollection: &model.PayloadCollectionInput{
			HTTPRequest: &model.HTTPPayloadCollectionInput{
				MimeTypes:        graphql.OmittableOf[[]*string](nil),
				MaxPayloadLength: graphql.OmittableOf(&maxPayload),
			},
			DbQuery: &model.DbQueryPayloadCollectionInput{
				MaxPayloadLength: graphql.OmittableOf[*int](nil),
			},
			Messaging: &model.MessagingPayloadCollectionInput{},
		},
	})

	require.NotNil(t, out.HttpRequest)
	require.Nil(t, out.HttpRequest.MimeTypes)
	require.Equal(t, int64(1024), *out.HttpRequest.MaxPayloadLength)
	require.Nil(t, out.HttpRequest.DropPartialPayloads)
	require.Nil(t, out.HttpResponse)

	require.NotNil(t, out.DbQuery)
	require.Nil(t, out.DbQuery.MaxPayloadLength)
	require.Nil(t, out.DbQuery.DropPartialPayloads)

	require.NotNil(t, out.Messaging)
	require.Nil(t, out.Messaging.MaxPayloadLength)
	require.Nil(t, out.Messaging.DropPartialPayloads)
}

func TestUpdateInstrumentationRuleClearsPayloadLimitOnExplicitNull(t *testing.T) {
	ctx := context.Background()
	ruleID := "payload-rule"
	ruleName := "payload rule"
	notes := ""
	disabled := false
	drop := true

	maxPayload := int64(2048)
	mimeTypes := []string{"application/json", "text/plain"}
	setFakeOdigosInstrumentationRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName: ruleName,
			PayloadCollection: &apirules.PayloadCollection{
				HttpRequest: &apirules.HttpPayloadCollection{
					MimeTypes:           &mimeTypes,
					MaxPayloadLength:    &maxPayload,
					DropPartialPayloads: &drop,
				},
			},
		},
	})

	// The edit form removes the limit by sending maxPayloadLength: null alongside the unchanged options.
	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		PayloadCollection: &model.PayloadCollectionInput{
			HTTPRequest: &model.HTTPPayloadCollectionInput{
				MimeTypes:           graphql.OmittableOf([]*string{StringPtr("application/json"), StringPtr("text/plain")}),
				MaxPayloadLength:    graphql.OmittableOf[*int](nil),
				DropPartialPayloads: graphql.OmittableOf(&drop),
			},
		},
	})
	require.NoError(t, err)

	updated, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.NotNil(t, updated.Spec.PayloadCollection)
	require.NotNil(t, updated.Spec.PayloadCollection.HttpRequest)
	require.Nil(t, updated.Spec.PayloadCollection.HttpRequest.MaxPayloadLength)
	require.Equal(t, []string{"application/json", "text/plain"}, *updated.Spec.PayloadCollection.HttpRequest.MimeTypes)
	require.True(t, *updated.Spec.PayloadCollection.HttpRequest.DropPartialPayloads)
}

func TestUpdateInstrumentationRulePreservesOmittedFields(t *testing.T) {
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
	headers := &apirules.HttpHeadersCollection{HeaderKeys: []string{"Authorization", "X-Request-Id"}}
	custom := &apirules.CustomInstrumentations{
		Java: []apirules.JavaCustomProbe{{ClassName: "com.example.Service", MethodName: "handle"}},
	}

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
			HeadersCollection:        headers,
			CustomInstrumentations:   custom,
			NetworkMetrics:           &apirules.NetworkMetricsConfig{},
			PayloadCollection:        &apirules.PayloadCollection{HttpRequest: &apirules.HttpPayloadCollection{}},
		},
	})

	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		// Metadata-only / partial clients omit selectors and type payloads.
	})
	require.NoError(t, err)

	updated, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "renamed rule", updated.Spec.RuleName)
	require.Equal(t, scopes, updated.Spec.Scopes)
	require.Equal(t, &libraries, updated.Spec.InstrumentationLibraries)
	require.Equal(t, headers, updated.Spec.HeadersCollection)
	require.Equal(t, custom, updated.Spec.CustomInstrumentations)
	require.NotNil(t, updated.Spec.NetworkMetrics)
	require.NotNil(t, updated.Spec.PayloadCollection)
	require.NotNil(t, updated.Spec.PayloadCollection.HttpRequest)
}

func TestUpdateInstrumentationRuleClearsScopesOnExplicitEmpty(t *testing.T) {
	ctx := context.Background()
	ruleID := "scoped-rule"
	ruleName := "rule"
	notes := "notes"
	disabled := false

	setFakeOdigosInstrumentationRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleID,
			Namespace: consts.DefaultOdigosNamespace,
		},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName: "rule",
			Scopes: &k8sconsts.SourcesScopes{
				Namespaces: []string{"payments"},
			},
			HeadersCollection: &apirules.HttpHeadersCollection{HeaderKeys: []string{"Authorization"}},
		},
	})

	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName:      &ruleName,
		Notes:         &notes,
		Disabled:      &disabled,
		SourcesScopes: graphql.OmittableOf([]*model.InstrumentationRuleSourcesScopeInput{}),
	})
	require.NoError(t, err)

	updated, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, updated.Spec.Scopes)
	require.Equal(t, []string{"Authorization"}, updated.Spec.HeadersCollection.HeaderKeys)
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
		SourcesScopes:            graphql.OmittableOf([]*model.InstrumentationRuleSourcesScopeInput{}),
		InstrumentationLibraries: graphql.OmittableOf([]*model.InstrumentationLibraryGlobalIDInput{}),
	})
	require.NoError(t, err)

	updatedRule, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, updatedRule.Spec.Scopes)
	require.NotNil(t, updatedRule.Spec.InstrumentationLibraries)
	require.Empty(t, *updatedRule.Spec.InstrumentationLibraries)
}

func TestUpdateInstrumentationRuleExplicitNullWidensSelectors(t *testing.T) {
	ctx := context.Background()
	ruleID := "scoped-rule"
	ruleName := "rule"
	notes := "notes"
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
			RuleName: "rule",
			Scopes: &k8sconsts.SourcesScopes{
				Namespaces: []string{"payments"},
			},
			InstrumentationLibraries: &libraries,
			HeadersCollection:        &apirules.HttpHeadersCollection{HeaderKeys: []string{"Authorization"}},
		},
	})

	// The edit form sends null for "Entire Cluster", and null type payloads that don't apply to the rule.
	_, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName:                 &ruleName,
		Notes:                    &notes,
		Disabled:                 &disabled,
		SourcesScopes:            graphql.OmittableOf[[]*model.InstrumentationRuleSourcesScopeInput](nil),
		InstrumentationLibraries: graphql.OmittableOf[[]*model.InstrumentationLibraryGlobalIDInput](nil),
		HeadersCollection:        nil,
	})
	require.NoError(t, err)

	updated, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, updated.Spec.Scopes)
	require.Nil(t, updated.Spec.InstrumentationLibraries)
	require.Equal(t, []string{"Authorization"}, updated.Spec.HeadersCollection.HeaderKeys)
}
