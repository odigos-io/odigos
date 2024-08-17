package instrumentation_ebpf

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetPodWorkloadObject(t *testing.T) {
	pr := &PodsReconciler{}
	cases := []struct {
		name             string
		pod              *corev1.Pod
		expectedWorkload workload.PodWorkload
	}{
		{
			name: "pod in deployment",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "ReplicaSet",
							Name: "deployment-1234",
						},
					},
					Namespace: "default",
				},
			},
			expectedWorkload: workload.PodWorkload{
				Kind:      workload.WorkloadKindDeployment,
				Name:      "deployment",
				Namespace: "default",
			},
		},
		{
			name: "pod with hyphen in name of deployment",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "ReplicaSet",
							Name: "deployment-foo-5678",
						},
					},
					Namespace: "default",
				},
			},
			expectedWorkload: workload.PodWorkload{
				Kind:      workload.WorkloadKindDeployment,
				Name:      "deployment-foo",
				Namespace: "default",
			},
		},
		{
			name: "pod in DaemonSet",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "DaemonSet",
							Name: "someDaemonSet",
						},
					},
					Namespace: "default",
				},
			},
			expectedWorkload: workload.PodWorkload{
				Kind:      workload.WorkloadKindDaemonSet,
				Name:      "someDaemonSet",
				Namespace: "default",
			},
		},
		{
			name: "pod in StatefulSet",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					OwnerReferences: []metav1.OwnerReference{
						{
							Kind: "StatefulSet",
							Name: "someStatefulSet",
						},
					},
					Namespace: "default",
				},
			},
			expectedWorkload: workload.PodWorkload{
				Kind:      workload.WorkloadKindStatefulSet,
				Name:      "someStatefulSet",
				Namespace: "default",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			workload, err := pr.getPodWorkloadObject(context.Background(), c.pod)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if workload.Kind != c.expectedWorkload.Kind {
				t.Errorf("expected kind %s, got %s", c.expectedWorkload.Kind, workload.Kind)
			}
			if workload.Name != c.expectedWorkload.Name {
				t.Errorf("expected name %s, got %s", c.expectedWorkload.Name, workload.Name)
			}
			if workload.Namespace != c.expectedWorkload.Namespace {
				t.Errorf("expected namespace %s, got %s", c.expectedWorkload.Namespace, workload.Namespace)
			}
		})
	}
}
