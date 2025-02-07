package agentenabled

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/podswebhook"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	podutils "github.com/odigos-io/odigos/instrumentor/internal/pod"
	webhookdeviceinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_device_injector"
	webhookenvinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_env_injector"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
		logger.Error(errors.New("expected a Pod but got a %T"), "failed to inject odigos agent")
		return nil
	}

	pw, err := p.podWorkload(ctx, pod)
	if err != nil {
		// TODO: if the webhook is enabled for all pods, this is not necessarily an error
		logger.Error(err, "failed to get pod workload details. Skipping Injection of ODIGOS agent")
		return nil
	} else if pw == nil {
		return nil
	}

	var ic odigosv1.InstrumentationConfig
	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	err = p.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// instrumentationConfig does not exist, this pod does not belong to any odigos workloads
			return nil
		}
		logger.Error(err, "failed to get instrumentationConfig. Skipping Injection of ODIGOS agent")
		return nil
	}

	if !ic.Spec.AgentInjectionEnabled {
		// instrumentation config exists, but no agent should be injected by webhook
		return nil
	}

	// Add odiglet installed node-affinity to the pod
	podutils.AddOdigletInstalledAffinity(pod)

	volumeMounted := false
	for i := range pod.Spec.Containers {
		podContainerSpec := &pod.Spec.Containers[i]
		containerConfig := ic.Spec.GetContainerAgentConfig(podContainerSpec.Name)
		if containerConfig == nil {
			// no config is found for this container, so skip (don't inject anything to it)
			continue
		}

		if !containerConfig.AgentEnabled || containerConfig.OtelDistroName == "" {
			// container config exists, but no agent should be injected by webhook to this container
			continue
		}

		containerVolumeMounted, err := injectOdigosToContainer(containerConfig, podContainerSpec, *pw)
		if err != nil {
			logger.Error(err, "failed to inject ODIGOS agent to container")
			continue
		}
		volumeMounted = volumeMounted || containerVolumeMounted
	}

	if volumeMounted {
		// only mount the volume if at least one container has a volume to mount
		podswebhook.MountPodVolume(pod)
	}

	// Inject ODIGOS environment variables and instrumentation device into all containers
	injectErr := p.injectOdigosInstrumentation(ctx, pod, &ic, pw)
	if injectErr != nil {
		logger.Error(injectErr, "failed to inject ODIGOS instrumentation. Skipping Injection of ODIGOS agent")
		return nil
	}

	return nil
}

func (p *PodsWebhook) podWorkload(ctx context.Context, pod *corev1.Pod) (*k8sconsts.PodWorkload, error) {
	// In certain scenarios, the raw request can be utilized to retrieve missing details, like the namespace.
	// For example, prior to Kubernetes version 1.24 (see https://github.com/kubernetes/kubernetes/pull/94637),
	// namespaced objects could be sent to admission webhooks with empty namespaces during their creation.
	admissionRequest, err := admission.RequestFromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get admission request: %w", err)
	}

	pw, err := workload.PodWorkloadObject(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pod workload details from pod: %w", err)
	}
	if pw == nil {
		// for pods which are not managed by odigos supported workload
		return nil, nil
	}

	if pw.Namespace == "" {
		if admissionRequest.Namespace != "" {
			// If the namespace is available in the admission request, set it in the podWorkload.Namespace.
			pw.Namespace = admissionRequest.Namespace
		} else {
			// It is a case that not supposed to happen, but if it does, return an error.
			return nil, fmt.Errorf("namespace is empty for pod %s/%s, Skipping Injection of ODIGOS environment variables", pod.Namespace, pod.Name)
		}
	}

	return pw, nil
}

func (p *PodsWebhook) injectOdigosInstrumentation(ctx context.Context, pod *corev1.Pod, ic *odigosv1.InstrumentationConfig, pw *k8sconsts.PodWorkload) error {
	logger := log.FromContext(ctx)

	otelSdkToUse, err := getRelevantOtelSDKs(ctx, p.Client, *pw)
	if err != nil {
		return fmt.Errorf("failed to determine OpenTelemetry SDKs: %w", err)
	}

	var serviceName *string
	var serviceNameEnv *corev1.EnvVar

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		runtimeDetails := ic.Status.GetRuntimeDetailsForContainer(*container)
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

		webhookdeviceinjector.InjectOdigosInstrumentationDevice(*pw, container, otelSdk, runtimeDetails)
		webhookenvinjector.InjectOdigosAgentEnvVars(logger, *pw, container, otelSdk, runtimeDetails)

		if shouldInjectServiceName(runtimeDetails.Language, otelSdk) {
			// Ensure the serviceName is fetched only once per pod
			if serviceName == nil {
				serviceName = p.getServiceNameForEnv(ctx, logger, pw)
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

		resourceAttributes := getResourceAttributes(pw, container.Name)
		resourceAttributesEnvValue := getResourceAttributesEnvVarValue(resourceAttributes)

		container.Env = append(container.Env, corev1.EnvVar{
			Name:  otelResourceAttributesEnvVarName,
			Value: resourceAttributesEnvValue,
		})
	}
	return nil
}

func injectOdigosToContainer(containerConfig *odigosv1.ContainerAgentConfig, podContainerSpec *corev1.Container, pw k8sconsts.PodWorkload) (bool, error) {

	distroName := containerConfig.OtelDistroName

	distroMetadata := distros.GetDistroByName(distroName)
	if distroMetadata == nil {
		return false, fmt.Errorf("distribution %s not found", distroName)
	}

	volumeMounted := false
	for _, agentDirectory := range distroMetadata.AgentDirectories {
		podswebhook.MountDirectory(podContainerSpec, agentDirectory.DirectoryName)
		podswebhook.InjectOdigosK8sEnvVars(podContainerSpec, distroName, pw.Namespace)
		volumeMounted = true
	}

	return volumeMounted, nil
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
