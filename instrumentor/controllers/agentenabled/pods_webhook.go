package agentenabled

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/podswebhook"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	podutils "github.com/odigos-io/odigos/instrumentor/internal/pod"
	webhookenvinjector "github.com/odigos-io/odigos/instrumentor/internal/webhook_env_injector"
	"github.com/odigos-io/odigos/instrumentor/sdks"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
)

type PodsWebhook struct {
	client.Client
	DistrosGetter *distros.Getter
	// decoder is used to decode the admission request's raw object into a structured corev1.Pod.
	Decoder     admission.Decoder
	WaspMutator func(*corev1.Pod, common.OdigosConfiguration) error
}

var _ admission.Handler = &PodsWebhook{}

func (p *PodsWebhook) InjectDecoder(d admission.Decoder) error {
	p.Decoder = d
	return nil
}

var (
	ErrNotOdigablePod               = errors.New("pod does not belong to an odigos workload")
	ErrInjectionDisabled            = errors.New("agent injection is disabled in instrumentation config")
	ErrIgnorePod                    = errors.New("pod is not eligible for mutation")
	ErrOdigletDeviceNotHealthy      = errors.New("odiglet device plugin is unhealthy")
	ErrMissingServiceName           = errors.New("instrumentation config is missing service name")
	ErrMissingInstrumentationConfig = errors.New("instrumentation config is missing")
	ErrMissingOdigosConfiguration   = errors.New("odigos configuration is missing")
	ErrUnknownDistroName            = errors.New("distro not found")
	ErrEnvVarInjection              = errors.New("failed to inject environment variables")
	ErrMountMethodNotSet            = errors.New("mount method is not set in ODIGOS config")
)

// Handle implements the admission.Handler interface to safely mutate Pod objects at creation time.
//
// This webhook applies ODIGOS instrumentation logic by:
// - Decoding the incoming Pod from the admission request
// - Cloning the Pod to ensure safe mutation
// - Injecting volumes, env vars, and labels based on ODIGOS configuration
// - Returning a patch that atomically applies changes to the original Pod
//
// If injection fails for any reason, the webhook returns an Allowed response with no changes,
// ensuring user workloads are never blocked by instrumentation logic.
func (p *PodsWebhook) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	if err := p.Decoder.Decode(req, &pod); err != nil {
		logger.Error(err, "unable to decode Pod from admission request")
		return admission.Allowed("proceeding without mutation")
	}

	// Clone Pod for safe mutation
	mutated := pod.DeepCopy()

	err := p.injectOdigos(ctx, mutated, req)
	switch {
	case errors.Is(err, ErrIgnorePod), errors.Is(err, ErrNotOdigablePod), errors.Is(err, ErrInjectionDisabled):
		// These are expected, allow without mutation
		return admission.Allowed("odigos injection skipped: not applicable")

	case err != nil:
		// Real failure, still allow pod to be created but log clearly
		logger.Error(err, "unexpected failure during odigos injection")
		return admission.Allowed("odigos injection failed internally")

	default:
		// Return patch
		mutatedRaw, err := json.Marshal(mutated)
		if err != nil {
			logger.Error(err, "failed to marshal mutated Pod")
			return admission.Allowed("proceeding without mutation")
		}

		return admission.PatchResponseFromRaw(req.Object.Raw, mutatedRaw)
	}
}

func (p *PodsWebhook) injectOdigos(ctx context.Context, pod *corev1.Pod, req admission.Request) error {
	logger := log.FromContext(ctx)

	odigosNamespace := env.GetCurrentNamespace()

	pw, err := p.podWorkload(ctx, pod, req)
	if err != nil {
		// TODO: if the webhook is enabled for all pods, this is not necessarily an error
		logger.Error(err, "failed to get pod workload details. Skipping Injection of ODIGOS agent")
		return ErrIgnorePod
	} else if pw == nil {
		return ErrNotOdigablePod
	}

	var ic odigosv1.InstrumentationConfig
	icName := workload.CalculateWorkloadRuntimeObjectName(pw.Name, pw.Kind)
	err = p.Get(ctx, client.ObjectKey{Namespace: pw.Namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// instrumentationConfig does not exist, this pod does not belong to any odigos workloads
			return ErrNotOdigablePod
		}
		return fmt.Errorf("%w: %v", ErrMissingInstrumentationConfig, err)
	}

	if !ic.Spec.AgentInjectionEnabled {
		// instrumentation config exists, but no agent should be injected by webhook
		return ErrInjectionDisabled
	}

	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, p.Client)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrMissingOdigosConfiguration, err)
	}

	if odigosConfiguration.MountMethod == nil {
		// we are reading the effective config which should already have the mount method resolved or defaulted
		return ErrMountMethodNotSet
	}
	mountMethod := *odigosConfiguration.MountMethod

	mountIsVirtualDevice := (mountMethod == common.K8sVirtualDeviceMountMethod)
	if mountIsVirtualDevice && odigosConfiguration.CheckDeviceHealthBeforeInjection != nil && *odigosConfiguration.CheckDeviceHealthBeforeInjection {
		err := podswebhook.CheckDevicePluginContainersHealth(ctx, p.Client, odigosNamespace)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrOdigletDeviceNotHealthy, err)
		}
	}

	// this is temporary and should be refactored so the service name and other resource attributes are written to agent config
	serviceName := ic.Spec.ServiceName
	if serviceName == "" {
		return ErrMissingServiceName
	}

	karpenterDisabled := odigosConfiguration.KarpenterEnabled == nil || !*odigosConfiguration.KarpenterEnabled
	mountIsHostPath := odigosConfiguration.MountMethod != nil && *odigosConfiguration.MountMethod == common.K8sHostPathMountMethod

	// Add odiglet-installed node affinity to the pod for non-Karpenter installations,
	// but only when the mount method is hostPath. This ensures that the pod is scheduled
	// only on nodes where odiglet is already installed.
	// For the device mount method, this is unnecessary because the device is guaranteed
	// to be present on the node before the pod is scheduled.
	if karpenterDisabled && mountIsHostPath {
		podutils.AddOdigletInstalledAffinity(pod)
	}

	volumeMounted := false
	waspSupported := false

	dirsToCopy := make(map[string]struct{})
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

		distroName := containerConfig.OtelDistroName
		distroMetadata := p.DistrosGetter.GetDistroByName(distroName)
		if distroMetadata == nil {
			return ErrUnknownDistroName
		}

		containerVolumeMounted, containerDirsToCopy, err := p.injectOdigosToContainer(containerConfig, podContainerSpec, *pw, serviceName, odigosConfiguration, distroMetadata, pod.OwnerReferences)
		if err != nil {
			return err
		}

		if distroMetadata.RuntimeAgent != nil && distroMetadata.RuntimeAgent.WaspSupported {
			waspSupported = true
		}

		volumeMounted = volumeMounted || containerVolumeMounted
		dirsToCopy = mergeMaps(dirsToCopy, containerDirsToCopy)
	}

	if mountMethod == common.K8sHostPathMountMethod && volumeMounted {
		// only mount the volume if at least one container has a volume to mount
		podswebhook.MountPodVolumeToHostPath(pod)
	}

	if odigosConfiguration.MountMethod != nil && *odigosConfiguration.MountMethod == common.K8sInitContainerMountMethod && volumeMounted {
		// only mount the volume if at least one container has a volume to mount
		podswebhook.MountPodVolumeToEmptyDir(pod)
		if len(dirsToCopy) > 0 {
			// Create the init container that will copy the directories to the empty dir based on dirsToCopy
			createInitContainer(pod, dirsToCopy, odigosConfiguration)
		}
	}

	if odigosConfiguration.WaspEnabled != nil && *odigosConfiguration.WaspEnabled && waspSupported && p.WaspMutator != nil {
		err = p.WaspMutator(pod, odigosConfiguration)
		if err != nil {
			return fmt.Errorf("failed to do wasp mutation: %w", err)
		}
	}

	// Inject ODIGOS environment variables and instrumentation device into all containers
	injectErr := p.injectOdigosInstrumentation(ctx, pod, &ic, pw, &odigosConfiguration)
	if injectErr != nil {
		return fmt.Errorf("%w: %v", ErrEnvVarInjection, injectErr)
	}

	if odigosConfiguration.UserInstrumentationEnvs != nil {
		podswebhook.InjectUserEnvForLang(&odigosConfiguration, pod, &ic)
	}

	// store the agents deployment value so we can later associate each pod with the instrumentation version.
	// we can pull only our pods into cache, and follow the lifecycle of the instrumentation process.
	pod.Labels[k8sconsts.OdigosAgentsMetaHashLabel] = ic.Spec.AgentsMetaHash

	return nil
}

func mergeMaps[T any](a, b map[string]T) map[string]T {
	for k, v := range b {
		a[k] = v
	}
	return a
}

func (p *PodsWebhook) podWorkload(ctx context.Context, pod *corev1.Pod, req admission.Request) (*k8sconsts.PodWorkload, error) {
	pw, err := workload.PodWorkloadObject(ctx, pod)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pod workload details from pod: %w", err)
	}
	if pw == nil {
		// for pods which are not managed by odigos supported workload
		return nil, nil
	}

	if pw.Namespace == "" {
		if req.Namespace != "" {
			// If the namespace is available in the admission request, set it in the podWorkload.Namespace.
			pw.Namespace = req.Namespace
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

func (p *PodsWebhook) injectOdigosToContainer(containerConfig *odigosv1.ContainerAgentConfig, podContainerSpec *corev1.Container,
	pw k8sconsts.PodWorkload, serviceName string, config common.OdigosConfiguration, distroMetadata *distro.OtelDistro, ownerReferences []metav1.OwnerReference) (bool, map[string]struct{}, error) {
	var err error

	// check for existing env vars so we don't introduce them again
	existingEnvNames := podswebhook.GetEnvVarNamesSet(podContainerSpec)

	// inject various kinds of distro environment variables
	existingEnvNames, err = podswebhook.InjectStaticEnvVarsToPodContainer(existingEnvNames, podContainerSpec, distroMetadata.EnvironmentVariables.StaticVariables, containerConfig.DistroParams)
	if err != nil {
		return false, nil, err
	}
	existingEnvNames = podswebhook.InjectOdigosK8sEnvVars(existingEnvNames, podContainerSpec, distroMetadata.Name, pw.Namespace)
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

	// agent span metrics configuration
	agentSpanMetricsEnabled := containerConfig.Metrics != nil && containerConfig.Metrics.SpanMetrics != nil
	supportsAgentSpanMetrics := distroMetadata.AgentMetrics != nil && distroMetadata.AgentMetrics.SpanMetrics != nil && distroMetadata.AgentMetrics.SpanMetrics.Supported
	otlpHttpMetricsEndpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d/v1/metrics", k8sconsts.OdigosNodeCollectorLocalTrafficServiceName, env.GetCurrentNamespace(), consts.OTLPHttpPort)
	if supportsAgentSpanMetrics && !agentSpanMetricsEnabled {
		// TODO(edenfed): This is an ugly hack to also collect jvm metrics via OTel when span metrics are enbabled
		// Its using the fact that only the distro we need it for has span metrics enabled
		// This should not go into main branch, but its a temp workaround.
		podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "OTEL_METRICS_EXPORTER", "otlp")
		podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "OTEL_EXPORTER_OTLP_METRICS_PROTOCOL", "http/protobuf")
		podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "OTEL_EXPORTER_OTLP_METRICS_ENDPOINT", otlpHttpMetricsEndpoint)
	}

	if agentSpanMetricsEnabled && supportsAgentSpanMetrics && distroMetadata.ConfigAsEnvVars {
		// serialize span metrics config to json and inject as env var
		spanMetricsConfigJson, err := json.Marshal(containerConfig.Metrics.SpanMetrics)
		if err != nil {
			return false, nil, fmt.Errorf("failed to marshal span metrics config: %w", err)
		}
		existingEnvNames = podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "ODIGOS_AGENT_SPAN_METRICS_CONFIG", string(spanMetricsConfigJson))
		podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "ODIGOS_EXPORTER_OTLP_METRICS_ENDPOINT", otlpHttpMetricsEndpoint)
	}

	// URL Templatization configuration
	urlTemplatizationEnabled := containerConfig.Traces != nil && containerConfig.Traces.UrlTemplatization != nil && len(containerConfig.Traces.UrlTemplatization.Rules) > 0
	supportsUrlTemplatization := distroMetadata.Traces != nil && distroMetadata.Traces.UrlTemplatization != nil && distroMetadata.Traces.UrlTemplatization.Supported
	if urlTemplatizationEnabled && supportsUrlTemplatization && distroMetadata.ConfigAsEnvVars {
		// parse URL templatization config to json using the existing AgentTracesConfig struct
		urlTemplatizationConfigJson, err := json.Marshal(containerConfig.Traces.UrlTemplatization)
		if err != nil {
			return false, nil, fmt.Errorf("failed to marshal URL templatization config: %w", err)
		}
		existingEnvNames = podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "ODIGOS_AGENT_URL_TEMPLATIZATION", string(urlTemplatizationConfigJson))
	}

	// Head Sampling configuration
	headSamplingEnabled := containerConfig.Traces != nil && containerConfig.Traces.HeadSampling != nil
	supportsHeadSampling := distroMetadata.Traces != nil && distroMetadata.Traces.HeadSampling != nil && distroMetadata.Traces.HeadSampling.Supported
	if headSamplingEnabled && supportsHeadSampling && distroMetadata.ConfigAsEnvVars {
		// serialize head sampling config to json and inject as env var
		headSamplingConfigJson, err := json.Marshal(containerConfig.Traces.HeadSampling)
		if err != nil {
			return false, nil, fmt.Errorf("failed to marshal head sampling config: %w", err)
		}
		existingEnvNames = podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "ODIGOS_AGENT_HEAD_SAMPLING", string(headSamplingConfigJson))
	}

	// Span Renamer configuration
	spanRenamerEnabled := containerConfig.Traces != nil && containerConfig.Traces.SpanRenamer != nil
	supportsSpanRenamer := distroMetadata.Traces != nil && distroMetadata.Traces.SpanRenamer != nil && distroMetadata.Traces.SpanRenamer.Supported
	if spanRenamerEnabled && supportsSpanRenamer && distroMetadata.ConfigAsEnvVars {
		// serialize span renamer config to json and inject as env var
		spanRenamerConfigJson, err := json.Marshal(containerConfig.Traces.SpanRenamer)
		if err != nil {
			return false, nil, fmt.Errorf("failed to marshal span renamer config: %w", err)
		}
		existingEnvNames = podswebhook.InjectConstEnvVarToPodContainer(existingEnvNames, podContainerSpec, "ODIGOS_AGENT_SPAN_RENAMER", string(spanRenamerConfigJson))
	}

	volumeMounted := false
	containerDirsToCopy := make(map[string]struct{})
	if distroMetadata.RuntimeAgent != nil {
		if *config.MountMethod == common.K8sHostPathMountMethod || *config.MountMethod == common.K8sInitContainerMountMethod {
			// mount directory only if the mount type is host-path or init container
			for _, agentDirectoryName := range distroMetadata.RuntimeAgent.DirectoryNames {
				containerDirsToCopy[agentDirectoryName] = struct{}{}
				podswebhook.MountDirectory(podContainerSpec, agentDirectoryName)
				volumeMounted = true
			}

			// if loader is enabled, mount the loader directory
			if config.AgentEnvVarsInjectionMethod != nil && distroMetadata.RuntimeAgent.LdPreloadInjectionSupported &&
				(*config.AgentEnvVarsInjectionMethod == common.LoaderFallbackToPodManifestInjectionMethod ||
					*config.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod) {
				containerDirsToCopy[filepath.Join(distro.AgentPlaceholderDirectory, consts.OdigosLoaderDirName)] = struct{}{}
				podswebhook.MountDirectory(podContainerSpec, filepath.Join(k8sconsts.OdigosAgentsDirectory, consts.OdigosLoaderDirName))
				volumeMounted = true
			}
		}

		if distroMetadata.RuntimeAgent.K8sAttrsViaEnvVars {
			podswebhook.InjectOtelResourceAndServiceNameEnvVars(existingEnvNames, podContainerSpec, distroMetadata.Name, pw, serviceName, ownerReferences)
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

	return volumeMounted, containerDirsToCopy, nil
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

func createInitContainer(pod *corev1.Pod, dirsToCopy map[string]struct{}, config common.OdigosConfiguration) {
	const (
		instrumentationsPath = "/instrumentations"
	)

	imageName := getInitContainerImage(config)

	var copyCommands []string

	// Sort the map keys to ensure deterministic order.
	// This is important only for tests due to limitations,
	// which require consistent command ordering for reliable assertions.
	var dirs []string
	for dir := range dirsToCopy {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		from := strings.ReplaceAll(dir, distro.AgentPlaceholderDirectory, instrumentationsPath)
		to := strings.ReplaceAll(dir, distro.AgentPlaceholderDirectory, k8sconsts.OdigosAgentsDirectory)
		copyCommands = append(copyCommands, fmt.Sprintf("cp -r %s %s", from, to))
	}

	// The init container uses 'sh -c' to run multiple 'cp' commands in sequence.
	// Each 'cp -r <src> <dst>' copies agent directories from the image's /instrumentations/
	// into the shared /var/odigos volume (an EmptyDir). This allows sidecar injection of
	// required binaries without writing to the host filesystem.
	falseConst := false
	agentInitContainer := corev1.Container{
		Name:            k8sconsts.OdigosInitContainerName,
		Image:           imageName,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"sh",
			"-c",
			strings.Join(copyCommands, " && "),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      k8sconsts.OdigosAgentMountVolumeName,
				MountPath: k8sconsts.OdigosAgentsDirectory,
			},
		},
		// explicitly set the privileged field and the allowPrivilegedEscalation fields to false
		// some security policies may require these fields to be explicitly set to false, and we don't need special permission in this container
		SecurityContext: &corev1.SecurityContext{
			Privileged:               &falseConst,
			AllowPrivilegeEscalation: &falseConst,
		},
	}

	// Set resource limits and requests for the instrumentation init container
	// We can always trust the values from the effective config, because it is validated and defaulted if not ok in the scheduler.
	cpuRequestQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%dm", config.AgentsInitContainerResources.RequestCPUm))
	memoryRequestQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%dMi", config.AgentsInitContainerResources.RequestMemoryMiB))
	cpuLimitQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%dm", config.AgentsInitContainerResources.LimitCPUm))
	memoryLimitQuantity, _ := resource.ParseQuantity(fmt.Sprintf("%dMi", config.AgentsInitContainerResources.LimitMemoryMiB))
	agentInitContainer.Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			"cpu":    cpuRequestQuantity,
			"memory": memoryRequestQuantity,
		},
		Limits: corev1.ResourceList{
			"cpu":    cpuLimitQuantity,
			"memory": memoryLimitQuantity,
		},
	}
	// Check if the init container already exists, this is done for safety and should never happen.
	for _, existing := range pod.Spec.InitContainers {
		if existing.Name == k8sconsts.OdigosInitContainerName {
			return
		}
	}
	pod.Spec.InitContainers = append(pod.Spec.InitContainers, agentInitContainer)
}

func getInitContainerImage(config common.OdigosConfiguration) string {
	initContainerImage := k8sconsts.OdigosInitContainerImage
	imageVersion := os.Getenv(consts.OdigosVersionEnvVarName)

	// In the installation/upgrade we always set the init container image as env var, so we can use it here
	if initContainerImageEnv, ok := os.LookupEnv(k8sconsts.OdigosInitContainerEnvVarName); ok {
		return initContainerImageEnv
	}

	// This is a fallback for the case where the init container image is not set as env var for some reason.
	return config.ImagePrefix + "/" + initContainerImage + ":" + imageVersion
}
