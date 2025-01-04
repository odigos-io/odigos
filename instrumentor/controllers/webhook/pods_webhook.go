package webhook

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common/resourceattributes"

	k8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/attribute"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const otelServiceNameEnvVarName = "OTEL_SERVICE_NAME"
const otelResourceAttributesEnvVarName = "OTEL_RESOURCE_ATTRIBUTES"

type resourceAttribute struct {
	Key   attribute.Key
	Value string
}

type PodsWebhook struct {
	client.Client
}

var _ webhook.CustomDefaulter = &PodsWebhook{}

func (p *PodsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	serviceName, podWorkload := p.getServiceNameForEnv(ctx, pod)

	// Inject ODIGOS environment variables into all containers
	p.injectOdigosEnvVars(pod, podWorkload, serviceName)

	return nil
}

// checks for the service name on the annotation, or fallback to the workload name
func (p *PodsWebhook) getServiceNameForEnv(ctx context.Context, pod *corev1.Pod) (*string, *workload.PodWorkload) {
	logger := log.FromContext(ctx)

	podWorkload, err := workload.PodWorkloadObject(ctx, pod)
	if err != nil {
		logger.Error(err, "failed to extract pod workload details from pod. skipping OTEL_SERVICE_NAME injection")
		return nil, nil
	}

	workloadObj, err := workload.GetWorkloadObject(ctx, client.ObjectKey{Namespace: podWorkload.Namespace, Name: podWorkload.Name}, podWorkload.Kind, p.Client)
	if err != nil {
		logger.Error(err, "failed to get workload object from cache. cannot check for workload annotation. using workload name as OTEL_SERVICE_NAME")
		return &podWorkload.Name, podWorkload
	}
	resolvedServiceName := workload.ExtractServiceNameFromAnnotations(workloadObj.GetAnnotations(), podWorkload.Name)
	return &resolvedServiceName, podWorkload
}

func (p *PodsWebhook) injectOdigosEnvVars(pod *corev1.Pod, podWorkload *workload.PodWorkload, serviceName *string) {
	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]

		// Pod name is not available yet in webhook so use downward API to get it
		podName := fmt.Sprintf("$(%s)", k8sconsts.OdigosEnvVarPodName)

		identifier := &resourceattributes.ContainerIdentifier{
			PodName:       podName,
			Namespace:     pod.Namespace,
			ContainerName: container.Name,
		}

		// Add container identifier as separate env vars:
		// This is used by process discovery to identify the container
		// Also, used by OpAMP clients to send it back to the server on the first heartbeat
		// TODO(edenfed): these values will be duplicated between the resource attributes and the env vars
		// We should consider removing these and only use the resource attributes
		modifications := map[string]envVarModification{
			k8sconsts.OdigosEnvVarNamespace: {
				Value:  identifier.Namespace,
				Action: Upsert,
			},
			k8sconsts.OdigosEnvVarPodName: {
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						FieldPath: "metadata.name",
					},
				},
				Action: Upsert,
			},
			k8sconsts.OdigosEnvVarContainerName: {
				Value:  identifier.ContainerName,
				Action: Upsert,
			},
		}

		resAttributes := resourceattributes.BeforePodStart(identifier)
		if resAttributes != nil {
			modifications[otelResourceAttributesEnvVarName] = envVarModification{
				Value:  resAttributes.ToEnvVarString(),
				Action: AppendWithComma,
			}
		}

		if serviceName != nil {
			modifications[otelServiceNameEnvVarName] = envVarModification{
				Value:  *serviceName,
				Action: Upsert,
			}
		}

		container.Env = p.modifyEnvVars(container.Env, modifications)
	}
}
