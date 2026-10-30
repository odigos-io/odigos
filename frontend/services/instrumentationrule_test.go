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
	custom := &apirules.CustomInstrumentationsWithMetadata{
		Java: []apirules.JavaCustomProbeWithMetadata{{
			JavaCustomProbe: apirules.JavaCustomProbe{ClassName: "com.example.Service", MethodName: "handle"},
		}},
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

func TestCustomInstrumentationsMetadataRoundTrip(t *testing.T) {
	notes := "called from checkout"
	addedAt := "2026-10-19T09:00:00Z"
	category := model.CustomProbeCategorySecurity
	className := "com.acme.Pricing"
	methodName := "calculate"
	signature := "SSL_write"

	got, err := getCustomInstrumentationsInput(model.InstrumentationRuleInput{
		CustomInstrumentations: &model.CustomInstrumentationsInput{
			Java: []*model.JavaCustomProbeInput{{
				ClassName:  &className,
				MethodName: &methodName,
				Notes:      &notes,
				Category:   &category,
				AddedAt:    &addedAt,
			}},
			Cpp: []*model.CppCustomProbeInput{{
				Signature: &signature,
				Notes:     &notes,
				Category:  &category,
				AddedAt:   &addedAt,
			}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "com.acme.Pricing", got.Java[0].ClassName)
	require.Equal(t, notes, got.Java[0].Notes)
	require.Equal(t, apirules.CustomProbeCategorySecurity, got.Java[0].Category)
	require.Equal(t, addedAt, got.Java[0].AddedAt)
	require.Equal(t, signature, got.Cpp[0].Signature)
	require.Equal(t, apirules.CustomProbeCategorySecurity, got.Cpp[0].Category)

	out := convertCustomInstrumentations(got)
	require.Equal(t, &notes, out.Java[0].Notes)
	require.Equal(t, &category, out.Java[0].Category)
	require.Equal(t, &addedAt, out.Java[0].AddedAt)
	require.Equal(t, &signature, out.Cpp[0].Signature)
	require.Equal(t, &category, out.Cpp[0].Category)
}

func TestPythonCustomInstrumentationsRoundTrip(t *testing.T) {
	notes := "slow checkout"
	module := "app.billing"
	qualname := "Invoice.calculate"
	probe := &model.PythonCustomProbeInput{Module: &module, Qualname: &qualname, Notes: &notes}

	got, err := getCustomInstrumentationsInput(model.InstrumentationRuleInput{
		CustomInstrumentations: &model.CustomInstrumentationsInput{
			Python: []*model.PythonCustomProbeInput{probe, probe},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []apirules.PythonCustomProbeWithMetadata{{
		PythonCustomProbe:   apirules.PythonCustomProbe{Module: module, Qualname: qualname},
		CustomProbeMetadata: apirules.CustomProbeMetadata{Notes: notes},
	}}, got.Python)

	out := convertCustomInstrumentations(got)
	require.Len(t, out.Python, 1)
	require.Equal(t, &module, out.Python[0].Module)
	require.Equal(t, &qualname, out.Python[0].Qualname)
	require.Equal(t, &notes, out.Python[0].Notes)

	_, err = getCustomInstrumentationsInput(model.InstrumentationRuleInput{
		CustomInstrumentations: &model.CustomInstrumentationsInput{
			Python: []*model.PythonCustomProbeInput{{Module: &module}},
		},
	})
	require.Error(t, err)
}

func TestCreateKernelTracesRule(t *testing.T) {
	ctx := context.Background()
	setFakeOdigosInstrumentationRuleClient(t)
	ruleName, notes, disabled, enabled, recordArgs := "kernel", "", false, true, true

	got, err := CreateInstrumentationRule(ctx, model.InstrumentationRuleInput{
		RuleName: &ruleName,
		Notes:    &notes,
		Disabled: &disabled,
		KernelTraces: &model.KernelTracesInput{
			Filesystem:   &model.KernelFilesystemTracesInput{},
			Subprocesses: &model.KernelSubprocessTracesInput{RecordCommandArgs: &recordArgs},
			CPUDelay:     &model.KernelCPUDelayTracesInput{Enabled: &disabled},
		},
	})
	require.NoError(t, err)
	require.Equal(t, model.InstrumentationRuleTypeKernelTraces, got.Type)
	require.Equal(t, &model.KernelTraces{
		Filesystem:   &model.KernelFilesystemTraces{Enabled: &enabled},
		Subprocesses: &model.KernelSubprocessTraces{RecordCommandArgs: &recordArgs},
	}, got.KernelTraces)

	rules, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, rules.Items, 1)
	require.Equal(t, &apirules.KernelTraces{
		Filesystem:   &apirules.KernelFilesystemTraces{},
		Subprocesses: &apirules.KernelSubprocessTraces{RecordCommandArgs: &recordArgs},
	}, rules.Items[0].Spec.KernelTraces)
}

func TestUpdateKernelTracesRuleReplacesAreas(t *testing.T) {
	ctx := context.Background()
	ruleID := "kernel-rule"
	setFakeOdigosInstrumentationRuleClient(t, &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{Name: ruleID, Namespace: consts.DefaultOdigosNamespace},
		Spec: v1alpha1.InstrumentationRuleSpec{
			KernelTraces: &apirules.KernelTraces{
				Filesystem:   &apirules.KernelFilesystemTraces{},
				Subprocesses: &apirules.KernelSubprocessTraces{},
			},
		},
	})
	ruleName, notes, disabled, enabled := "kernel", "", false, true

	got, err := UpdateInstrumentationRule(ctx, ruleID, model.InstrumentationRuleInput{
		RuleName:     &ruleName,
		Notes:        &notes,
		Disabled:     &disabled,
		KernelTraces: &model.KernelTracesInput{CPUDelay: &model.KernelCPUDelayTracesInput{Enabled: &enabled}},
	})
	require.NoError(t, err)
	require.Equal(t, model.InstrumentationRuleTypeKernelTraces, got.Type)
	require.Equal(t, &model.KernelTraces{CPUDelay: &model.KernelCPUDelayTraces{Enabled: &enabled}}, got.KernelTraces)

	updated, err := kube.DefaultClient.OdigosClient.InstrumentationRules(consts.DefaultOdigosNamespace).Get(ctx, ruleID, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, &apirules.KernelTraces{CPUDelay: &apirules.KernelCPUDelayTraces{}}, updated.Spec.KernelTraces)
}
