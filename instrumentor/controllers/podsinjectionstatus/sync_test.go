package podsinjectionstatus

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	podsInjectionStatus "github.com/odigos-io/odigos/status/instrumentationconfig/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testAgentsMetaHash = "agents-meta-hash"

func TestSyncWorkloadIgnoresMatchingPodsInOtherNamespaces(t *testing.T) {
	scheme := newPodsInjectionStatusTestScheme(t)
	ctx := context.Background()

	targetNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target"}}
	otherNamespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "other"}}
	deployment := newPodsInjectionStatusTestDeployment(targetNamespace.Name, "shared-app")
	ic := newPodsInjectionStatusTestInstrumentationConfig(deployment)
	otherNamespacePod := newPodsInjectionStatusTestPod(otherNamespace.Name, "other-pod", map[string]string{
		"app.kubernetes.io/name":            "shared-app",
		k8sconsts.OdigosAgentsMetaHashLabel: testAgentsMetaHash,
	})

	c := newPodsInjectionStatusTestClient(scheme,
		targetNamespace,
		otherNamespace,
		newEffectiveConfigMap(),
		deployment,
		ic,
		otherNamespacePod,
	)

	err := syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})
	require.NoError(t, err)

	updated := getPodsInjectionStatusTestInstrumentationConfig(t, ctx, c, ic.Namespace, ic.Name)
	require.NotNil(t, updated.Status.PodsManifestInjectionStatus)
	assert.False(t, updated.Status.PodsManifestInjectionStatus.HasInjectedUpToDatePods)
	assert.False(t, updated.Status.PodsManifestInjectionStatus.HasInjectedOutOfDatePods)
	assert.False(t, updated.Status.PodsManifestInjectionStatus.HasUninjectedPods)

	condition := meta.FindStatusCondition(updated.Status.Conditions, podsInjectionStatus.PodsInjectionType)
	require.NotNil(t, condition)
	assert.Equal(t, string(podsInjectionStatus.PodsInjectionReasonNoRunningPods), condition.Reason)
}

func TestSyncWorkloadUsesFullLabelSelector(t *testing.T) {
	scheme := newPodsInjectionStatusTestScheme(t)
	ctx := context.Background()

	namespace := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "target"}}
	deployment := newPodsInjectionStatusTestDeployment(namespace.Name, "expression-app")
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "app",
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{"expression-app"},
			},
		},
	}
	ic := newPodsInjectionStatusTestInstrumentationConfig(deployment)
	matchingPod := newPodsInjectionStatusTestPod(namespace.Name, "matching-pod", map[string]string{
		"app":                               "expression-app",
		k8sconsts.OdigosAgentsMetaHashLabel: testAgentsMetaHash,
	})
	unrelatedPod := newPodsInjectionStatusTestPod(namespace.Name, "unrelated-pod", map[string]string{
		"app": "unrelated",
	})

	c := newPodsInjectionStatusTestClient(scheme,
		namespace,
		newEffectiveConfigMap(),
		deployment,
		ic,
		matchingPod,
		unrelatedPod,
	)

	err := syncWorkload(ctx, c, k8sconsts.PodWorkload{
		Name:      deployment.Name,
		Namespace: deployment.Namespace,
		Kind:      k8sconsts.WorkloadKindDeployment,
	})
	require.NoError(t, err)

	updated := getPodsInjectionStatusTestInstrumentationConfig(t, ctx, c, ic.Namespace, ic.Name)
	require.NotNil(t, updated.Status.PodsManifestInjectionStatus)
	assert.True(t, updated.Status.PodsManifestInjectionStatus.HasInjectedUpToDatePods)
	assert.False(t, updated.Status.PodsManifestInjectionStatus.HasInjectedOutOfDatePods)
	assert.False(t, updated.Status.PodsManifestInjectionStatus.HasUninjectedPods)

	condition := meta.FindStatusCondition(updated.Status.Conditions, podsInjectionStatus.PodsInjectionType)
	require.NotNil(t, condition)
	assert.Equal(t, string(podsInjectionStatus.PodsInjectionReasonPodsInjectedSuccessfully), condition.Reason)
}

func newPodsInjectionStatusTestScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	require.NoError(t, appsv1.AddToScheme(scheme))
	return scheme
}

func newPodsInjectionStatusTestClient(scheme *runtime.Scheme, objects ...client.Object) client.WithWatch {
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&odigosv1.InstrumentationConfig{}).
		WithObjects(objects...).
		Build()
}

func newPodsInjectionStatusTestDeployment(namespace, name string) *appsv1.Deployment {
	deployment := testutil.NewMockTestDeployment(&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}, name)
	deployment.Spec.Template.Labels = deployment.Spec.Selector.MatchLabels
	return deployment
}

func newPodsInjectionStatusTestInstrumentationConfig(deployment *appsv1.Deployment) *odigosv1.InstrumentationConfig {
	ic := testutil.NewMockInstrumentationConfig(deployment)
	ic.Spec.AgentsMetaHash = testAgentsMetaHash
	return ic
}

func newPodsInjectionStatusTestPod(namespace, name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}

func newEffectiveConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosEffectiveConfigName,
			Namespace: consts.DefaultOdigosNamespace,
		},
	}
}

func getPodsInjectionStatusTestInstrumentationConfig(t *testing.T, ctx context.Context, c client.Client, namespace, name string) *odigosv1.InstrumentationConfig {
	t.Helper()

	updated := &odigosv1.InstrumentationConfig{}
	err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, updated)
	require.NoError(t, err)
	return updated
}
