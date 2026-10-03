package graph

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// RequireOnprem is the licensing gate in front of every instrumentation-rule mutation. Reading it
// wrongly either locks paying customers out of the feature or hands it to community installations,
// and neither shows up as an error anywhere.

const onpremRefusalMessage = "this feature is only supported for onprem installations"

// onpremCluster installs a cache client holding the given odigos-deployment ConfigMap data and
// restores the package global afterwards.
func onpremCluster(t *testing.T, objects ...client.Object) {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, wuOdigosNamespace)

	previous := kube.CacheClient
	kube.CacheClient = fake.NewClientBuilder().WithScheme(wuScheme(t)).WithObjects(objects...).Build()
	t.Cleanup(func() { kube.CacheClient = previous })
}

func onpremDeploymentConfigMap(namespace string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: k8sconsts.OdigosDeploymentConfigMapName, Namespace: namespace},
		Data:       data,
	}
}

func TestRequireOnpremAdmitsAnOnpremInstallation(t *testing.T) {
	onpremCluster(t, onpremDeploymentConfigMap(wuOdigosNamespace, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.OnPremOdigosTier),
	}))

	assert.NoError(t, RequireOnprem(context.Background()))
}

func TestRequireOnpremRefusesEveryNonOnpremTier(t *testing.T) {
	tests := []struct {
		name string
		data map[string]string
	}{
		{name: "community tier", data: map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.CommunityOdigosTier)}},
		// a key that is present but empty must not be mistaken for onprem
		{name: "empty tier", data: map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: ""}},
		{name: "tier key absent", data: map[string]string{"SOME_OTHER_KEY": string(common.OnPremOdigosTier)}},
		{name: "no data at all", data: nil},
		// a value that merely contains "onprem" is not the onprem tier
		{name: "lookalike tier", data: map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: "onprem-trial"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			onpremCluster(t, onpremDeploymentConfigMap(wuOdigosNamespace, tt.data))

			err := RequireOnprem(context.Background())
			require.Error(t, err)
			assert.EqualError(t, err, onpremRefusalMessage)
		})
	}
}

// An unreadable cluster must not be reported as "not onprem": that turns a transient API failure
// into a licensing message the user cannot act on.
func TestRequireOnpremReportsAMissingDeploymentConfigMapAsTheUnderlyingError(t *testing.T) {
	onpremCluster(t)

	err := RequireOnprem(context.Background())

	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "expected the kubernetes error to be propagated, got %v", err)
	assert.NotEqual(t, onpremRefusalMessage, err.Error())
}

// The tier is read from the odigos namespace named by CURRENT_NS; the same ConfigMap sitting in
// another namespace must not grant access.
func TestRequireOnpremReadsTheDeploymentConfigMapFromTheOdigosNamespace(t *testing.T) {
	onpremCluster(t, onpremDeploymentConfigMap("some-other-namespace", map[string]string{
		k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.OnPremOdigosTier),
	}))

	err := RequireOnprem(context.Background())

	require.Error(t, err)
	assert.True(t, apierrors.IsNotFound(err), "expected a not-found error, got %v", err)
}

// `odigos-deployment` and `effective-config` are two same-shaped ConfigMaps in the same namespace.
// Reading the wrong one compiles and, with a single-ConfigMap fixture, would even pass.
func TestRequireOnpremReadsTheDeploymentConfigMapNotTheEffectiveConfig(t *testing.T) {
	onpremCluster(t,
		onpremDeploymentConfigMap(wuOdigosNamespace, map[string]string{
			k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.CommunityOdigosTier),
		}),
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosEffectiveConfigName, Namespace: wuOdigosNamespace},
			Data:       map[string]string{k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.OnPremOdigosTier)},
		},
	)

	assert.EqualError(t, RequireOnprem(context.Background()), onpremRefusalMessage)
}

// The ConfigMap name, the key and the tier value are all written by the helm chart and the CLI and
// read back here, with nothing linking the two sides at compile time. Pin the wire literals.
func TestTheOnpremTierIsReadFromTheLiteralsTheInstallerWrites(t *testing.T) {
	assert.Equal(t, "odigos-deployment", k8sconsts.OdigosDeploymentConfigMapName)
	assert.Equal(t, "ODIGOS_TIER", k8sconsts.OdigosDeploymentConfigMapTierKey)
	assert.Equal(t, "onprem", string(common.OnPremOdigosTier))

	onpremCluster(t, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "odigos-deployment", Namespace: wuOdigosNamespace},
		Data:       map[string]string{"ODIGOS_TIER": "onprem"},
	})

	assert.NoError(t, RequireOnprem(context.Background()))
}

func onpremRuleInput() model.InstrumentationRuleInput {
	ruleName, notes, disabled := "payload collection", "", false
	return model.InstrumentationRuleInput{RuleName: &ruleName, Notes: &notes, Disabled: &disabled}
}

// onpremRuleMutationResults holds the outcome of every resolver guarded by RequireOnprem. The
// three return different shapes, so each needs its own assertion that a refusal was honoured —
// a shared "did it error" check cannot see `DeleteInstrumentationRule` returning true alongside
// the refusal.
type onpremRuleMutationResults struct {
	created *model.InstrumentationRule
	updated *model.InstrumentationRule
	deleted bool
	errs    map[string]error
}

func onpremRuleMutations(ctx context.Context) onpremRuleMutationResults {
	mutation := (&Resolver{}).Mutation()

	created, createErr := mutation.CreateInstrumentationRule(ctx, onpremRuleInput())
	updated, updateErr := mutation.UpdateInstrumentationRule(ctx, "some-rule", onpremRuleInput())
	deleted, deleteErr := mutation.DeleteInstrumentationRule(ctx, "some-rule")

	return onpremRuleMutationResults{
		created: created,
		updated: updated,
		deleted: deleted,
		errs:    map[string]error{"create": createErr, "update": updateErr, "delete": deleteErr},
	}
}

func TestEveryInstrumentationRuleMutationIsRefusedOnACommunityInstallation(t *testing.T) {
	onpremCluster(t, onpremDeploymentConfigMap(wuOdigosNamespace, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.CommunityOdigosTier),
	}))

	odigosClient := odigosfake.NewSimpleClientset()
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: odigosClient.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	got := onpremRuleMutations(context.Background())

	require.Len(t, got.errs, 3)
	for name, err := range got.errs {
		require.Error(t, err, name)
		assert.EqualError(t, err, onpremRefusalMessage, name)
	}
	assert.Nil(t, got.created, "create must not return a rule when it is refused")
	assert.Nil(t, got.updated, "update must not return a rule when it is refused")
	assert.False(t, got.deleted, "delete must not report success when it is refused")
	assert.Empty(t, odigosClient.Actions(), "a refused mutation must not touch the cluster")
}

// The complementary half: without it, a gate that refuses unconditionally would pass the test
// above. On an onprem installation the mutations must reach the service layer.
func TestTheInstrumentationRuleMutationsReachTheClusterOnAnOnpremInstallation(t *testing.T) {
	onpremCluster(t, onpremDeploymentConfigMap(wuOdigosNamespace, map[string]string{
		k8sconsts.OdigosDeploymentConfigMapTierKey: string(common.OnPremOdigosTier),
	}))

	odigosClient := odigosfake.NewSimpleClientset()
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{OdigosClient: odigosClient.OdigosV1alpha1()})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	got := onpremRuleMutations(context.Background())

	for name, err := range got.errs {
		if err != nil {
			assert.NotEqual(t, onpremRefusalMessage, err.Error(), name)
		}
	}
	assert.NotNil(t, got.created, "create must go through on an onprem installation")

	verbs := make([]string, 0, len(odigosClient.Actions()))
	for _, action := range odigosClient.Actions() {
		verbs = append(verbs, action.GetVerb())
	}
	assert.Subset(t, verbs, []string{"create", "get", "delete"},
		"each gated mutation must have reached the instrumentation rule client")
}
