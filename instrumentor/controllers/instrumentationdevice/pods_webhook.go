package instrumentationdevice

import (
	"context"
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"sigs.k8s.io/controller-runtime/pkg/client"

	common "github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	EnvVarNamespace     = "ODIGOS_WORKLOAD_NAMESPACE"
	EnvVarContainerName = "ODIGOS_CONTAINER_NAME"
	EnvVarPodName       = "ODIGOS_POD_NAME"
)

type PodsWebhook struct {
	Client client.Client
}

var _ webhook.CustomDefaulter = &PodsWebhook{}

func (p *PodsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	logger := log.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	instApp, err := workload.GetRuntimeDetailsForPod(ctx, p.Client, pod)
	if err != nil {
		logger.Error(err, "Failed to get runtime details for pod")
		return err
	}

	// Inject ODIGOS environment variables into all containers
	p.injectOdigosEnvVars(pod)

	return nil
}

func (p *PodsWebhook) envVarsToMap(envVars []corev1.EnvVar) map[string]corev1.EnvVar {
	envMap := make(map[string]corev1.EnvVar)
	for i := range envVars {
		envMap[envVars[i].Name] = envVars[i]
	}
	return envMap
}

func (p *PodsWebhook) injectOdigosEnvVars(pod *corev1.Pod) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// Check if the container does NOT have device in conatiner limits. If so, skip the environment injection.
		if !hasOdigosInstrumentationInLimits(container.Resources) {
			continue
		}

		envsMap := p.envVarsToMap(container.Env)
		envsMap[EnvVarNamespace] = corev1.EnvVar{
			Name:  EnvVarNamespace,
			Value: pod.Namespace,
		}

		envsMap[EnvVarPodName] = corev1.EnvVar{
			Name: EnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		}

		envsMap[EnvVarContainerName] = corev1.EnvVar{
			Name:  EnvVarContainerName,
			Value: container.Name,
		}

		p.persistEnvVars(envsMap, container)
	}
}

func (p *PodsWebhook) persistEnvVars(envsMap map[string]corev1.EnvVar, container *corev1.Container) {
	envs := make([]corev1.EnvVar, 0, len(envsMap))
	for _, env := range envsMap {
		envs = append(envs, env)
	}
	container.Env = envs
}

// Helper function to check if a container's resource limits have a key starting with the specified namespace
func hasOdigosInstrumentationInLimits(resources corev1.ResourceRequirements) bool {
	for resourceName := range resources.Limits {
		if strings.HasPrefix(string(resourceName), common.OdigosResourceNamespace) {
			return true
		}
	}
	return false
}
