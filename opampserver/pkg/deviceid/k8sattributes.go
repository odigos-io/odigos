package deviceid

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/consts"
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

func (k *K8sPodInfoResolver) getServiceNameFromAnnotation(ctx context.Context, name string, kind string, namespace string) (string, bool) {
	obj, err := k.getWorkloadObject(ctx, name, kind, namespace)
	if err != nil {
		k.logger.Error(err, "failed to get workload object to resolve reported service name annotation. will use fallback service name")
		return "", false
	}

	annotations := obj.GetAnnotations()
	if annotations == nil {
		// no annotations, so service name is not specified by user. fallback to workload name
		return "", false
	}

	overwrittenName, exists := annotations[consts.OdigosReportedNameAnnotation]
	if !exists {
		// the is no annotation by user for specific reported service name for this workload
		// fallback to workload name
		return "", false
	}

	return overwrittenName, true
}

func (k *K8sPodInfoResolver) getWorkloadObject(ctx context.Context, name string, kind string, namespace string) (metav1.Object, error) {
	switch kind {
	case "Deployment":
		return k.kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	case "StatefulSet":
		return k.kubeClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "DaemonSet":
		return k.kubeClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	case "Pod":
		return k.kubeClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	return nil, errors.New("failed to get workload object for kind: " + kind)
}

// Resolves the service name, with the following priority:
// 1. If the user added reported name annotation to the workload, use it
// 2. If the pod has multiple containers, use the container name as service name
// 3. Otherwise, use the workload name as service name
//
// if one of the above conditions has err, it will be logged and the next condition will be checked
func (k *K8sPodInfoResolver) ResolveServiceName(ctx context.Context, workloadName string, workloadKind string, containerDetails *ContainerDetails) string {

	// we always fetch the fresh service name from the annotation to make sure the most up to date value is returned
	serviceName, foundReportedName := k.getServiceNameFromAnnotation(ctx, workloadName, workloadKind, containerDetails.PodNamespace)
	if foundReportedName {
		return serviceName
	}

	if containerDetails.ContainersInPod > 1 {
		// multiple containers in pod, use container name as service name
		// might want to revisit in the future:
		// should we include the workload name?
		// should we use same service name for multiple containers if configured via annotation?
		return containerDetails.ContainerName
	}

	return workloadName
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
