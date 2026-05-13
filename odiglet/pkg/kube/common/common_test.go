package common

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMatchingPodsForWorkloadOnNode_RespectsInstrumentationConfigNamespace(t *testing.T) {
	prevNodeName := env.Current.NodeName
	env.Current.NodeName = "node-a"
	t.Cleanup(func() {
		env.Current.NodeName = prevNodeName
	})

	ic := &odigosv1.InstrumentationConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workload.CalculateWorkloadRuntimeObjectName("api", "Deployment"),
			Namespace: "prod",
		},
	}

	started := true
	newPod := func(name, namespace string) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						Name: "api",
						Kind: "Deployment",
					},
				},
			},
			Spec: corev1.PodSpec{
				NodeName: "node-a",
				Containers: []corev1.Container{
					{Name: "app"},
				},
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:    "app",
						Ready:   true,
						Started: &started,
					},
				},
			},
		}
	}

	pods, err := MatchingPodsForWorkloadOnNode(ic, corev1.PodList{
		Items: []corev1.Pod{
			newPod("api-prod-1", "prod"),
			newPod("api-staging-1", "staging"),
		},
	})
	if err != nil {
		t.Fatalf("matching pods failed: %v", err)
	}

	if len(pods) != 1 {
		t.Fatalf("expected exactly 1 pod, got %d", len(pods))
	}

	if pods[0].Namespace != "prod" {
		t.Fatalf("expected selected pod namespace to be prod, got %s", pods[0].Namespace)
	}
}
