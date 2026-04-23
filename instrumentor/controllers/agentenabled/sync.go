package agentenabled

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapi "github.com/odigos-io/odigos/common/api"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/distroresolver"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/dynamicconfig"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signals"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type agentInjectedStatusCondition struct {
	// The status represents 3 possible states for the agent injected:
	// - True: agent is injected (e.g. instrumentation config is updated to inject the agent in webhook)
	// - False: agent is not injected permanently (e.g. no supported agent can be injected to this workload due to unsupported runtimes, ignored containers, etc.)
	// - Unknown: agent is not injected and the state is transient (e.g. waiting for runtime inspection to complete, for collector to be ready, etc.)
	Status metav1.ConditionStatus

	// The reason represents the reason why the agent is not injected in an enum closed set of values.
	// use the AgentInjectionReason constants to set the value to the appropriate reason.
	Reason odigosv1.AgentEnabledReason

	// Human-readable message for the condition. it will show up in the ui and tools,
	// and should describe any additional context for the condition in free-form text.
	Message string
}

func reconcileAll(ctx context.Context, c client.Client, dp *distros.Provider, rolloutConcurrencyLimiter *rollout.RolloutConcurrencyLimiter) (ctrl.Result, error) {
	allInstrumentationConfigs := odigosv1.InstrumentationConfigList{}
	listErr := c.List(ctx, &allInstrumentationConfigs)
	if listErr != nil {
		return ctrl.Result{}, listErr
	}

	conf, err := k8sutils.GetCurrentOdigosConfiguration(ctx, c)
	if err != nil {
		return ctrl.Result{}, err
	}

	var allErrs error
	aggregatedResult := ctrl.Result{}
	for _, ic := range allInstrumentationConfigs.Items {
		res, workloadErr := reconcileWorkload(ctx, c, ic.Name, ic.Namespace, dp, &conf, rolloutConcurrencyLimiter)
		if workloadErr != nil {
			allErrs = errors.Join(allErrs, workloadErr)
		}
		if !res.IsZero() {
			if aggregatedResult.RequeueAfter == 0 {
				aggregatedResult.RequeueAfter = res.RequeueAfter
			} else if res.RequeueAfter < aggregatedResult.RequeueAfter {
				aggregatedResult.RequeueAfter = res.RequeueAfter
			}
		}
	}

	return aggregatedResult, allErrs
}

func reconcileWorkload(ctx context.Context, c client.Client, icName string, namespace string, distroProvider *distros.Provider, conf *common.OdigosConfiguration, rolloutConcurrencyLimiter *rollout.RolloutConcurrencyLimiter) (ctrl.Result, error) {
	logger := commonlogger.FromContext(ctx)

	pw, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(icName, namespace)
	if err != nil {
		logger.Error(err, "error parsing workload info from runtime object name")
		return ctrl.Result{}, nil // return nil so not to retry
	}

	ic := odigosv1.InstrumentationConfig{}
	err = c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// instrumentation config is deleted, trigger a rollout for the associated workload
			// this should happen once per workload, as the instrumentation config is deleted
			rolloutResult, doErr := rollout.Do(ctx, c, nil, pw, conf, distroProvider, rolloutConcurrencyLimiter)
			return rolloutResult.Result, doErr
		}
		return ctrl.Result{}, err
	}
	logger.Info("Reconciling workload for InstrumentationConfig object agent enabling", "name", ic.Name, "namespace", ic.Namespace, "instrumentationConfigName", ic.Name)

	condition, err := updateInstrumentationConfigSpec(ctx, c, pw, &ic, distroProvider, conf)
	if err != nil {
		return ctrl.Result{}, err
	}

	err = c.Update(ctx, &ic)
	if err != nil {
		return utils.K8SUpdateErrorHandler(err)
	}

	cond := metav1.Condition{
		Type:    odigosv1.AgentEnabledStatusConditionType,
		Status:  condition.Status,
		Reason:  string(condition.Reason),
		Message: condition.Message,
	}

	agentEnabledChanged := meta.SetStatusCondition(&ic.Status.Conditions, cond)
	rolloutResult, doErr := rollout.Do(ctx, c, &ic, pw, conf, distroProvider, rolloutConcurrencyLimiter)

	if rolloutResult.StatusChanged || agentEnabledChanged {
		updateErr := c.Status().Update(ctx, &ic)
		if updateErr != nil {
			return utils.K8SUpdateErrorHandler(updateErr)
		}
	}

	return rolloutResult.Result, doErr
}

func updateInstrumentationConfigAgentsMetaHash(ic *odigosv1.InstrumentationConfig, newValue string) {
	if ic.Spec.AgentsMetaHash == newValue {
		return
	}
	ic.Spec.AgentsMetaHash = newValue
	ic.Spec.AgentsMetaHashChangedTime = &metav1.Time{Time: time.Now()}
}

// this function receives a workload object, and updates the instrumentation config object ptr.
// if the function returns without an error, it means the instrumentation config object was updated.
// caller should persist the object to the API server.
// if the function returns without an error, it also returns an agentInjectedStatusCondition object
// which records what should be written to the status.conditions field of the instrumentation config
// and later be used for viability and monitoring purposes.
func updateInstrumentationConfigSpec(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload, ic *odigosv1.InstrumentationConfig, distroProvider *distros.Provider, effectiveConfig *common.OdigosConfiguration) (*agentInjectedStatusCondition, error) {
	logger := commonlogger.FromContext(ctx)
	cg, irls, agentLevelActions, samplingRules, workloadObj, err := getRelevantResources(ctx, c, pw)
	if err != nil {
		// error of fetching one of the resources, retry
		return nil, err
	}

	// Check for workloads that are in backoff state and not eligible for instrumentation.
	backoffCondition, err := hasUninstrumentedPodsWithBackoff(ctx, c, pw, ic, logger)
	if err != nil {
		return nil, err
	}
	if backoffCondition != nil {
		// Set the WorkloadRollout condition
		meta.SetStatusCondition(&ic.Status.Conditions, metav1.Condition{
			Type:    odigosv1.WorkloadRolloutStatusConditionType,
			Status:  metav1.ConditionFalse,
			Reason:  string(odigosv1.AgentEnabledReasonCrashLoopBackOff),
			Message: "Workload has pods in backoff state - not eligible for instrumentation",
		})
		return backoffCondition, nil
	}

	// check if we are waiting for some transient prerequisites to be completed before injecting the agent
	prerequisiteCompleted, reason, message := isReadyForInstrumentation(cg, ic)
	if !prerequisiteCompleted {
		ic.Spec.AgentInjectionEnabled = false
		updateInstrumentationConfigAgentsMetaHash(ic, "")
		ic.Spec.Containers = []odigosv1.ContainerAgentConfig{}
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionUnknown,
			Reason:  reason,
			Message: message,
		}, nil
	}

	defaultDistrosPerLanguage := distroProvider.GetDefaultDistroNames()
	distroPerLanguage := distroresolver.CalculateDefaultDistroPerLanguage(defaultDistrosPerLanguage, irls, distroProvider.Getter)

	// If the source was already marked for instrumentation, but has caused a CrashLoopBackOff or ImagePullBackOff we'd like to stop
	// instrumentating it and to disable future instrumentation of this service.
	// Recovery from rollback is already handled in reconcileWorkload before this function is called.
	rollbackOccurred := ic.Status.RollbackOccurred
	// Get existing backoff reason from status conditions if available
	var existingBackoffReason odigosv1.AgentEnabledReason
	for _, condition := range ic.Status.Conditions {
		if condition.Type == odigosv1.AgentEnabledStatusConditionType {
			reason := odigosv1.AgentEnabledReason(condition.Reason)
			if reason == odigosv1.AgentEnabledReasonCrashLoopBackOff || reason == odigosv1.AgentEnabledReasonImagePullBackOff {
				existingBackoffReason = reason
				break
			}
		}
	}
	// If not found in conditions, check existing container configs
	if existingBackoffReason == "" {
		for _, container := range ic.Spec.Containers {
			if container.AgentEnabledReason == odigosv1.AgentEnabledReasonCrashLoopBackOff || container.AgentEnabledReason == odigosv1.AgentEnabledReasonImagePullBackOff {
				existingBackoffReason = container.AgentEnabledReason
				break
			}
		}
	}
	// If not found in containers and we are in rollback state, default to CrashLoopBackOff
	if rollbackOccurred && existingBackoffReason == "" {
		existingBackoffReason = odigosv1.AgentEnabledReasonCrashLoopBackOff
	}
	containersConfig := make([]odigosv1.ContainerAgentConfig, 0, len(ic.Spec.Containers))
	collectorConfigs := make([]commonapi.ContainerCollectorConfig, 0, len(ic.Spec.Containers))
	runtimeDetailsByContainer := ic.RuntimeDetailsByContainer()
	podManifestInjectionOptional := true // pod manifest is optional, unless some container agent requires it

	for containerName, containerRuntimeDetails := range runtimeDetailsByContainer {

		// at this point, containerRuntimeDetails can be nil, indicating we have no runtime details for this container
		// from automatic runtime detection or overrides.
		containerOverride := ic.GetOverridesForContainer(containerName)
		containerDistro, err := distroresolver.ResolveDistroForContainer(effectiveConfig, containerRuntimeDetails, distroPerLanguage, distroProvider.Getter, containerOverride, containerName)
		if err != nil {
			// if we cannot match a distro for the container, we cannot instrument it.
			// mark this in the container agent config and avoid any further processing.
			containersConfig = append(containersConfig, odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  err.AgentEnabledReason,
				AgentEnabledMessage: err.AgentEnabledMessage,
			})
			continue
		}

		// calculate and verify there are enabled signals for this container.
		enabledSignals, disabledInfo := signals.GetEnabledSignalsForContainer(cg, irls)
		if disabledInfo != nil {
			containersConfig = append(containersConfig, odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  disabledInfo.AgentEnabledReason,
				AgentEnabledMessage: disabledInfo.AgentEnabledMessage,
			})
			continue
		}

		// calculate the dynamic configs for this container.
		dynamicContainerConfigs, disabledInfo := dynamicconfig.CalculateDynamicContainerConfig(containerName, irls, effectiveConfig, containerRuntimeDetails, agentLevelActions, samplingRules, workloadObj, pw, containerDistro, enabledSignals)
		if disabledInfo != nil {
			containersConfig = append(containersConfig, odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  disabledInfo.AgentEnabledReason,
				AgentEnabledMessage: disabledInfo.AgentEnabledMessage,
			})
			continue
		}

		agentConfig := calculateContainerAgentConfig(containerName, containerDistro, effectiveConfig, containerRuntimeDetails, rollbackOccurred, existingBackoffReason)
		// add the dynamic agent configs to enabled agents
		if agentConfig.AgentEnabled {
			agentConfig.Traces = dynamicContainerConfigs.AgentTracesConfig
			agentConfig.Metrics = dynamicContainerConfigs.AgentMetricsConfig
			agentConfig.Logs = dynamicContainerConfigs.AgentLogsConfig
		}
		containersConfig = append(containersConfig, agentConfig)

		if agentConfig.AgentEnabled {
			// add collector config for the container if there is any
			if dynamicContainerConfigs.CollectorConfig != nil {
				dynamicContainerConfigs.CollectorConfig.ContainerName = containerName
				collectorConfigs = append(collectorConfigs, *dynamicContainerConfigs.CollectorConfig)
			}

			// if at least one container has agent enabled, and pod manifest injection is required,
			// then the overall pod manifest injection is required.
			if !agentConfig.PodManifestInjectionOptional {
				podManifestInjectionOptional = false
			}
		}
	}

	ic.Spec.Containers = containersConfig
	ic.Spec.PodManifestInjectionOptional = podManifestInjectionOptional
	ic.Spec.WorkloadCollectorConfig = collectorConfigs
	// after updating the container configs, we can go over them and produce a useful aggregated status for the user
	// if any container is instrumented, we can set the status to true
	// if all containers are not instrumented, we can set the status to false and provide a reason
	// notice the reason is an aggregate of the different containers, so it might not be the most accurate in edge cases.
	// but it should be good enough for the user to understand why the agent is not injected.
	// at this point we know we have containers, since the runtime is completed.
	aggregatedCondition := containerConfigToStatusCondition(ic.Spec.Containers[0])
	instrumentedContainerNames := []string{}
	for _, containerConfig := range ic.Spec.Containers {
		if containerConfig.AgentEnabled {
			instrumentedContainerNames = append(instrumentedContainerNames, containerConfig.ContainerName)
		}
		if odigosv1.AgentInjectionReasonPriority(containerConfig.AgentEnabledReason) > odigosv1.AgentInjectionReasonPriority(aggregatedCondition.Reason) {
			// set to the most specific (highest priority) reason from multiple containers.
			aggregatedCondition = containerConfigToStatusCondition(containerConfig)
		}
	}
	if len(instrumentedContainerNames) > 0 {
		// if any instrumented containers are found, the pods webhook should process pods for this workload.
		// set the AgentInjectionEnabled to true to signal that.
		ic.Spec.AgentInjectionEnabled = !rollbackOccurred
		ic.Spec.PodManifestInjectionOptional = ic.Spec.AgentInjectionEnabled && podManifestInjectionOptional
		agentsDeploymentHash, err := rollout.HashForContainersConfig(containersConfig)
		if err != nil {
			return nil, err
		}
		updateInstrumentationConfigAgentsMetaHash(ic, string(agentsDeploymentHash))
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionTrue,
			Reason:  odigosv1.AgentEnabledReasonEnabledSuccessfully,
			Message: fmt.Sprintf("agent enabled in %d containers: %v", len(instrumentedContainerNames), instrumentedContainerNames),
		}, nil
	} else {
		// if none of the containers are instrumented, we can set the status to false
		// to signal to the webhook that those pods should not be processed.
		ic.Spec.AgentInjectionEnabled = false
		ic.Spec.PodManifestInjectionOptional = false
		updateInstrumentationConfigAgentsMetaHash(ic, "")
		return aggregatedCondition, nil
	}
}

// hasUninstrumentedPodsWithBackoff checks if the workload has pods in backoff state before instrumentation.
func hasUninstrumentedPodsWithBackoff(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload, ic *odigosv1.InstrumentationConfig, logger *commonlogger.ContextLogger) (*agentInjectedStatusCondition, error) {
	// CronJob and Job workloads don't have a label selector like Deployments/StatefulSets/DaemonSets,
	// so we skip the backoff check for them. Their pods are managed differently through the Job controller.
	if pw.Kind == k8sconsts.WorkloadKindCronJob || pw.Kind == k8sconsts.WorkloadKindJob {
		return nil, nil
	}

	workloadClientObj := workload.ClientObjectFromWorkloadKind(pw.Kind)
	if getErr := c.Get(ctx, client.ObjectKey{Name: pw.Name, Namespace: pw.Namespace}, workloadClientObj); getErr == nil {
		hasPodInBackoff, backoffErr := rollout.WorkloadHasNonInstrumentedPodInBackoff(ctx, c, workloadClientObj)
		if backoffErr != nil {
			logger.Debug("failed to check for pods in backoff", "err", backoffErr, "workload", pw.Name, "namespace", pw.Namespace)
			return nil, fmt.Errorf("failed to check for pods in backoff: %w", backoffErr)
		}
		if hasPodInBackoff {
			logger.Debug("workload has pods in backoff state", "workload", pw.Name, "namespace", pw.Namespace)
			return &agentInjectedStatusCondition{
				Status:  metav1.ConditionFalse,
				Reason:  odigosv1.AgentEnabledReasonCrashLoopBackOff,
				Message: "Workload has pods in backoff state before instrumentation - cannot instrument crashlooping workload",
			}, nil
		}
	}
	return nil, nil
}

func containerConfigToStatusCondition(containerConfig odigosv1.ContainerAgentConfig) *agentInjectedStatusCondition {
	if containerConfig.AgentEnabled {
		// no expecting to hit this case, but for completeness
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionTrue,
			Reason:  odigosv1.AgentEnabledReasonEnabledSuccessfully,
			Message: fmt.Sprintf("agent enabled for container %s", containerConfig.ContainerName),
		}
	} else {
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionFalse,
			Reason:  containerConfig.AgentEnabledReason,
			Message: containerConfig.AgentEnabledMessage,
		}
	}
}

func getEnvVarFromList(envVars []odigosv1.EnvVar, envVarName string) (string, bool) {
	// here we check for the value of LD_PRELOAD in the EnvVars list,
	// which returns the env as read from /proc to make sure if there is any value set,
	// via any mechanism (manifest, device, script, other agent, etc.) then we are aware.
	for _, envVar := range envVars {
		if envVar.Name == envVarName {
			return envVar.Value, true
		}
	}
	return "", false
}

// will check if loader injection is supported based on the runtime inspection.
// loader is only allowed if:
// - LD_PRELOAD is not used in the container for some other purpose (other agent)
// - container is not running in secure execution mode
//
// returns "nil" on success, and a agent disabled info with reason and message if not supported.
func isLoaderInjectionSupportedByRuntimeDetails(runtimeDetails *odigosv1.RuntimeDetailsByContainer) *odigosv1.AgentDisabledInfo {
	// check for conditions to inject ldpreload when it is the only method configured.
	secureExecution := runtimeDetails.SecureExecutionMode == nil || *runtimeDetails.SecureExecutionMode
	if secureExecution {
		return &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: "container is running in secure execution mode and injection method is set to 'loader'",
		}
	}

	odigosLoaderPath := filepath.Join(k8sconsts.OdigosAgentsDirectory, commonconsts.OdigosLoaderDirName, commonconsts.OdigosLoaderName)
	ldPreloadVal, ldPreloadFoundInInspection := getEnvVarFromList(runtimeDetails.EnvVars, commonconsts.LdPreloadEnvVarName)
	ldPreloadUnsetOrExpected := !ldPreloadFoundInInspection || strings.Contains(ldPreloadVal, odigosLoaderPath)
	if !ldPreloadUnsetOrExpected {
		return &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: "container is already using LD_PRELOAD env var, and injection method is set to 'loader'. current value: " + ldPreloadVal,
		}
	}

	return nil
}

// Will calculate the env injection method for the container based on the relevant parameters.
// returned paramters are:
// - the env injection method to use for this container (may be nil if no injection should take place)
// - a container agent config to signal any failures in using the loader in these conditions.
//
// second returned value acts as an "error" value, user should first check if it's not nil and handle any errors accordingly.
func getEnvInjectionDecision(
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	distro *distro.OtelDistro,
) (*common.EnvInjectionDecision, *odigosv1.AgentDisabledInfo) {
	if effectiveConfig.AgentEnvVarsInjectionMethod == nil {
		// this should never happen, as the config is reconciled with default value.
		// it is only here as a safety net.
		return nil, &odigosv1.AgentDisabledInfo{
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: "no injection method configured for odigos agent",
		}
	}

	// If we should try loader, check for this first
	distroSupportsLoader := distro.RuntimeAgent != nil && distro.RuntimeAgent.LdPreloadInjectionSupported
	loaderRequested := (*effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod || *effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderFallbackToPodManifestInjectionMethod)

	if distroSupportsLoader && loaderRequested {
		disabledInfo := isLoaderInjectionSupportedByRuntimeDetails(runtimeDetails)
		if disabledInfo != nil {
			// loader is requested by config and distro, but not supported by the runtime details.
			if *effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod {
				// config requires us to use loader when it is supported by distro,
				// thus we can't use it and need fail the injection.
				return nil, disabledInfo
			} // else - we will not return and continue to the next injection method.
		} else {
			// loader is requested by config and distro, and supported by the runtime details.
			// thus, we can use the loader injection method in webhook.
			loaderInjectionMethod := common.EnvInjectionDecisionLoader
			return &loaderInjectionMethod, nil
		}
	}

	// at this point, we know that either:
	// - user configured to use pod manifest injection, or
	// - user requested loader fallback to pod manifest, and we are at the fallback stage.
	distroHasAppendEnvVar := len(distro.EnvironmentVariables.AppendOdigosVariables) > 0
	if !distroHasAppendEnvVar {
		// this is a common case, where a distro doesn't support nor loader or append env var injection.
		// at the time of writing, this is golang, dotnet, php, ruby.
		// for those we mark env injection as nil to denote "no injection"
		// and return err as nil to denote "no error".
		return nil, nil
	}

	envInjectionDecision := common.EnvInjectionDecisionPodManifest
	return &envInjectionDecision, nil
}

func calculateContainerAgentConfig(containerName string,
	d *distro.OtelDistro,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	rollbackOccurred bool,
	existingBackoffReason odigosv1.AgentEnabledReason,
) odigosv1.ContainerAgentConfig {

	distroName := d.Name

	envInjectionDecision, envInjectionDisabledInfo := getEnvInjectionDecision(effectiveConfig, runtimeDetails, d)
	if envInjectionDisabledInfo != nil {
		// if we have a container agent config with reason and message, we return it.
		// this is a failure to inject the agent, and we should not proceed with other checks.
		return odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  envInjectionDisabledInfo.AgentEnabledReason,
			AgentEnabledMessage: envInjectionDisabledInfo.AgentEnabledMessage,
		}
	}

	distroParameters, err := calculateDistroParams(d, runtimeDetails, envInjectionDecision)
	if err != nil {
		return *err
	}

	if rollbackOccurred {
		message := fmt.Sprintf("Pods entered %s; instrumentation disabled", existingBackoffReason)
		return odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  existingBackoffReason,
			AgentEnabledMessage: message,
			OtelDistroName:      distroName,
			DistroParams:        distroParameters,
		}
	}

	podManifestInjectionOptional := !distro.IsRestartRequired(d, effectiveConfig)

	// check for presence of other agents
	if runtimeDetails.OtherAgent != nil {
		if effectiveConfig.AllowConcurrentAgents == nil || !*effectiveConfig.AllowConcurrentAgents {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonOtherAgentDetected,
				AgentEnabledMessage: fmt.Sprintf("odigos agent not enabled due to other instrumentation agent '%s' detected running in the container", runtimeDetails.OtherAgent.Name),
			}
		}
		return odigosv1.ContainerAgentConfig{
			ContainerName:                containerName,
			AgentEnabled:                 true,
			PodManifestInjectionOptional: podManifestInjectionOptional,
			AgentEnabledReason:           odigosv1.AgentEnabledReasonEnabledSuccessfully,
			AgentEnabledMessage:          fmt.Sprintf("we are operating alongside the %s, which is not the recommended configuration. We suggest disabling the %s for optimal performance.", runtimeDetails.OtherAgent.Name, runtimeDetails.OtherAgent.Name),
			OtelDistroName:               distroName,
			DistroParams:                 distroParameters,
			EnvInjectionMethod:           envInjectionDecision,
		}
	}

	return odigosv1.ContainerAgentConfig{
		ContainerName:                containerName,
		AgentEnabled:                 true,
		PodManifestInjectionOptional: podManifestInjectionOptional,
		OtelDistroName:               distroName,
		DistroParams:                 distroParameters,
		EnvInjectionMethod:           envInjectionDecision,
	}
}

// This function checks if we are waiting for some transient prerequisites to be completed before injecting the agent.
// We can't really know for sure if something is transient or permanent,
// but the assumption is that if we wait it should eventually be resolve.
//
// Things we are waiting for before injecting the agent:
//
// 1. Node collector to be ready (so there is a receiver for the telemetry)
//   - Can be transient state, in case the node collector is starting
//   - Can be permanent state (image pull error, lack of resources, etc.)
//
// 2. No runtime details for this workload
//   - Can be transient state, until odiglet calculates the runtime details in a short while
//   - Can be permanent state (odiglet is not running, workload has not running pods, etc.)
//
// The function returns
// - waitingForPrerequisites: true if we are waiting for some prerequisites to be completed before injecting the agent
// - reason: AgentInjectionReason enum value that represents the reason why we are waiting
// - message: human-readable message that describes the reason why we are waiting
func isReadyForInstrumentation(cg *odigosv1.CollectorsGroup, ic *odigosv1.InstrumentationConfig) (bool, odigosv1.AgentEnabledReason, string) {
	// Check if the node collector is ready
	isNodeCollectorReady, message := isNodeCollectorReady(cg)
	if !isNodeCollectorReady {
		return false, odigosv1.AgentEnabledReasonWaitingForNodeCollector, message
	}

	gotReadySignals, message := gotReadySignals(cg)
	if !gotReadySignals {
		return false, odigosv1.AgentEnabledReasonNoCollectedSignals, message
	}

	// if there are any overrides, we use them (and exist early)
	for _, containerOverride := range ic.Spec.ContainersOverrides {
		if containerOverride.RuntimeInfo != nil {
			return true, odigosv1.AgentEnabledReasonEnabledSuccessfully, ""
		}
	}

	hasAutomaticRuntimeDetection := len(ic.Status.RuntimeDetailsByContainer) > 0
	if !hasAutomaticRuntimeDetection {
		// differentiate between the case where we expect runtime detection to be completed soon,
		// vs the case where we know it is staled due to no running pods preventing the runtime inspection
		for _, condition := range ic.Status.Conditions {
			if condition.Type == odigosv1.RuntimeDetectionStatusConditionType {
				if odigosv1.RuntimeDetectionReason(condition.Reason) == odigosv1.RuntimeDetectionReasonNoRunningPods {
					return false, odigosv1.AgentEnabledReasonRuntimeDetailsUnavailable, "agent will be enabled once runtime details from running pods is available"
				}
			}
		}
		return false, odigosv1.AgentEnabledReasonWaitingForRuntimeInspection, "waiting for runtime inspection to complete"
	}

	// report success if both prerequisites are completed
	return true, odigosv1.AgentEnabledReasonEnabledSuccessfully, ""
}

func isNodeCollectorReady(cg *odigosv1.CollectorsGroup) (bool, string) {
	if cg == nil {
		return false, "waiting for OpenTelemetry Collector to be created"
	}

	if !cg.Status.Ready {
		return false, "waiting for OpenTelemetry Collector to be ready"
	}

	// node collector is ready to receive telemetry
	return true, ""
}

func gotReadySignals(cg *odigosv1.CollectorsGroup) (bool, string) {
	if cg == nil {
		return false, "waiting for OpenTelemetry Collector to be created"
	}

	if len(cg.Status.ReceiverSignals) == 0 {
		return false, "no signals are being collected"
	}

	return true, ""
}
