package instrumentation_instance

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

func testPod() *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-pod",
			Namespace: "default",
			UID:       "11111111-1111-1111-1111-111111111111",
		},
	}
}

func newTestClient(t *testing.T, pod *corev1.Pod) (client.Client, *runtime.Scheme) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, odigosv1.AddToScheme(scheme))
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(pod).
		WithStatusSubresource(&odigosv1.InstrumentationInstance{}).
		Build()
	return c, scheme
}

// Native SDK agents report the pid from their own container's pid namespace, so two containers of
// the same pod commonly both report pid 1. Each must still get its own InstrumentationInstance,
// otherwise one container's health silently overwrites the other's and one of them appears to the
// user as having no instrumented process at all.
func TestUpdateInstrumentationInstanceStatus_TwoContainersSamePid(t *testing.T) {
	ctx := context.Background()
	pod := testPod()
	c, scheme := newTestClient(t, pod)

	healthy := true
	unhealthy := false
	emptyMessage := ""
	failureMessage := "failed to load agent"

	require.NoError(t, UpdateInstrumentationInstanceStatus(ctx, pod, "nodejs", c, "deployment-my-app", 1, scheme,
		WithHealthy(&healthy, "LoadedSuccessfully", &emptyMessage)))
	require.NoError(t, UpdateInstrumentationInstanceStatus(ctx, pod, "python", c, "deployment-my-app", 1, scheme,
		WithHealthy(&unhealthy, "FailedToLoad", &failureMessage)))

	var instances odigosv1.InstrumentationInstanceList
	require.NoError(t, c.List(ctx, &instances, client.InNamespace(pod.Namespace)))
	require.Len(t, instances.Items, 2, "both containers shared a single InstrumentationInstance")

	byContainer := map[string]odigosv1.InstrumentationInstance{}
	for _, instance := range instances.Items {
		byContainer[instance.Spec.ContainerName] = instance
	}

	nodejs, ok := byContainer["nodejs"]
	require.True(t, ok)
	require.NotNil(t, nodejs.Status.Healthy)
	assert.True(t, *nodejs.Status.Healthy, "the python container's failure was reported on the nodejs container")

	python, ok := byContainer["python"]
	require.True(t, ok)
	require.NotNil(t, python.Status.Healthy)
	assert.False(t, *python.Status.Healthy)
	assert.Equal(t, failureMessage, python.Status.Message)
}

// A clean exit of one container deletes only its own instance; the other container is still running
// and must keep reporting.
func TestDeleteInstrumentationInstance_OnlyDeletesItsOwnContainer(t *testing.T) {
	ctx := context.Background()
	pod := testPod()
	c, scheme := newTestClient(t, pod)

	healthy := true
	emptyMessage := ""
	require.NoError(t, UpdateInstrumentationInstanceStatus(ctx, pod, "nodejs", c, "deployment-my-app", 1, scheme,
		WithHealthy(&healthy, "LoadedSuccessfully", &emptyMessage)))
	require.NoError(t, UpdateInstrumentationInstanceStatus(ctx, pod, "python", c, "deployment-my-app", 1, scheme,
		WithHealthy(&healthy, "LoadedSuccessfully", &emptyMessage)))

	require.NoError(t, DeleteInstrumentationInstance(ctx, pod, "python", c, 1))

	var instances odigosv1.InstrumentationInstanceList
	require.NoError(t, c.List(ctx, &instances, client.InNamespace(pod.Namespace)))
	require.Len(t, instances.Items, 1, "deleting one container's instance also removed the other container's")
	assert.Equal(t, "nodejs", instances.Items[0].Spec.ContainerName)
}

func TestInstrumentationInstanceNameIsUniquePerContainer(t *testing.T) {
	assert.NotEqual(t,
		InstrumentationInstanceName("my-pod", "nodejs", 1),
		InstrumentationInstanceName("my-pod", "python", 1),
	)
	assert.Equal(t, "my-pod-nodejs-1", InstrumentationInstanceName("my-pod", "nodejs", 1))
}
