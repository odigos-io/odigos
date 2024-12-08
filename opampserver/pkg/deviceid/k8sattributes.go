package deviceid

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// the purpose of this class is to use the k8s api to resolve container details
// into more useful, workloads details
type K8sPodInfoResolver struct {
	logger     logr.Logger
	kubeClient *kubernetes.Clientset
}

func NewK8sPodInfoResolver(logger logr.Logger, kubeClient *kubernetes.Clientset) *K8sPodInfoResolver {
	return &K8sPodInfoResolver{
		logger:     logger,
		kubeClient: kubeClient,
	}
}

// GetWorkloadNameByOwner gets the workload name and kind from the owner reference
func (k *K8sPodInfoResolver) GetWorkloadNameByOwner(ctx context.Context, podNamespace string, podName string) (string, string, *corev1.Pod, error) {
	pod, err := k.kubeClient.CoreV1().Pods(podNamespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", "", nil, err
	}

	ownerRefs := pod.GetOwnerReferences()
	for _, ownerRef := range ownerRefs {
		workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(ownerRef)
		if err == nil {
			return workloadName, string(workloadKind), pod, nil
		}
	}

	return podName, "Pod", pod, nil
}
