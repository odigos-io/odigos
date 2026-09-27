package podswebhook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const devicePluginTestNamespace = "odigos-test"

func newDeviceTestClient(objects ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objects...).Build()
}

func newOdigletDaemonSet(namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "odiglet",
			Namespace: namespace,
			Labels:    map[string]string{"app.kubernetes.io/name": "odiglet"},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app.kubernetes.io/name": "odiglet"},
			},
		},
	}
}

func newOdigletPod(namespace, name string, containerStatuses ...corev1.ContainerStatus) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    map[string]string{"app.kubernetes.io/name": "odiglet"},
		},
		Status: corev1.PodStatus{
			ContainerStatuses: containerStatuses,
		},
	}
}

func waitingContainerStatus(name, reason string) corev1.ContainerStatus {
	return corev1.ContainerStatus{
		Name: name,
		State: corev1.ContainerState{
			Waiting: &corev1.ContainerStateWaiting{Reason: reason},
		},
	}
}

func TestInjectDeviceToContainer(t *testing.T) {
	t.Run("initializes the resource lists and requests a single device", func(t *testing.T) {
		container := &corev1.Container{Name: "test"}

		InjectDeviceToContainer(container, "instrumentation.odigos.io/generic")

		expected := resource.MustParse("1")
		assert.Equal(t, expected, container.Resources.Limits["instrumentation.odigos.io/generic"])
		assert.Equal(t, expected, container.Resources.Requests["instrumentation.odigos.io/generic"])
	})

	t.Run("existing resources are preserved", func(t *testing.T) {
		container := &corev1.Container{
			Name: "test",
			Resources: corev1.ResourceRequirements{
				Limits:   corev1.ResourceList{"cpu": resource.MustParse("500m")},
				Requests: corev1.ResourceList{"memory": resource.MustParse("128Mi")},
			},
		}

		InjectDeviceToContainer(container, "instrumentation.odigos.io/generic")

		assert.Equal(t, resource.MustParse("500m"), container.Resources.Limits["cpu"])
		assert.Equal(t, resource.MustParse("128Mi"), container.Resources.Requests["memory"])
		assert.Equal(t, resource.MustParse("1"), container.Resources.Limits["instrumentation.odigos.io/generic"])
	})
}

func TestCheckDevicePluginContainersHealth(t *testing.T) {
	ctx := context.Background()

	t.Run("no odiglet daemonset", func(t *testing.T) {
		c := newDeviceTestClient()

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no odiglet daemonset")
	})

	t.Run("odiglet daemonset in another namespace is not considered", func(t *testing.T) {
		c := newDeviceTestClient(newOdigletDaemonSet("other-namespace"))

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no odiglet daemonset")
	})

	t.Run("multiple odiglet daemonsets", func(t *testing.T) {
		second := newOdigletDaemonSet(devicePluginTestNamespace)
		second.Name = "odiglet-second"
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), second)

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "multiple odiglet daemonsets")
	})

	t.Run("no odiglet pods", func(t *testing.T) {
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace))

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no odiglet pods found")
	})

	t.Run("healthy device plugin container", func(t *testing.T) {
		pod := newOdigletPod(devicePluginTestNamespace, "odiglet-abc", corev1.ContainerStatus{
			Name:  "deviceplugin",
			Ready: true,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		})
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), pod)

		require.NoError(t, CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace))
	})

	t.Run("device plugin in crash loop backoff", func(t *testing.T) {
		pod := newOdigletPod(devicePluginTestNamespace, "odiglet-abc",
			waitingContainerStatus("deviceplugin", "CrashLoopBackOff"))
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), pod)

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "crash loop backoff")
		assert.Contains(t, err.Error(), "odigos-test/odiglet-abc")
	})

	t.Run("device plugin in image pull backoff", func(t *testing.T) {
		pod := newOdigletPod(devicePluginTestNamespace, "odiglet-abc",
			waitingContainerStatus("deviceplugin", "ImagePullBackOff"))
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), pod)

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "image pull backoff")
	})

	t.Run("a container that is still initializing does not block injection", func(t *testing.T) {
		pod := newOdigletPod(devicePluginTestNamespace, "odiglet-abc",
			waitingContainerStatus("deviceplugin", "ContainerCreating"))
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), pod)

		require.NoError(t, CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace))
	})

	t.Run("backoff in another odiglet container does not block injection", func(t *testing.T) {
		pod := newOdigletPod(devicePluginTestNamespace, "odiglet-abc",
			waitingContainerStatus("odiglet", "CrashLoopBackOff"),
			corev1.ContainerStatus{
				Name:  "deviceplugin",
				Ready: true,
				State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
			})
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), pod)

		require.NoError(t, CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace))
	})

	t.Run("an unhealthy pod anywhere in the daemonset blocks injection", func(t *testing.T) {
		healthy := newOdigletPod(devicePluginTestNamespace, "odiglet-healthy", corev1.ContainerStatus{
			Name:  "deviceplugin",
			Ready: true,
			State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
		})
		unhealthy := newOdigletPod(devicePluginTestNamespace, "odiglet-unhealthy",
			waitingContainerStatus("deviceplugin", "CrashLoopBackOff"))
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), healthy, unhealthy)

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "odiglet-unhealthy")
	})

	t.Run("pods that do not match the daemonset selector are not considered", func(t *testing.T) {
		unrelated := newOdigletPod(devicePluginTestNamespace, "unrelated",
			waitingContainerStatus("deviceplugin", "CrashLoopBackOff"))
		unrelated.Labels = map[string]string{"app.kubernetes.io/name": "some-other-app"}
		c := newDeviceTestClient(newOdigletDaemonSet(devicePluginTestNamespace), unrelated)

		err := CheckDevicePluginContainersHealth(ctx, c, devicePluginTestNamespace)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "no odiglet pods found")
	})
}
