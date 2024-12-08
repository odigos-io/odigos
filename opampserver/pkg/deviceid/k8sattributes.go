package deviceid

import (
	"context"

	"github.com/go-logr/logr"
	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
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

func (k *K8sPodInfoResolver) getServiceNameFromInstrumentationConfig(ctx context.Context, name string, kind string, namespace string) (string, bool) {

	instConfigName := workload.CalculateWorkloadRuntimeObjectName(name, workload.WorkloadKind(kind))
	cfg, err := rest.InClusterConfig()
	if err != nil {
		k.logger.Error(err, "Failed to init Kubernetes API client")
	}
	odigosKubeClient, err := odigosclientset.NewForConfig(cfg)
	if err != nil {
		k.logger.Error(err, "Failed to init Odigos client")
	}

	instConfig, err := odigosKubeClient.OdigosV1alpha1().InstrumentationConfigs(namespace).Get(ctx, instConfigName, metav1.GetOptions{})
	if err != nil {
		k.logger.Error(err, "Failed to get InstrumentationConfig for workload", "name", name, "kind", kind, "namespace", namespace)
		return "", false
	}

	if instConfig.Spec.ServiceName == "" {
		k.logger.Info("ServiceName is not specified in InstrumentationConfig, falling back to workload name", "name", name, "namespace", namespace)
		return "", false
	}
	return instConfig.Spec.ServiceName, true
}

// Resolves the service name, with the following priority:
// 1. If the user added reported name annotation to the workload, use it
// 2. Otherwise, use the workload name as service name
//
// if one of the above conditions has err, it will be logged and the next condition will be checked
func (k *K8sPodInfoResolver) ResolveServiceName(ctx context.Context, workloadName string, workloadKind string, containerDetails *ContainerDetails) string {

	// we always fetch the fresh service name from the annotation to make sure the most up to date value is returned
	serviceName, foundReportedName := k.getServiceNameFromInstrumentationConfig(ctx, workloadName, workloadKind, containerDetails.PodNamespace)
	if foundReportedName {
		return serviceName
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
