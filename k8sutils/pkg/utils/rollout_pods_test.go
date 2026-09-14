package utils

import (
	"context"
	"errors"
	"testing"

	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func boolPtr(b bool) *bool {
	return &b
}

func healthyPod(name string, labels map[string]string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: rolloutTestNamespace,
			Labels:    labels,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:         "app",
				Ready:        true,
				Started:      boolPtr(true),
				RestartCount: 0,
			}},
		},
	}
}

func podList(pods ...*corev1.Pod) *corev1.PodList {
	list := &corev1.PodList{}
	for _, pod := range pods {
		list.Items = append(list.Items, *pod)
	}
	return list
}

// failingListClient returns a clientset whose pod List always fails, which is how the caller of
// VerifyAllPodsAreRunning learns that it must retry rather than treat the workload as healthy.
func failingListClient(listErr error) kubernetes.Interface {
	client := k8sfake.NewClientset()
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, listErr
	})
	return client
}

// unreachableClient fails the test if any pod is listed, pinning the short circuits that happen
// before the API is consulted.
func unreachableClient(t *testing.T) kubernetes.Interface {
	t.Helper()
	client := k8sfake.NewClientset()
	client.PrependReactor("list", "pods", func(action k8stesting.Action) (bool, runtime.Object, error) {
		t.Errorf("pods were listed even though the check should have short circuited")
		return true, nil, nil
	})
	return client
}

func podListActions(client *k8sfake.Clientset) []k8stesting.ListActionImpl {
	var listActions []k8stesting.ListActionImpl
	for _, action := range client.Actions() {
		if list, ok := action.(k8stesting.ListActionImpl); ok && list.Resource.Resource == "pods" {
			listActions = append(listActions, list)
		}
	}
	return listActions
}

func TestCheckAllPodsRunning(t *testing.T) {
	matchLabels := map[string]string{"app": "frontend"}

	tests := []struct {
		name     string
		pods     *corev1.PodList
		expected bool
	}{
		{
			name:     "a workload with no pods has nothing pending",
			pods:     podList(),
			expected: true,
		},
		{
			name:     "every pod is running and ready",
			pods:     podList(healthyPod("frontend-1", matchLabels), healthyPod("frontend-2", matchLabels)),
			expected: true,
		},
		{
			name: "a pod that is not running yet",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.Phase = corev1.PodPending
				return podList(pod)
			}(),
			expected: false,
		},
		{
			name: "a pod that has restarted",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.ContainerStatuses[0].RestartCount = 1
				return podList(pod)
			}(),
			expected: false,
		},
		{
			name: "a pod whose container is not ready",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.ContainerStatuses[0].Ready = false
				return podList(pod)
			}(),
			expected: false,
		},
		{
			name: "a pod whose container has not started",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.ContainerStatuses[0].Started = boolPtr(false)
				return podList(pod)
			}(),
			expected: false,
		},
		{
			name: "a pod reporting no container statuses",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.ContainerStatuses = nil
				return podList(pod)
			}(),
			expected: false,
		},
		{
			name: "the scan does not stop at the first healthy pod",
			pods: func() *corev1.PodList {
				broken := healthyPod("frontend-2", matchLabels)
				broken.Status.Phase = corev1.PodPending
				return podList(healthyPod("frontend-1", matchLabels), broken)
			}(),
			expected: false,
		},
		{
			name: "a restart in a second container of the same pod still counts",
			pods: func() *corev1.PodList {
				pod := healthyPod("frontend-1", matchLabels)
				pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
					Name:         "sidecar",
					Ready:        true,
					Started:      boolPtr(true),
					RestartCount: 2,
				})
				return podList(pod)
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, checkAllPodsRunning(tt.pods))
		})
	}
}

func TestVerifyAllPodsAreRunning(t *testing.T) {
	matchLabels := map[string]string{"app": "frontend"}

	t.Run("a finished rollout with healthy pods", func(t *testing.T) {
		client := k8sfake.NewClientset(healthyPod("frontend-1", matchLabels), healthyPod("frontend-2", matchLabels))

		running, err := VerifyAllPodsAreRunning(context.Background(), client, doneDeployment())

		assert.NoError(t, err)
		assert.True(t, running)
	})

	t.Run("a finished rollout with an unhealthy pod", func(t *testing.T) {
		pending := healthyPod("frontend-2", matchLabels)
		pending.Status.Phase = corev1.PodPending
		client := k8sfake.NewClientset(healthyPod("frontend-1", matchLabels), pending)

		running, err := VerifyAllPodsAreRunning(context.Background(), client, doneDeployment())

		assert.NoError(t, err)
		assert.False(t, running)
	})

	t.Run("an unfinished rollout is reported without listing pods", func(t *testing.T) {
		deployment := doneDeployment()
		deployment.Status.UpdatedReplicas = 1

		running, err := VerifyAllPodsAreRunning(context.Background(), unreachableClient(t), deployment)

		assert.NoError(t, err)
		assert.False(t, running)
	})

	t.Run("a workload with no selector is assumed healthy", func(t *testing.T) {
		deployment := doneDeployment()
		deployment.Spec.Selector = &metav1.LabelSelector{}

		running, err := VerifyAllPodsAreRunning(context.Background(), unreachableClient(t), deployment)

		assert.NoError(t, err)
		assert.True(t, running)
	})

	t.Run("a list failure is surfaced rather than reported as healthy", func(t *testing.T) {
		listErr := errors.New("the apiserver is unavailable")

		running, err := VerifyAllPodsAreRunning(context.Background(), failingListClient(listErr), doneDeployment())

		assert.Error(t, err)
		assert.Contains(t, err.Error(), listErr.Error())
		assert.False(t, running)
	})

	// A selector that is too wide would let an unrelated unhealthy pod block the rollout, and a
	// selector that ignores the namespace would do the same across the cluster.
	t.Run("only the pods selected by the workload are inspected", func(t *testing.T) {
		otherWorkloadPod := healthyPod("cart-1", map[string]string{"app": "cart"})
		otherWorkloadPod.Status.Phase = corev1.PodPending

		otherNamespacePod := healthyPod("frontend-1", matchLabels)
		otherNamespacePod.Namespace = "other-namespace"
		otherNamespacePod.Status.Phase = corev1.PodPending

		client := k8sfake.NewClientset(healthyPod("frontend-2", matchLabels), otherWorkloadPod, otherNamespacePod)

		running, err := VerifyAllPodsAreRunning(context.Background(), client, doneDeployment())

		assert.NoError(t, err)
		assert.True(t, running)
	})

	t.Run("the workload selector is sent to the apiserver", func(t *testing.T) {
		client := k8sfake.NewClientset()

		_, err := VerifyAllPodsAreRunning(context.Background(), client, doneStatefulSet())
		assert.NoError(t, err)

		listActions := podListActions(client)
		assert.Len(t, listActions, 1)
		assert.Equal(t, "app=cart", listActions[0].GetListRestrictions().Labels.String())
		assert.Equal(t, rolloutTestNamespace, listActions[0].Namespace)
	})

	t.Run("every supported workload kind is listed by its own selector", func(t *testing.T) {
		workloads := map[string]struct {
			obj              metav1.Object
			expectedSelector string
		}{
			"deployment":       {obj: doneDeployment(), expectedSelector: "app=frontend"},
			"statefulset":      {obj: doneStatefulSet(), expectedSelector: "app=cart"},
			"daemonset":        {obj: doneDaemonSet(), expectedSelector: "app=log-shipper"},
			"deploymentconfig": {obj: doneDeploymentConfig(), expectedSelector: "app=billing"},
			"argo rollout":     {obj: doneArgoRollout(), expectedSelector: "app=search"},
		}

		for name, workload := range workloads {
			t.Run(name, func(t *testing.T) {
				client := k8sfake.NewClientset()

				running, err := VerifyAllPodsAreRunning(context.Background(), client, workload.obj)

				assert.NoError(t, err)
				assert.True(t, running)
				listActions := podListActions(client)
				assert.Len(t, listActions, 1)
				assert.Equal(t, workload.expectedSelector, listActions[0].GetListRestrictions().Labels.String())
			})
		}
	})
}

func TestIsTerminating(t *testing.T) {
	deletedAt := metav1.NewTime(metav1.Now().Time)

	tests := []struct {
		name     string
		obj      *appsv1.Deployment
		expected bool
	}{
		{
			name:     "a live object",
			obj:      doneDeployment(),
			expected: false,
		},
		{
			name: "an object being deleted",
			obj: func() *appsv1.Deployment {
				d := doneDeployment()
				d.DeletionTimestamp = &deletedAt
				return d
			}(),
			expected: true,
		},
		{
			name: "a zero deletion timestamp does not mean terminating",
			obj: func() *appsv1.Deployment {
				d := doneDeployment()
				d.DeletionTimestamp = &metav1.Time{}
				return d
			}(),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsTerminating(tt.obj))
		})
	}
}
