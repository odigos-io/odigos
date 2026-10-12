package recommendations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	recommendationcatalog "github.com/odigos-io/odigos/recommendations"
)

func existingRecommendation(rec recommendationcatalog.Recommendation) *odigosv1.Recommendation {
	return &odigosv1.Recommendation{
		ObjectMeta: metav1.ObjectMeta{Name: rec.K8sObjectName, Namespace: testNamespace},
		Spec:       odigosv1.RecommendationSpec{Type: rec.Type},
	}
}

func reconcileRecommendation(t *testing.T, c client.Client, name string) (ctrl.Result, error) {
	t.Helper()
	reconciler := &RecommendationReconciler{Client: c}
	return reconciler.Reconcile(testContext(), ctrl.Request{
		NamespacedName: types.NamespacedName{Namespace: testNamespace, Name: name},
	})
}

func recommendationExists(t *testing.T, c client.Client, name string) bool {
	t.Helper()
	err := c.Get(testContext(), types.NamespacedName{Namespace: testNamespace, Name: name}, &odigosv1.Recommendation{})
	if apierrors.IsNotFound(err) {
		return false
	}
	require.NoError(t, err)
	return true
}

// catalogRecommendations returns one oss and one enterprise only recommendation from the shipped
// catalog, so the reconciler is exercised with the names it sees in a real cluster.
func catalogRecommendations(t *testing.T) (oss, enterpriseOnly recommendationcatalog.Recommendation) {
	t.Helper()
	for _, rec := range loadCatalog(t) {
		if rec.OSS && oss.K8sObjectName == "" {
			oss = rec
		}
		if !rec.OSS && enterpriseOnly.K8sObjectName == "" {
			enterpriseOnly = rec
		}
	}
	require.NotEmpty(t, oss.K8sObjectName, "catalog should have an oss recommendation")
	require.NotEmpty(t, enterpriseOnly.K8sObjectName, "catalog should have an enterprise only recommendation")
	return oss, enterpriseOnly
}

func TestRecommendationReconciler(t *testing.T) {
	useTestNamespace(t)
	oss, enterpriseOnly := catalogRecommendations(t)

	// a recommendation that was removed from the catalog (for example after a downgrade) would
	// otherwise stay in the cluster forever and keep being shown in the ui.
	t.Run("deletes a recommendation that is not in the catalog", func(t *testing.T) {
		stale := &odigosv1.Recommendation{
			ObjectMeta: metav1.ObjectMeta{Name: "no-such-recommendation", Namespace: testNamespace},
			Spec:       odigosv1.RecommendationSpec{Type: common.RecommendationTypeSampleHealthProbes},
		}
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}), stale)

		result, err := reconcileRecommendation(t, c, stale.Name)

		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assert.False(t, recommendationExists(t, c, stale.Name))
		assert.Empty(t, c.applies)
	})

	t.Run("deletes an enterprise recommendation on the community tier", func(t *testing.T) {
		useTier(t, common.CommunityOdigosTier)
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}), existingRecommendation(enterpriseOnly))

		_, err := reconcileRecommendation(t, c, enterpriseOnly.K8sObjectName)

		require.NoError(t, err)
		assert.False(t, recommendationExists(t, c, enterpriseOnly.K8sObjectName))
	})

	t.Run("keeps an enterprise recommendation on an enterprise tier", func(t *testing.T) {
		useTier(t, common.OnPremOdigosTier)
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}), existingRecommendation(enterpriseOnly))

		_, err := reconcileRecommendation(t, c, enterpriseOnly.K8sObjectName)

		require.NoError(t, err)
		assert.True(t, recommendationExists(t, c, enterpriseOnly.K8sObjectName))
		assert.Equal(t, enterpriseOnly.Type, ptr.Deref(c.single(t).spec.Type, ""))
	})

	// the recommendation was deleted by a user, the reconciler recreates it from the catalog.
	t.Run("recreates a deleted recommendation from the catalog", func(t *testing.T) {
		useTier(t, common.CommunityOdigosTier)
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))

		_, err := reconcileRecommendation(t, c, oss.K8sObjectName)

		require.NoError(t, err)
		assert.True(t, recommendationExists(t, c, oss.K8sObjectName))
	})

	t.Run("a deleted recommendation that is not in the catalog is not an error", func(t *testing.T) {
		c := newRecordingClient(t, effectiveConfigMap(t, &common.OdigosConfiguration{}))

		result, err := reconcileRecommendation(t, c, "no-such-recommendation")

		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{}, result)
		assert.Empty(t, c.applies)
	})

	// the effective config is created by the scheduler, which may not have run yet when the
	// autoscaler starts. failing the reconcile instead of requeueing would rely on the controller
	// runtime backoff and log an error on every startup.
	t.Run("requeues while the effective config is missing", func(t *testing.T) {
		useTier(t, common.CommunityOdigosTier)
		c := newRecordingClient(t)

		result, err := reconcileRecommendation(t, c, oss.K8sObjectName)

		require.NoError(t, err)
		assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, result)
	})
}

func TestRecommendationsSyncReconciler_RequeuesWithoutEffectiveConfig(t *testing.T) {
	useTestNamespace(t)
	useTier(t, common.CommunityOdigosTier)
	loadCatalog(t)
	c := newRecordingClient(t)

	reconciler := &RecommendationsSyncReconciler{Client: c}
	result, err := reconciler.Reconcile(testContext(), ctrl.Request{})

	require.NoError(t, err)
	assert.Equal(t, ctrl.Result{Requeue: true, RequeueAfter: 5 * time.Second}, result)
	assert.Empty(t, c.applies)
}
