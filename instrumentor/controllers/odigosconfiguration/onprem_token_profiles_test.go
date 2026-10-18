package odigosconfiguration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// The two profiles an on-prem token grants in these tests. Neither ships a yaml manifest, so
// the reconcile never needs the dynamic client, and allow_concurrent_agents carries a
// ModifyConfigFunc whose effect on the effective config is observable.
const (
	tpTokenProfile  = common.ProfileName("java-native-instrumentations")
	tpEffectProfile = common.ProfileName("allow_concurrent_agents")
	// a profile set in helm values rather than granted by the token
	tpConfiguredProfile = common.ProfileName("pod-manifest-env-var-injection")
)

// tpGrantedProfiles is the exact encoding instrumentor/controllers/odigospro publishes for a
// token granting these two profiles - see
// TestProfilesStringUsesTheSeparatorTheEntitlementReaderSplitsOn, which pins the writing half.
const tpGrantedProfiles = string(tpTokenProfile) + ", " + string(tpEffectProfile)

func tpScheme() *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := odigosv1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}
	return scheme
}

func tpConfigMap(ns string, name string, cfg *common.OdigosConfiguration) *corev1.ConfigMap {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		panic(err)
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
}

// tpDeploymentConfigMap is odigos-deployment as the odigospro controller leaves it after
// reading an on-prem token. Pass an empty tokenProfiles to omit the key entirely, which is
// what a token granting no profiles looks like.
func tpDeploymentConfigMap(ns string, tokenProfiles string) *corev1.ConfigMap {
	data := map[string]string{k8sconsts.OdigosDeploymentConfigMapVersionKey: "v1.2.3"}
	if tokenProfiles != "" {
		data[k8sconsts.OdigosDeploymentConfigMapOnPremClientProfilesKey] = tokenProfiles
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: k8sconsts.OdigosDeploymentConfigMapName},
		Data:       data,
	}
}

func tpReconciler(tier common.OdigosTier, objs ...client.Object) *odigosConfigurationController {
	scheme := tpScheme()
	return &odigosConfigurationController{
		Client:        fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build(),
		Scheme:        scheme,
		Tier:          tier,
		OdigosVersion: "v1.2.3",
	}
}

func tpEffectiveConfig(t *testing.T, c client.Client, ns string) common.OdigosConfiguration {
	t.Helper()
	cm := &corev1.ConfigMap{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Namespace: ns, Name: consts.OdigosEffectiveConfigName}, cm))
	var cfg common.OdigosConfiguration
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg))
	return cfg
}

// The profiles an on-prem token grants are carried through the cluster as a single
// odigos-deployment ConfigMap value, written by instrumentor/controllers/odigospro and split
// back apart here. Nothing in the type system connects the two: if either side changes its
// separator, every profile but the first silently stops being effective, and the customer
// loses the features the token pays for with no error anywhere.
func TestOnPremTokenProfilesReachTheEffectiveConfig(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := tpReconciler(common.OnPremOdigosTier,
		tpConfigMap(ns, consts.OdigosConfigurationName, &common.OdigosConfiguration{}),
		tpDeploymentConfigMap(ns, tpGrantedProfiles),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	effective := tpEffectiveConfig(t, r.Client, ns)
	assert.ElementsMatch(t, []common.ProfileName{tpTokenProfile, tpEffectProfile}, effective.Profiles)
	// the second granted profile is the one a separator mismatch drops, and its only
	// observable effect is this field
	require.NotNil(t, effective.AllowConcurrentAgents, "%s was not applied to the effective config", tpEffectProfile)
	assert.True(t, *effective.AllowConcurrentAgents)
}

// The token grant is still filtered by the tier: a community install presented with the same
// odigos-deployment document must not become entitled to on-prem profiles.
func TestOnPremTokenProfilesAreFilteredByTier(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := tpReconciler(common.CommunityOdigosTier,
		tpConfigMap(ns, consts.OdigosConfigurationName, &common.OdigosConfiguration{}),
		tpDeploymentConfigMap(ns, tpGrantedProfiles),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	effective := tpEffectiveConfig(t, r.Client, ns)
	assert.NotContains(t, effective.Profiles, tpTokenProfile, "%s is an on-prem only profile", tpTokenProfile)
	// allow_concurrent_agents is a community profile, so the token still grants it; proving
	// that keeps this test from passing just because the whole grant was dropped
	assert.Contains(t, effective.Profiles, tpEffectProfile)
}

// Token profiles are merged with the profiles set in helm values rather than replacing them,
// and a profile name that no longer exists has to be dropped instead of failing the reconcile.
func TestTokenProfilesAreMergedWithTheConfiguredProfiles(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := tpReconciler(common.OnPremOdigosTier,
		tpConfigMap(ns, consts.OdigosConfigurationName, &common.OdigosConfiguration{
			Profiles: []common.ProfileName{tpConfiguredProfile, "a-profile-that-was-removed"},
		}),
		tpDeploymentConfigMap(ns, string(tpTokenProfile)),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	effective := tpEffectiveConfig(t, r.Client, ns)
	assert.ElementsMatch(t, []common.ProfileName{tpConfiguredProfile, tpTokenProfile}, effective.Profiles)
}

// A community install, or an on-prem token granting no profiles, leaves the key out
// altogether. Reading it unconditionally would add a profile named "" to the list.
func TestEffectiveProfilesWithoutTheTokenProfilesKey(t *testing.T) {
	ns := env.GetCurrentNamespace()

	r := tpReconciler(common.OnPremOdigosTier,
		tpConfigMap(ns, consts.OdigosConfigurationName, &common.OdigosConfiguration{
			Profiles: []common.ProfileName{tpConfiguredProfile},
		}),
		tpDeploymentConfigMap(ns, ""),
	)

	_, err := r.Reconcile(context.Background(), ctrl.Request{})
	require.NoError(t, err)

	effective := tpEffectiveConfig(t, r.Client, ns)
	assert.Equal(t, []common.ProfileName{tpConfiguredProfile}, effective.Profiles)
}
