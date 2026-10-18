package graph

import (
	"context"
	"testing"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/executor"
	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	apirules "github.com/odigos-io/odigos/common/api/instrumentationrules"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// The partial-update semantics only work if gqlgen's generated unmarshaller actually leaves an
// omitted variable field unset while recording an explicit null as set-with-a-nil-value. Building
// a graphql.Omittable by hand in a unit test assumes that; these tests prove it by running the
// mutations the UI sends through the real generated executor, with the only difference between
// the runs being whether the variables carry the key at all. Before #5954 the two were
// indistinguishable, and the "All Operations" / "Entire Cluster" edits were silently dropped.

// omitUseCacheClient points the informer-backed read client at a fake seeded with objects.
func omitUseCacheClient(t *testing.T, objects ...client.Object) {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, v1alpha1.AddToScheme(scheme))

	previous := kube.CacheClient
	kube.CacheClient = ctrlfake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
	t.Cleanup(func() { kube.CacheClient = previous })
}

// omitUseOdigosClient points the live write client at a fake clientset and returns it so the test
// can read back what was persisted.
func omitUseOdigosClient(t *testing.T, objects ...runtime.Object) *odigosfake.Clientset {
	t.Helper()

	clientset := odigosfake.NewSimpleClientset(objects...)
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: clientset.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })
	return clientset
}

func omitExecuteMutation(t *testing.T, query string, variables map[string]any) {
	t.Helper()

	exec := executor.New(NewExecutableSchema(Config{Resolvers: &Resolver{}}))
	ctx := graphql.StartOperationTrace(context.Background())
	operationContext, errs := exec.CreateOperationContext(ctx, &graphql.RawParams{Query: query, Variables: variables})
	require.Empty(t, errs)

	responses, ctx := exec.DispatchOperation(ctx, operationContext)
	require.Empty(t, responses(ctx).Errors)
}

const omitUpdateNoisyOperationMutation = `
mutation UpdateNoisyOperationRule($samplingId: ID!, $ruleId: ID!, $rule: NoisyOperationRuleInput!) {
  updateNoisyOperationRule(samplingId: $samplingId, ruleId: $ruleId, rule: $rule) {
    ruleId
  }
}`

// omitStoredSampling is a Sampling CR holding one scoped noisy-operation rule, the state a user
// reaches by narrowing a rule to a single route in one namespace.
func omitStoredSampling(namespace string) (*v1alpha1.Sampling, string) {
	percentage := 5.0
	rule := v1alpha1.NoisyOperation{
		Name:             "payments healthz",
		SourceScopes:     &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
		Operation:        &commonapisampling.HeadSamplingOperationMatcher{HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{Route: "/healthz"}},
		PercentageAtMost: &percentage,
		Notes:            "stored note",
	}
	return &v1alpha1.Sampling{
		ObjectMeta: metav1.ObjectMeta{Name: "default-sampling", Namespace: namespace},
		Spec: v1alpha1.SamplingSpec{
			Name:            "default-sampling",
			NoisyOperations: []v1alpha1.NoisyOperation{rule},
		},
	}, v1alpha1.ComputeNoisyOperationHash(&rule)
}

// omitUpdateStoredNoisyOperation seeds the same Sampling CR into both cluster clients (the rule
// update reads through the cache and writes through the live client) and returns the rule as it
// was persisted.
func omitUpdateStoredNoisyOperation(t *testing.T, ruleVariables map[string]any) v1alpha1.NoisyOperation {
	t.Helper()

	namespace := env.GetCurrentNamespace()
	cached, ruleID := omitStoredSampling(namespace)
	live, _ := omitStoredSampling(namespace)

	omitUseCacheClient(t, cached)
	liveClient := omitUseOdigosClient(t, live)

	omitExecuteMutation(t, omitUpdateNoisyOperationMutation, map[string]any{
		"samplingId": live.Name,
		"ruleId":     ruleID,
		"rule":       ruleVariables,
	})

	updated, err := liveClient.OdigosV1alpha1().Samplings(namespace).Get(context.Background(), live.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Len(t, updated.Spec.NoisyOperations, 1)
	return updated.Spec.NoisyOperations[0]
}

// What the UI sends when the user picks "All Operations" / "all sources" / "drop all": the keys
// are present with a null value.
func TestAnExplicitNullInTheMutationVariablesWidensTheStoredSamplingRule(t *testing.T) {
	got := omitUpdateStoredNoisyOperation(t, map[string]any{
		"name":             "payments healthz",
		"sourceScopes":     nil,
		"operation":        nil,
		"percentageAtMost": nil,
		"notes":            nil,
	})

	require.Nil(t, got.Operation, "a null operation widens the rule to all operations")
	require.Nil(t, got.SourceScopes, "a null sourceScopes widens the rule to all sources")
	require.Nil(t, got.PercentageAtMost, "a null percentageAtMost resets the rule to drop all")
	require.Empty(t, got.Notes, "a null note clears the note")
	require.Equal(t, "payments healthz", got.Name)
}

// What a partial client sends: only the field it is changing. Everything it leaves out keeps its
// stored value, so a bulk enable/disable cannot widen a scoped rule.
func TestOmittingAFieldFromTheMutationVariablesKeepsTheStoredSamplingRuleValue(t *testing.T) {
	_, ruleID := omitStoredSampling(env.GetCurrentNamespace())
	got := omitUpdateStoredNoisyOperation(t, map[string]any{"disabled": true})

	require.True(t, got.Disabled)
	require.NotNil(t, got.Operation)
	require.Equal(t, "/healthz", got.Operation.HttpServer.Route)
	require.Equal(t, &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}}, got.SourceScopes)
	require.Equal(t, 5.0, *got.PercentageAtMost)
	require.Equal(t, "stored note", got.Notes)
	require.Equal(t, "payments healthz", got.Name)
	// The rule id hashes the scope and the matcher, so an edit that touches neither must keep the
	// id the UI already holds addressable.
	require.Equal(t, ruleID, v1alpha1.ComputeNoisyOperationHash(&got))
}

const omitUpdateInstrumentationRuleMutation = `
mutation UpdateInstrumentationRule($ruleId: ID!, $rule: InstrumentationRuleInput!) {
  updateInstrumentationRule(ruleId: $ruleId, instrumentationRule: $rule) {
    ruleId
  }
}`

// The instrumentation rule input splits the two policies across one payload, and the UI sends both
// halves in the same mutation: null selectors mean "Entire Cluster" and must clear, while the type
// payloads the rule's type doesn't use also arrive as null and must survive untouched. Running
// them together is what proves the two are actually told apart rather than both happening to be
// handled the same way.
func TestANullSelectorClearsWhileANullTypePayloadSurvivesTheSameMutation(t *testing.T) {
	namespace := env.GetCurrentNamespace()
	libraries := []v1alpha1.InstrumentationLibraryGlobalId{{Name: "spring-webmvc"}}
	stored := &v1alpha1.InstrumentationRule{
		ObjectMeta: metav1.ObjectMeta{Name: "scoped-rule", Namespace: namespace},
		Spec: v1alpha1.InstrumentationRuleSpec{
			RuleName:                 "scoped rule",
			Scopes:                   &k8sconsts.SourcesScopes{Namespaces: []string{"payments"}},
			InstrumentationLibraries: &libraries,
			HeadersCollection:        &apirules.HttpHeadersCollection{HeaderKeys: []string{"Authorization"}},
			PayloadCollection:        &apirules.PayloadCollection{HttpRequest: &apirules.HttpPayloadCollection{}},
		},
	}

	// The mutation is behind the onprem gate, which reads the tier off this ConfigMap.
	omitUseCacheClient(t, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosDeploymentConfigMapName, Namespace: namespace},
		Data:       map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.OnPremOdigosTier)},
	})
	liveClient := omitUseOdigosClient(t, stored)

	omitExecuteMutation(t, omitUpdateInstrumentationRuleMutation, map[string]any{
		"ruleId": stored.Name,
		"rule": map[string]any{
			"ruleName":                 "cluster wide rule",
			"notes":                    "",
			"disabled":                 false,
			"sourcesScopes":            nil,
			"instrumentationLibraries": nil,
			"headersCollection":        nil,
			"payloadCollection":        nil,
		},
	})

	updated, err := liveClient.OdigosV1alpha1().InstrumentationRules(namespace).Get(context.Background(), stored.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Nil(t, updated.Spec.Scopes, "a null sourcesScopes widens the rule to the entire cluster")
	require.Nil(t, updated.Spec.InstrumentationLibraries, "a null instrumentationLibraries widens the rule to all libraries")
	require.Equal(t, []string{"Authorization"}, updated.Spec.HeadersCollection.HeaderKeys,
		"a null headersCollection must not drop the stored headers")
	require.NotNil(t, updated.Spec.PayloadCollection, "a null payloadCollection must not drop the stored payloads")
	require.NotNil(t, updated.Spec.PayloadCollection.HttpRequest)
}
