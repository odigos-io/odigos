package agentenabled

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/podswebhook"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	podutils "github.com/odigos-io/odigos/instrumentor/internal/pod"
	webhookenvinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_env_injector"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type PodsWebhook struct {
	client.Client
	DistrosGetter *distros.Getter
}

var _ webhook.CustomDefaulter = &PodsWebhook{}

func (p *PodsWebhook) Default(ctx context.Context, obj runtime.Object) error {
	logger := log.FromContext(ctx)
	pod, ok := obj.(*corev1.Pod)

	if !ok {
		logger.Error(errors.New("expected a Pod but got a %T"), "failed to inject odigos agent")
		return nil
	}

	// the following check is another safety guard which is meant to avoid injecting the agent any instrumentation
	// (including the device) in case the odiglet daemonset is not ready.
	// it can cover cases like that odiglet is deleted and no replicas are ready.
	odigosNamespace := env.GetCurrentNamespace()
	odigletDaemonset := &appsv1.DaemonSet{}
	if err := p.Get(ctx, client.ObjectKey{Namespace: odigosNamespace, Name: k8sconsts.OdigletDaemonSetName}, odigletDaemonset); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("odiglet daemonset not found, skipping Injection of ODIGOS agent")
		} else {
			logger.Error(err, "failed to verify odiglet daemonset ready for instrumented pod. Skipping Injection of ODIGOS agent")
		}
		return nil
	}
	if odigletDaemonset.Status.NumberReady == 0 {
		logger.Error(errors.New("odiglet daemonset is not ready. Skipping Injection of ODIGOS agent"), "failed to verify odiglet daemonset ready for instrumented pod")
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

	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, p.Client)
	if err != nil {
		logger.Error(err, "failed to get ODIGOS config. Skipping Injection of ODIGOS agent")
		return nil
	}
	if odigosConfiguration.MountMethod == nil {
		// we are reading the effective config which should already have the mount method resolved or defaulted
		logger.Error(errors.New("mount method is not set in ODIGOS config"), "Skipping Injection of ODIGOS agent")
		return nil
	}

	// this is temporary and should be refactored so the service name and other resource attributes are written to agent config
	serviceName := ic.Spec.ServiceName
	if serviceName == "" {
		logger.Error(errors.New("failed to get service name for pod"), "Skipping Injection of ODIGOS agent")
		return nil
	}

	karpenterDisabled := odigosConfiguration.KarpenterEnabled == nil || !*odigosConfiguration.KarpenterEnabled
	mountIsVirtualDevice := odigosConfiguration.MountMethod == nil || *odigosConfiguration.MountMethod == common.K8sVirtualDeviceMountMethod

	// Add odiglet installed node-affinity to the pod, for non Karpenter installations
	// and if the mount method is not virtual device (which is the default)
	if karpenterDisabled && !mountIsVirtualDevice {
		podutils.AddOdigletInstalledAffinity(pod)
	}

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

		containerVolumeMounted, err := p.injectOdigosToContainer(containerConfig, podContainerSpec, *pw, serviceName, odigosConfiguration)
		if err != nil {
			logger.Error(err, "failed to inject ODIGOS agent to container")
			continue
		}
		volumeMounted = volumeMounted || containerVolumeMounted
	}

	if *odigosConfiguration.MountMethod == common.K8sHostPathMountMethod && volumeMounted {
		// only mount the volume if at least one container has a volume to mount
		podswebhook.MountPodVolume(pod)
	}

	// Inject ODIGOS environment variables and instrumentation device into all containers
	injectErr := p.injectOdigosInstrumentation(ctx, pod, &ic, pw, &odigosConfiguration)
	if injectErr != nil {
		logger.Error(injectErr, "failed to inject ODIGOS instrumentation. Skipping Injection of ODIGOS agent")
		return nil
	}

	if odigosConfiguration.UserInstrumentationEnvs != nil {
		podswebhook.InjectUserEnvForLang(&odigosConfiguration, pod, &ic)
	}

	// store the agents deployment value so we can later associate each pod with the instrumentation version.
	// we can pull only our pods into cache, and follow the lifecycle of the instrumentation process.
	pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel] = ic.Spec.AgentsMetaHash

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

func (p *PodsWebhook) injectOdigosInstrumentation(ctx context.Context, pod *corev1.Pod, ic *odigosv1.InstrumentationConfig, pw *k8sconsts.PodWorkload, config *common.OdigosConfiguration) error {
	logger := log.FromContext(ctx)

	otelSdkToUse, err := getRelevantOtelSDKs(ctx, p.Client, *pw)
	if err != nil {
		return fmt.Errorf("failed to determine OpenTelemetry SDKs: %w", err)
	}

	for i := range pod.Spec.Containers {
		container := &pod.Spec.Containers[i]
		runtimeDetails := getRuntimeInfoForContainerName(ic, container.Name)
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

		err = webhookenvinjector.InjectOdigosAgentEnvVars(ctx, logger, container, otelSdk, runtimeDetails, config)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PodsWebhook) injectOdigosToContainer(containerConfig *odigosv1.ContainerAgentConfig, podContainerSpec *corev1.Container, pw k8sconsts.PodWorkload, serviceName string, config common.OdigosConfiguration) (bool, error) {
	var err error

	distroName := containerConfig.OtelDistroName
	distroMetadata := p.DistrosGetter.GetDistroByName(distroName)
	if distroMetadata == nil {
		return false, fmt.Errorf("distribution %s not found", distroName)
	}

	// check for existing env vars so we don't introduce them again
	existingEnvNames := podswebhook.GetEnvVarNamesSet(podContainerSpec)

	// inject various kinds of distro environment variables
	existingEnvNames, err = podswebhook.InjectStaticEnvVarsToPodContainer(existingEnvNames, podContainerSpec, distroMetadata.EnvironmentVariables.StaticVariables, containerConfig.DistroParams)
	if err != nil {
		return false, err
	}
	existingEnvNames = podswebhook.InjectOdigosK8sEnvVars(existingEnvNames, podContainerSpec, distroName, pw.Namespace)
	if distroMetadata.EnvironmentVariables.OpAmpClientEnvironments {
		existingEnvNames = podswebhook.InjectOpampServerEnvVar(existingEnvNames, podContainerSpec)
	}
	if distroMetadata.EnvironmentVariables.SignalsAsStaticOtelEnvVars {
		tracesEnabled := containerConfig.Traces != nil
		metricsEnabled := containerConfig.Metrics != nil
		logsEnabled := containerConfig.Logs != nil
		existingEnvNames = podswebhook.InjectSignalsAsStaticOtelEnvVars(existingEnvNames, podContainerSpec, tracesEnabled, metricsEnabled, logsEnabled)
	}
	if distroMetadata.EnvironmentVariables.OtlpHttpLocalNode {
		existingEnvNames = podswebhook.InjectOtlpHttpEndpointEnvVar(existingEnvNames, podContainerSpec)
	}

	volumeMounted := false
	if distroMetadata.RuntimeAgent != nil {
		if *config.MountMethod == common.K8sHostPathMountMethod {
			// mount directory only if the mount type is host-path
			for _, agentDirectoryName := range distroMetadata.RuntimeAgent.DirectoryNames {
				podswebhook.MountDirectory(podContainerSpec, agentDirectoryName)
				volumeMounted = true
			}

			// if loader is enabled, mount the loader directory
			if config.AgentEnvVarsInjectionMethod != nil &&
				(*config.AgentEnvVarsInjectionMethod == common.LoaderFallbackToPodManifestInjectionMethod ||
					*config.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod) {
				podswebhook.MountDirectory(podContainerSpec, filepath.Join(k8sconsts.OdigosAgentsDirectory, consts.OdigosLoaderDirName))
			}
		}
		if distroMetadata.RuntimeAgent.K8sAttrsViaEnvVars {
			podswebhook.InjectOtelResourceAndServiceNameEnvVars(existingEnvNames, podContainerSpec, distroName, pw, serviceName)
		}
		// TODO: once we have a flag to enable/disable device injection, we should check it here.
		if distroMetadata.RuntimeAgent.Device != nil {

			// amir 17 feb 2025, this is here only for migration.
			// even if mount method is not device, we still need to inject the deprecated agent specific device
			// while we remove them one by one
			isGenericDevice := *distroMetadata.RuntimeAgent.Device == k8sconsts.OdigosGenericDeviceName
			if *config.MountMethod == common.K8sVirtualDeviceMountMethod || !isGenericDevice {
				deviceName := *distroMetadata.RuntimeAgent.Device
				// TODO: currently devices are composed with glibc as input for dotnet.
				// as devices will soon converge to a single device, I am hardcoding the logic here,
				// which will eventually be removed once dotnet specific devices are removed.
				if containerConfig.DistroParams != nil {
					libcType, ok := containerConfig.DistroParams[common.LibcTypeDistroParameterName]
					if ok {
						libcPrefix := ""
						if libcType == string(common.Musl) {
							libcPrefix = "musl-"
						}
						deviceName = strings.ReplaceAll(deviceName, "{{param.LIBC_TYPE}}", libcPrefix)
					}
				}
				podswebhook.InjectDeviceToContainer(podContainerSpec, deviceName)
			}
		}
	}

	return volumeMounted, nil
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

func getRuntimeInfoForContainerName(ic *odigosv1.InstrumentationConfig, containerName string) *odigosv1.RuntimeDetailsByContainer {

	// first look for the value in overrides
	for i := range ic.Spec.ContainersOverrides {
		if ic.Spec.ContainersOverrides[i].ContainerName == containerName {
			if ic.Spec.ContainersOverrides[i].RuntimeInfo != nil {
				return ic.Spec.ContainersOverrides[i].RuntimeInfo
			} else {
				break
			}
		}
	}

	// if not found in overrides, look for the value in automatic runtime detection
	for _, container := range ic.Status.RuntimeDetailsByContainer {
		if container.ContainerName == containerName {
			return &container
		}
	}

	// if both are not found, return we don't have runtime info for this container
	return nil
}
