package services

import (
	"context"
	"errors"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stesting "k8s.io/client-go/testing"
)

func storedRecommendation(t *testing.T, name string) *v1alpha1.Recommendation {
	t.Helper()
	rec, err := kube.DefaultClient.OdigosClient.Recommendations(recommendationsTestNamespace).
		Get(context.Background(), name, metav1.GetOptions{})
	require.NoError(t, err)
	return rec
}

func TestGetRecommendations_ConvertsEveryRecommendationInTheOdigosNamespace(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)

	inAnotherNamespace := recommendationCR("url-templatization", common.RecommendationTypeUrlTemplatization)
	inAnotherNamespace.Namespace = "some-other-namespace"
	setFakeRecommendationClient(t,
		recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes),
		dismissed(recommendationCR("infer-db-attributes", common.RecommendationTypeInferDBAttributes)),
		inAnotherNamespace,
	)

	got, err := GetRecommendations(context.Background())

	require.NoError(t, err)
	require.Len(t, got, 2, "only recommendations in the odigos namespace may be returned")

	byName := map[string]*model.Recommendation{}
	for _, rec := range got {
		byName[rec.Name] = rec
	}
	require.Contains(t, byName, "sample-health-probes")
	require.Contains(t, byName, "infer-db-attributes")
	assert.False(t, byName["sample-health-probes"].Dismissed)
	assert.True(t, byName["infer-db-attributes"].Dismissed)
}

// recommendations is a GraphQL non-null list, so a cluster with no Recommendation CRs has to
// render as [] rather than null.
func TestGetRecommendationsWithNoRecommendationsReturnsAnEmptyList(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)

	got, err := GetRecommendations(context.Background())

	require.NoError(t, err)
	assert.NotNil(t, got)
	assert.Empty(t, got)
}

func TestGetRecommendations_ARecommendationMissingFromTheCatalogFailsTheWholeQuery(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t,
		recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes),
		recommendationCR("left-over-recommendation", "TypeTheCatalogDoesNotKnow"),
	)

	_, err := GetRecommendations(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "left-over-recommendation")
}

func TestGetRecommendations_ListFailureIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	clientset := setFakeRecommendationClient(t)
	clientset.PrependReactor("list", "recommendations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("the api server said no")
	})

	_, err := GetRecommendations(context.Background())

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get recommendations")
	assert.Contains(t, err.Error(), "the api server said no")
}

func TestSetRecommendationDismissed_DismissAddsTheLabelTheAutoscalerFiltersOn(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t, recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes))

	got, err := SetRecommendationDismissed(context.Background(), "sample-health-probes", true)

	require.NoError(t, err)
	assert.True(t, got.Dismissed, "the returned model must reflect the new state")
	// Pinned as a literal: the label is a cross-component contract, written here and read by
	// the autoscaler's recommendation sync.
	assert.Equal(t, "true",
		storedRecommendation(t, "sample-health-probes").Labels["odigos.io/recommendation-dismissed"])
}

func TestSetRecommendationDismissed_RestoreRemovesTheLabel(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t, dismissed(recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes)))

	got, err := SetRecommendationDismissed(context.Background(), "sample-health-probes", false)

	require.NoError(t, err)
	assert.False(t, got.Dismissed)
	assert.NotContains(t, storedRecommendation(t, "sample-health-probes").Labels,
		k8sconsts.RecommendationDismissedLabel)
}

func TestSetRecommendationDismissed_KeepsLabelsItDoesNotOwn(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	rec := recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes)
	rec.Labels = map[string]string{"app.kubernetes.io/managed-by": "helm"}
	setFakeRecommendationClient(t, rec)
	ctx := context.Background()

	_, err := SetRecommendationDismissed(ctx, "sample-health-probes", true)
	require.NoError(t, err)
	assert.Equal(t, "helm", storedRecommendation(t, "sample-health-probes").Labels["app.kubernetes.io/managed-by"])

	_, err = SetRecommendationDismissed(ctx, "sample-health-probes", false)
	require.NoError(t, err)
	assert.Equal(t, "helm", storedRecommendation(t, "sample-health-probes").Labels["app.kubernetes.io/managed-by"])
}

// A recommendation that was never dismissed has no labels at all, so restoring it must not
// dereference the missing map.
func TestSetRecommendationDismissed_RestoringAnUnlabelledRecommendationIsANoop(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t, recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes))

	got, err := SetRecommendationDismissed(context.Background(), "sample-health-probes", false)

	require.NoError(t, err)
	assert.False(t, got.Dismissed)
	assert.Empty(t, storedRecommendation(t, "sample-health-probes").Labels)
}

func TestSetRecommendationDismissed_MissingRecommendationIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	setFakeRecommendationClient(t)

	_, err := SetRecommendationDismissed(context.Background(), "sample-health-probes", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `failed to get recommendation "sample-health-probes"`)
}

func TestSetRecommendationDismissed_UpdateFailureIsReported(t *testing.T) {
	useRecommendationsNamespace(t)
	loadRecommendationCatalog(t)
	clientset := setFakeRecommendationClient(t,
		recommendationCR("sample-health-probes", common.RecommendationTypeSampleHealthProbes))
	clientset.PrependReactor("update", "recommendations", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("conflict on update")
	})

	_, err := SetRecommendationDismissed(context.Background(), "sample-health-probes", true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `failed to update recommendation "sample-health-probes" dismissed state`)
	assert.Contains(t, err.Error(), "conflict on update")
}
