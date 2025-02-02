package instrumentationdevice

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	webhookdeviceinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_device_injector"
	webhookenvinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_env_injector"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
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
	logger := log.FromContext(ctx)

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("expected a Pod but got a %T", obj)
	}

	if pod.Annotations == nil {
		pod.Annotations = map[string]string{}
	}

	// Inject ODIGOS environment variables and instrumentation device into all containers
	return p.injectOdigosInstrumentation(ctx, logger, pod)
}

func (p *PodsWebhook) injectOdigosInstrumentation(ctx context.Context, logger logr.Logger, pod *corev1.Pod) error {
	// Environment variables that remain consistent across all containers
	commonEnvVars := getCommonEnvVars()

	// In certain scenarios, the raw request can be utilized to retrieve missing details, like the namespace.
	// For example, prior to Kubernetes version 1.24 (see https://github.com/kubernetes/kubernetes/pull/94637),
	// namespaced objects could be sent to admission webhooks with empty namespaces during their creation.
	admissionRequest, err := admission.RequestFromContext(ctx)
	if err != nil {
		return fmt.Errorf("failed to get admission request: %w", err)
	}

	podWorkload, err := workload.PodWorkloadObject(ctx, pod)
	if err != nil {
		return fmt.Errorf("failed to extract pod workload details from pod: %w", err)
	}

	if podWorkload.Namespace == "" {
		if admissionRequest.Namespace != "" {
			// If the namespace is available in the admission request, set it in the podWorkload.Namespace.
			podWorkload.Namespace = admissionRequest.Namespace
		} else {
			// It is a case that not supposed to happen, but if it does, return an error.
			return fmt.Errorf("namespace is empty for pod %s/%s, Skipping Injection of ODIGOS environment variables", pod.Namespace, pod.Name)
		}
	}
	var workloadInstrumentationConfig odigosv1.InstrumentationConfig
	instrumentationConfigName := workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind)

	if err := p.Get(ctx, client.ObjectKey{Namespace: podWorkload.Namespace, Name: instrumentationConfigName}, &workloadInstrumentationConfig); err != nil {
		return fmt.Errorf("failed to get instrumentationConfig: %w", err)
	}

	otelSdkToUse, err := getRelevantOtelSDKs(ctx, p.Client, *podWorkload)
	if err != nil {
		return fmt.Errorf("failed to determine OpenTelemetry SDKs: %w", err)
	}

	var serviceName *string
	var serviceNameEnv *corev1.EnvVar

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		runtimeDetails := workloadInstrumentationConfig.Status.GetRuntimeDetailsForContainer(*container)
		if runtimeDetails == nil {
			continue
		}

		if runtimeDetails.Language == common.UnknownProgrammingLanguage {
			continue
		}

		otelSdk, found := otelSdkToUse[runtimeDetails.Language]
		if !found {
			continue
		}

		webhookdeviceinjector.InjectOdigosInstrumentationDevice(*podWorkload, container, otelSdk, runtimeDetails)
		webhookenvinjector.InjectOdigosAgentEnvVars(logger, *podWorkload, container, otelSdk, runtimeDetails)

		// Check if the environment variables are already present, if so skip inject them again.
		if envVarsExist(container.Env, commonEnvVars) {
			continue
		}

		containerNameEnv := corev1.EnvVar{Name: k8sconsts.OdigosEnvVarContainerName, Value: container.Name}
		container.Env = append(container.Env, append(commonEnvVars, containerNameEnv)...)

		if shouldInjectServiceName(runtimeDetails.Language, otelSdk) {
			// Ensure the serviceName is fetched only once per pod
			if serviceName == nil {
				serviceName = p.getServiceNameForEnv(ctx, logger, podWorkload)
			}
			// Initialize serviceNameEnv only once per pod if serviceName is valid
			if serviceName != nil && serviceNameEnv == nil {
				serviceNameEnv = &corev1.EnvVar{
					Name:  otelServiceNameEnvVarName,
					Value: *serviceName,
				}
			}

			if !otelNameExists(container.Env) {
				container.Env = append(container.Env, *serviceNameEnv)
			}
		}

		resourceAttributes := getResourceAttributes(podWorkload, container.Name)
		resourceAttributesEnvValue := getResourceAttributesEnvVarValue(resourceAttributes)

		container.Env = append(container.Env, corev1.EnvVar{
			Name:  otelResourceAttributesEnvVarName,
			Value: resourceAttributesEnvValue,
		})
	}
	return nil
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

func getWorkloadKindAttributeKey(podWorkload *k8sconsts.PodWorkload) attribute.Key {
	switch podWorkload.Kind {
	case k8sconsts.WorkloadKindDeployment:
		return semconv.K8SDeploymentNameKey
	case k8sconsts.WorkloadKindStatefulSet:
		return semconv.K8SStatefulSetNameKey
	case k8sconsts.WorkloadKindDaemonSet:
		return semconv.K8SDaemonSetNameKey
	}
	return attribute.Key("")
}

func getResourceAttributes(podWorkload *k8sconsts.PodWorkload, containerName string) []resourceAttribute {
	if podWorkload == nil {
		return []resourceAttribute{}
	}

	workloadKindKey := getWorkloadKindAttributeKey(podWorkload)
	return []resourceAttribute{
		{
			Key:   semconv.K8SContainerNameKey,
			Value: containerName,
		},
		{
			Key:   semconv.K8SNamespaceNameKey,
			Value: podWorkload.Namespace,
		},
		{
			Key:   workloadKindKey,
			Value: podWorkload.Name,
		},
	}
}

func getResourceAttributesEnvVarValue(ra []resourceAttribute) string {
	var attrs []string
	for _, a := range ra {
		attrs = append(attrs, fmt.Sprintf("%s=%s", a.Key, a.Value))
	}
	return strings.Join(attrs, ",")
}

func otelNameExists(containerEnv []corev1.EnvVar) bool {
	for _, envVar := range containerEnv {
		if envVar.Name == otelServiceNameEnvVarName {
			return true
		}
	}
	return false
}

// this is used to set the OTEL_SERVICE_NAME for programming languages and otel sdks that requires it.
// eBPF instrumentations sets the service name in code, thus it's not needed here.
// OpAMP sends the service name in the protocol, thus it's not needed here.
// We are only left with OSS Java and Dotnet that requires the OTEL_SERVICE_NAME to be set.
func shouldInjectServiceName(pl common.ProgrammingLanguage, otelsdk common.OtelSdk) bool {
	if pl == common.DotNetProgrammingLanguage {
		return true
	}
	if pl == common.JavaProgrammingLanguage && otelsdk.SdkTier == common.CommunityOtelSdkTier {
		return true
	}
	return false
}

func getCommonEnvVars() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: k8sconsts.OdigosEnvVarNamespace,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
		{
			Name: k8sconsts.OdigosEnvVarPodName,
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}
}

// checks for the service name on the annotation, or fallback to the workload name
func (p *PodsWebhook) getServiceNameForEnv(ctx context.Context, logger logr.Logger, podWorkload *k8sconsts.PodWorkload) *string {
	workloadObj := workload.ClientObjectFromWorkloadKind(podWorkload.Kind)
	err := p.Client.Get(ctx, client.ObjectKey{Namespace: podWorkload.Namespace, Name: podWorkload.Name}, workloadObj)
	if err != nil {
		logger.Error(err, "failed to get workload object from cache. cannot check for workload source. using workload name as OTEL_SERVICE_NAME")
		return &podWorkload.Name
	}

	resolvedServiceName, err := sourceutils.OtelServiceNameBySource(ctx, p.Client, workloadObj)
	if err != nil {
		logger.Error(err, "failed to get OTel service name from source. using workload name as OTEL_SERVICE_NAME")
		return &podWorkload.Name
	}

	if resolvedServiceName == "" {
		resolvedServiceName = podWorkload.Name
	}

	return &resolvedServiceName
}

func getRelevantOtelSDKs(ctx context.Context, kubeClient client.Client, podWorkload k8sconsts.PodWorkload) (map[common.ProgrammingLanguage]common.OtelSdk, error) {

	instrumentationRules := odigosv1.InstrumentationRuleList{}
	if err := kubeClient.List(ctx, &instrumentationRules); err != nil {
		return nil, err
	}

	otelSdkToUse := sdks.GetDefaultSDKs()
	for i := range instrumentationRules.Items {
		rule := &instrumentationRules.Items[i]
		if rule.Spec.Disabled || rule.Spec.OtelSdks == nil {
			// we only care about rules that have otel sdks configuration
			continue
		}

		if !utils.IsWorkloadParticipatingInRule(podWorkload, rule) {
			// filter rules that do not apply to the workload
			continue
		}

		for lang, otelSdk := range rule.Spec.OtelSdks.OtelSdkByLanguage {
			// languages can override the default otel sdk or another rule.
			// there is not check or warning if a language is defined in multiple rules at the moment.
			otelSdkToUse[lang] = otelSdk
		}
	}

	return otelSdkToUse, nil
}
