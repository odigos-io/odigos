package instrumentationdevice

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/webhook"

	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
)

const (
	EnvVarNamespace     = "ODIGOS_CONTAINER_NAMESPACE"
	EnvVarWorkloadKind  = "ODIGOS_WORKLOAD_KIND"
	EnvVarContainerName = "ODIGOS_CONTAINER_NAME"
	EnvVarPodName       = "ODIGOS_POD_NAME"
)

type PodsWebhook struct{}

var _ webhook.CustomDefaulter = &PodsWebhook{}

func (p *PodsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	// Inject ODIGOS environment variables into all containers
	injectOdigosEnvVars(pod)

	return nil
}

func injectOdigosEnvVars(pod *corev1.Pod) {

	namespace := pod.Namespace
	workloadKind := getWorkloadKind(pod)

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		envVars := []corev1.EnvVar{
			{
				Name:  EnvVarNamespace,
				Value: namespace,
			},
			{
				Name:  EnvVarWorkloadKind,
				Value: workloadKind,
			},
			{
				Name:  EnvVarContainerName,
				Value: container.Name,
			},
			{
				Name: EnvVarPodName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
			},
		}

		container.Env = append(container.Env, envVars...)
	}
}

// We can assume ReplicaSet is Deployment because this come after we identify the workload is supported by Odigos
func getWorkloadKind(pod *corev1.Pod) string {
	for _, ownerRef := range pod.OwnerReferences {
		switch ownerRef.Kind {
		case "ReplicaSet":
			return "Deployment"
		case "StatefulSet":
			return "StatefulSet"
		case "DaemonSet":
			return "DaemonSet"
		}
	}
	return "Unknown"
}
