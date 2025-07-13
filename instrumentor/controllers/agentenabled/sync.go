package agentenabled

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func reconcileAll(ctx context.Context, c client.Client, dp *distros.Provider) (ctrl.Result, error) {
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
		res, workloadErr := reconcileWorkload(ctx, c, ic.Name, ic.Namespace, dp, &conf)
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

func reconcileWorkload(ctx context.Context, c client.Client, icName string, namespace string, distroProvider *distros.Provider, conf *common.OdigosConfiguration) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

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
			_, res, err := rollout.Do(ctx, c, nil, pw, conf)
			return res, err
		}
		return ctrl.Result{}, err
	}

	logger.Info("Reconciling workload for InstrumentationConfig object agent enabling", "name", ic.Name, "namespace", ic.Namespace, "instrumentationConfigName", ic.Name)

	condition, err := updateInstrumentationConfigSpec(ctx, c, pw, &ic, distroProvider)
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
	rolloutChanged, res, err := rollout.Do(ctx, c, &ic, pw, conf)

	if rolloutChanged || agentEnabledChanged {
		updateErr := c.Status().Update(ctx, &ic)
		if updateErr != nil {
			// if the update fails, we should not return an error, but rather log it and retry later.
			return utils.K8SUpdateErrorHandler(updateErr)
		}
	}

	return res, err
}

// this function receives a workload object, and updates the instrumentation config object ptr.
// if the function returns without an error, it means the instrumentation config object was updated.
// caller should persist the object to the API server.
// if the function returns without an error, it also returns an agentInjectedStatusCondition object
// which records what should be written to the status.conditions field of the instrumentation config
// and later be used for viability and monitoring purposes.
func updateInstrumentationConfigSpec(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload, ic *odigosv1.InstrumentationConfig, distroProvider *distros.Provider) (*agentInjectedStatusCondition, error) {
	cg, irls, effectiveConfig, err := getRelevantResources(ctx, c, pw)
	if err != nil {
		// error of fetching one of the resources, retry
		return nil, err
	}

	// check if we are waiting for some transient prerequisites to be completed before injecting the agent
	prerequisiteCompleted, reason, message := isReadyForInstrumentation(cg, ic)
	if !prerequisiteCompleted {
		ic.Spec.AgentInjectionEnabled = false
		ic.Spec.AgentsMetaHash = ""
		ic.Spec.Containers = []odigosv1.ContainerAgentConfig{}
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionUnknown,
			Reason:  reason,
			Message: message,
		}, nil
	}

	defaultDistrosPerLanguage := distroProvider.GetDefaultDistroNames()
	distroPerLanguage := calculateDefaultDistroPerLanguage(defaultDistrosPerLanguage, irls, distroProvider.Getter)

	// If the source was already marked for instrumentation, but has caused a CrashLoopBackOff we'd like to stop
	// instrumentating it and to disable future instrumentation of this service
	crashDetected := ic.Status.RollbackOccurred
	containersConfig := make([]odigosv1.ContainerAgentConfig, 0, len(ic.Spec.Containers))
	// ContainersOverrides will always list all containers of the workloads, so we can use it to iterate.
	for i := range ic.Spec.ContainersOverrides {
		containerName := ic.Spec.ContainersOverrides[i].ContainerName
		var containerRuntimeDetails *odigosv1.RuntimeDetailsByContainer
		// always take the override if it exists, before taking the automatic runtime detection.
		if ic.Spec.ContainersOverrides[i].RuntimeInfo != nil {
			containerRuntimeDetails = ic.Spec.ContainersOverrides[i].RuntimeInfo
		} else {
			// find this container by name in the automatic runtime detection
			for j := range ic.Status.RuntimeDetailsByContainer {
				if ic.Status.RuntimeDetailsByContainer[j].ContainerName == containerName {
					containerRuntimeDetails = &ic.Status.RuntimeDetailsByContainer[j]
					break
				}
			}
		}
		// at this point, containerRuntimeDetails can be nil, indicating we have no runtime details for this container
		// from automatic runtime detection or overrides.
		currentContainerConfig := calculateContainerInstrumentationConfig(containerName, effectiveConfig, containerRuntimeDetails, distroPerLanguage, distroProvider.Getter, crashDetected, cg, irls)
		containersConfig = append(containersConfig, currentContainerConfig)
	}
	ic.Spec.Containers = containersConfig

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
		ic.Spec.AgentInjectionEnabled = !crashDetected
		agentsDeploymentHash, err := rollout.HashForContainersConfig(containersConfig)
		if err != nil {
			return nil, err
		}
		ic.Spec.AgentsMetaHash = string(agentsDeploymentHash)
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionTrue,
			Reason:  odigosv1.AgentEnabledReasonEnabledSuccessfully,
			Message: fmt.Sprintf("agent enabled in %d containers: %v", len(instrumentedContainerNames), instrumentedContainerNames),
		}, nil
	} else {
		// if none of the containers are instrumented, we can set the status to false
		// to signal to the webhook that those pods should not be processed.
		ic.Spec.AgentInjectionEnabled = false
		ic.Spec.AgentsMetaHash = ""
		return aggregatedCondition, nil
	}
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

func getEnabledSignalsForContainer(nodeCollectorsGroup *odigosv1.CollectorsGroup, irls *[]odigosv1.InstrumentationRule) (tracesEnabled bool, metricsEnabled bool, logsEnabled bool) {
	tracesEnabled = false
	metricsEnabled = false
	logsEnabled = false

	if nodeCollectorsGroup == nil {
		// if the node collectors group is not created yet,
		// it means the collectors are not running thus all signals are disabled.
		return false, false, false
	}

	// first set each signal to enabled/disabled based on the node collectors group global signals for collection.
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.TracesObservabilitySignal) {
		tracesEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.MetricsObservabilitySignal) {
		metricsEnabled = true
	}
	if slices.Contains(nodeCollectorsGroup.Status.ReceiverSignals, common.LogsObservabilitySignal) {
		logsEnabled = true
	}

	// disable specific signals if they are disabled in any of the workload level instrumentation rules.
	for _, irl := range *irls {

		// these signals are in the workload level,
		// and library specific rules are not relevant to the current calculation.
		if irl.Spec.InstrumentationLibraries != nil {
			continue
		}

		// if any instrumentation rule has trace config disabled, we should disable traces for this container.
		// the list is already filtered to only include rules that are relevant to the current workload.
		if irl.Spec.TraceConfig != nil && irl.Spec.TraceConfig.Disabled != nil && *irl.Spec.TraceConfig.Disabled {
			tracesEnabled = false
		}
	}

	return tracesEnabled, metricsEnabled, logsEnabled
}

func getEnvVarFromRuntimeDetails(runtimeDetails *odigosv1.RuntimeDetailsByContainer, envVarName string) (string, bool) {
	// here we check for the value of LD_PRELOAD in the EnvVars list,
	// which returns the env as read from /proc to make sure if there is any value set,
	// via any mechanism (manifest, device, script, other agent, etc.) then we are aware.
	for _, envVar := range runtimeDetails.EnvVars {
		if envVar.Name == envVarName {
			return envVar.Value, true
		}
	}
	return "", false
}

func calculateContainerInstrumentationConfig(containerName string,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	distroPerLanguage map[common.ProgrammingLanguage]string,
	distroGetter *distros.Getter,
	crashDetected bool,
	nodeCollectorsGroup *odigosv1.CollectorsGroup,
	irls *[]odigosv1.InstrumentationRule) odigosv1.ContainerAgentConfig {

	tracesEnabled, metricsEnabled, logsEnabled := getEnabledSignalsForContainer(nodeCollectorsGroup, irls)

	// at this time, we don't populate the signals specific configs, but we will do it soon
	var tracesConfig *odigosv1.AgentTracesConfig
	var metricsConfig *odigosv1.AgentMetricsConfig
	var logsConfig *odigosv1.AgentLogsConfig
	if tracesEnabled {
		tracesConfig = &odigosv1.AgentTracesConfig{}
	}
	if metricsEnabled {
		metricsConfig = &odigosv1.AgentMetricsConfig{}
	}
	if logsEnabled {
		logsConfig = &odigosv1.AgentLogsConfig{}
	}

	// check if container is ignored by name, assuming IgnoredContainers is a short list.
	// This should be done first, because user should see workload not instrumented if container is ignored over unknown language in case both exist.
	for _, ignoredContainer := range effectiveConfig.IgnoredContainers {
		if ignoredContainer == containerName {
			return odigosv1.ContainerAgentConfig{
				ContainerName:      containerName,
				AgentEnabled:       false,
				AgentEnabledReason: odigosv1.AgentEnabledReasonIgnoredContainer,
			}
		}
	}

	if runtimeDetails == nil {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonRuntimeDetailsUnavailable,
		}
	}

	// check unknown language first. if language is not supported, we can skip the rest of the checks.
	if runtimeDetails.Language == common.UnknownProgrammingLanguage {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
		}
	}

	// check for deprecated "ignored" language
	// TODO: remove this in odigos v1.1
	if runtimeDetails.Language == common.IgnoredProgrammingLanguage {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonIgnoredContainer,
		}
	}

	// get relevant distroName for the detected language
	distroName, ok := distroPerLanguage[runtimeDetails.Language]
	if !ok {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonNoAvailableAgent,
		}
	}

	distro := distroGetter.GetDistroByName(distroName)
	if distro == nil {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonNoAvailableAgent,
		}
	}

	// if no signals are enabled, we don't need to inject the agent.
	// based on the rules, we can have a case where no signals are enabled for specific container.
	// TODO: check if this distro supports no signals enabled (instead of checking for any signal)
	if !tracesEnabled && !metricsEnabled && !logsEnabled {
		return odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonNoCollectedSignals,
			AgentEnabledMessage: "all signals are disabled, no agent will be injected",
		}
	}

	// check for injection method based on the distro and runtime details
	containsAppendEnvVar := len(distro.EnvironmentVariables.AppendOdigosVariables) > 0
	ldPreloadInjectionSupported := distro.RuntimeAgent != nil && distro.RuntimeAgent.LdPreloadInjectionSupported
	if containsAppendEnvVar && ldPreloadInjectionSupported &&
		(effectiveConfig.AgentEnvVarsInjectionMethod != nil && *effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod) {

		// check for conditions to inject ldpreload when it is the only method configured.
		secureExecution := runtimeDetails.SecureExecutionMode == nil || *runtimeDetails.SecureExecutionMode
		if secureExecution {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
				AgentEnabledMessage: "container is running in secure execution mode and injection method is set to 'loader'",
			}
		}

		// check if the LD_PRELOAD env var is not already present in the manifest and the runtime details env var is not set or set to the odigos loader path.
		odigosLoaderPath := filepath.Join(k8sconsts.OdigosAgentsDirectory, commonconsts.OdigosLoaderDirName, commonconsts.OdigosLoaderName)
		ldPreloadValue, foundInInspection := getEnvVarFromRuntimeDetails(runtimeDetails, "LD_PRELOAD")
		ldPreloadUnsetOrExpected := !foundInInspection || strings.Contains(ldPreloadValue, odigosLoaderPath)
		if !ldPreloadUnsetOrExpected {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
				AgentEnabledMessage: "container is already using LD_PRELOAD env var, and injection method is set to 'loader'. current value: " + ldPreloadValue,
			}
		}
	}

	// check if the runtime version is in supported range if it is provided
	if runtimeDetails.RuntimeVersion != "" && len(distro.RuntimeEnvironments) == 1 {
		constraint, err := version.NewConstraint(distro.RuntimeEnvironments[0].SupportedVersions)
		if err != nil {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("failed to parse supported versions constraint: %s", distro.RuntimeEnvironments[0].SupportedVersions),
			}
		}
		detectedVersion, err := version.NewVersion(runtimeDetails.RuntimeVersion)
		if err != nil {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version: %s", runtimeDetails.RuntimeVersion),
			}
		}
		if !constraint.Check(detectedVersion) {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("%s runtime not supported by OpenTelemetry. supported versions: '%s', found: %s", distro.RuntimeEnvironments[0].Name, constraint, detectedVersion),
			}
		}
	}

	distroParameters := map[string]string{}
	for _, parameterName := range distro.RequireParameters {
		switch parameterName {
		case common.LibcTypeDistroParameterName:
			if runtimeDetails.LibCType == nil {
				return odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
					AgentEnabledMessage: fmt.Sprintf("missing required parameter '%s' for distro '%s'", common.LibcTypeDistroParameterName, distroName),
				}
			}
			distroParameters[common.LibcTypeDistroParameterName] = string(*runtimeDetails.LibCType)

		case distroTypes.RuntimeVersionMajorMinorDistroParameterName:
			if runtimeDetails.RuntimeVersion == "" {
				return odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
					AgentEnabledMessage: fmt.Sprintf("missing required parameter '%s' for distro '%s'", distroTypes.RuntimeVersionMajorMinorDistroParameterName, distroName),
				}
			}
			version, err := version.NewVersion(runtimeDetails.RuntimeVersion)
			if err != nil {
				return odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
					AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version: %s", runtimeDetails.RuntimeVersion),
				}
			}
			versionAsMajorMinor, err := common.MajorMinorStringOnly(version)
			if err != nil {
				return odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
					AgentEnabledMessage: fmt.Sprintf("failed to parse runtime version as major.minor: %s", runtimeDetails.RuntimeVersion),
				}
			}
			distroParameters[distroTypes.RuntimeVersionMajorMinorDistroParameterName] = versionAsMajorMinor

		default:
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
				AgentEnabledMessage: fmt.Sprintf("unsupported parameter '%s' for distro '%s'", parameterName, distroName),
			}
		}
	}

	if crashDetected {
		return odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonCrashLoopBackOff,
			AgentEnabledMessage: "Pods entered CrashLoopBackOff; instrumentation disabled",
			OtelDistroName:      distroName,
			DistroParams:        distroParameters,
		}
	}

	// check for presence of other agents
	if runtimeDetails.OtherAgent != nil {
		if effectiveConfig.AllowConcurrentAgents == nil || !*effectiveConfig.AllowConcurrentAgents {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonOtherAgentDetected,
				AgentEnabledMessage: fmt.Sprintf("odigos agent not enabled due to other instrumentation agent '%s' detected running in the container", runtimeDetails.OtherAgent.Name),
			}
		} else {

			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        true,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonEnabledSuccessfully,
				AgentEnabledMessage: fmt.Sprintf("we are operating alongside the %s, which is not the recommended configuration. We suggest disabling the %s for optimal performance.", runtimeDetails.OtherAgent.Name, runtimeDetails.OtherAgent.Name),
				OtelDistroName:      distroName,
				DistroParams:        distroParameters,
				Traces:              tracesConfig,
				Metrics:             metricsConfig,
				Logs:                logsConfig,
			}
		}
	}

	return odigosv1.ContainerAgentConfig{
		ContainerName:  containerName,
		AgentEnabled:   true,
		OtelDistroName: distroName,
		DistroParams:   distroParameters,
		Traces:         tracesConfig,
		Metrics:        metricsConfig,
		Logs:           logsConfig,
	}

}

func calculateDefaultDistroPerLanguage(defaultDistros map[common.ProgrammingLanguage]string,
	instrumentationRules *[]odigosv1.InstrumentationRule, dg *distros.Getter) map[common.ProgrammingLanguage]string {

	distrosPerLanguage := make(map[common.ProgrammingLanguage]string, len(defaultDistros))
	for lang, distroName := range defaultDistros {
		distrosPerLanguage[lang] = distroName
	}

	for _, rule := range *instrumentationRules {
		if rule.Spec.OtelDistros == nil {
			continue
		}
		for _, distroName := range rule.Spec.OtelDistros.OtelDistroNames {
			distro := dg.GetDistroByName(distroName)
			if distro == nil {
				continue
			}

			lang := distro.Language
			distrosPerLanguage[lang] = distroName
		}
	}

	return distrosPerLanguage
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
