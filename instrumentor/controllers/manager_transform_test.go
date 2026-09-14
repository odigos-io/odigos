package controllers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func transformTestPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "checkout-7d4c8b5f9b-p2xzq",
			Namespace: "payments",
			ManagedFields: []metav1.ManagedFieldsEntry{
				{Manager: "kubelet", Operation: metav1.ManagedFieldsOperationUpdate},
			},
		},
		Spec: corev1.PodSpec{
			NodeName:   "node-running-agents",
			Containers: []corev1.Container{{Name: "checkout", Image: "checkout:1.0"}},
		},
		Status: corev1.PodStatus{
			Phase:   corev1.PodRunning,
			Message: "started",
		},
	}
}

// The instrumented-nodes controllers list pods by node through a field index on
// spec.nodeName, which reads whatever the cache holds; dropping the node name
// here leaves every node unlabeled and odiglet unscheduled.
func TestPodTransformFunc_KeepsTheNodeNameOfAScheduledPod(t *testing.T) {
	transformed, err := podTransformFunc(transformTestPod())

	require.NoError(t, err)
	pod, isPod := transformed.(*corev1.Pod)
	require.True(t, isPod)
	assert.Equal(t, "node-running-agents", pod.Spec.NodeName)
}

func TestPodTransformFunc_DropsTheRestOfTheSpecAndTheManagedFields(t *testing.T) {
	transformed, err := podTransformFunc(transformTestPod())

	require.NoError(t, err)
	pod := transformed.(*corev1.Pod)
	assert.Empty(t, pod.Spec.Containers, "container specs are not used and dominate the cache size")
	assert.Empty(t, pod.ManagedFields)
	assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
	assert.Equal(t, "started", pod.Status.Message)
}

func TestPodTransformFunc_KeepsTheWholeSpecOfAStaticPod(t *testing.T) {
	staticPod := transformTestPod()
	staticPod.Annotations = map[string]string{"kubernetes.io/config.source": "file"}
	staticPod.OwnerReferences = []metav1.OwnerReference{{
		APIVersion: "v1",
		Kind:       "Node",
		Name:       "node-running-agents",
	}}

	transformed, err := podTransformFunc(staticPod)

	require.NoError(t, err)
	pod := transformed.(*corev1.Pod)
	assert.Equal(t, "node-running-agents", pod.Spec.NodeName)
	assert.Len(t, pod.Spec.Containers, 1, "a static pod is its own workload, so its spec is needed")
}

func TestPodTransformFunc_AnAlreadyTransformedPodIsReturnedAsIs(t *testing.T) {
	once, err := podTransformFunc(transformTestPod())
	require.NoError(t, err)

	twice, err := podTransformFunc(once)

	require.NoError(t, err)
	assert.Same(t, once, twice)
}

func TestPodTransformFunc_RejectsAnObjectThatIsNotAPod(t *testing.T) {
	transformed, err := podTransformFunc(&corev1.Node{})

	assert.ErrorContains(t, err, "expected a Pod")
	assert.Nil(t, transformed)
}
