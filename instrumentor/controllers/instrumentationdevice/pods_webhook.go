package instrumentationdevice

import (
	"context"
	"fmt"
	"strings"

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

	// Common environment variables that do not change across containers
	commonEnvVars := []corev1.EnvVar{
		{
			Name: EnvVarNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
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

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// Check if the container does NOT have device in conatiner limits. If so, skip the environment injection.
		if !hasOdigosInstrumentationInLimits(container.Resources) {
			continue
		}

		// Check if the environment variables are already present, if so skip inject them again.
		if envVarsExist(container.Env, commonEnvVars) {
			continue
		}

		container.Env = append(container.Env, append(commonEnvVars, corev1.EnvVar{
			Name:  EnvVarContainerName,
			Value: container.Name,
		})...)
	}
}

func envVarsExist(containerEnv []corev1.EnvVar, commonEnvVars []corev1.EnvVar) bool {
	envMap := make(map[string]struct{})
	for _, envVar := range containerEnv {
		envMap[envVar.Name] = struct{}{} // Inserting empty struct as value
	}

	for _, commonEnvVar := range commonEnvVars {
		if _, exists := envMap[commonEnvVar.Name]; exists { // Checking if key exists
			return true
		}
	}
	return false
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
