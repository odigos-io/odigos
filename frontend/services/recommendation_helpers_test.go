package services

import (
	"context"
	"testing"

	odigosfake "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/fake"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/recommendations"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"
)

// Deliberately not the default namespace, so that code reading the namespace from a
// hardcoded literal rather than from the environment fails these tests.
const recommendationsTestNamespace = "odigos-recommendations-test"

func useRecommendationsNamespace(t *testing.T) {
	t.Helper()
	t.Setenv(consts.CurrentNamespaceEnvVar, recommendationsTestNamespace)
}

// loadRecommendationCatalog loads the shipped manifests into the package-level catalog
// that GetByType/GetByK8sObjectName read, and returns every entry.
func loadRecommendationCatalog(t *testing.T) []recommendations.Recommendation {
	t.Helper()
	require.NoError(t, recommendations.Load())
	catalog := recommendations.Get()
	require.NotEmpty(t, catalog, "the shipped recommendation catalog must not be empty")
	return catalog
}

func recommendationCatalogEntry(t *testing.T, recType common.RecommendationType) recommendations.Recommendation {
	t.Helper()
	loadRecommendationCatalog(t)
	entry, ok := recommendations.GetByType(recType)
	require.True(t, ok, "recommendation %q must exist in the shipped catalog", recType)
	return entry
}

func recommendationCR(name string, recType common.RecommendationType) *v1alpha1.Recommendation {
	return &v1alpha1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: recommendationsTestNamespace},
		Spec: v1alpha1.RecommendationSpec{
			Type:          recType,
			Applied:       true,
			ConditionsMet: true,
		},
	}
}

func dismissed(rec *v1alpha1.Recommendation) *v1alpha1.Recommendation {
	rec.Labels = map[string]string{k8sconsts.RecommendationDismissedLabel: "true"}
	return rec
}

// setFakeRecommendationClient points kube.DefaultClient at a fake odigos clientset holding
// the given Recommendation and Action objects, and returns it so tests can inspect actions.
func setFakeRecommendationClient(t *testing.T, objects ...runtime.Object) *odigosfake.Clientset {
	t.Helper()

	clientset := odigosfake.NewSimpleClientset(objects...)
	previous := kube.DefaultClient
	kube.SetDefaultClient(&kube.Client{
		Interface:    k8sfake.NewSimpleClientset(),
		OdigosClient: clientset.OdigosV1alpha1(),
	})
	t.Cleanup(func() { kube.SetDefaultClient(previous) })

	return clientset
}

func storedActions(t *testing.T) []v1alpha1.Action {
	t.Helper()
	list, err := kube.DefaultClient.OdigosClient.Actions(recommendationsTestNamespace).
		List(context.Background(), metav1.ListOptions{})
	require.NoError(t, err)
	return list.Items
}

// recommendationsConfigClient builds the controller-runtime client ApplyRecommendationRemediation
// writes config steps through. The odigos-configuration ConfigMap is always present because
// upsertLocalUiConfig reads it to own the ConfigMap it creates.
func recommendationsConfigClient(t *testing.T, objects ...client.Object) client.Client {
	t.Helper()

	owner := &v1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name:      consts.OdigosConfigurationName,
		Namespace: recommendationsTestNamespace,
		UID:       types.UID("odigos-configuration-uid"),
	}}

	return fake.NewClientBuilder().
		WithScheme(newScheme()).
		WithObjects(append([]client.Object{owner}, objects...)...).
		Build()
}

func recommendationsLocalUiConfigMap(t *testing.T, cfg *common.OdigosConfiguration) client.Object {
	t.Helper()

	data, err := yaml.Marshal(cfg)
	require.NoError(t, err)

	return &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: consts.OdigosLocalUiConfigName, Namespace: recommendationsTestNamespace},
		Data:       map[string]string{consts.OdigosConfigurationFileName: string(data)},
	}
}

func storedLocalUiConfigMap(t *testing.T, c client.Client) *v1.ConfigMap {
	t.Helper()

	var cm v1.ConfigMap
	require.NoError(t, c.Get(context.Background(),
		types.NamespacedName{Namespace: recommendationsTestNamespace, Name: consts.OdigosLocalUiConfigName}, &cm))
	return &cm
}

func storedLocalUiConfiguration(t *testing.T, c client.Client) *common.OdigosConfiguration {
	t.Helper()

	cm := storedLocalUiConfigMap(t, c)
	var cfg common.OdigosConfiguration
	require.NoError(t, yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg))
	return &cfg
}
