package agentenabled

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	distroTypes "github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
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

	for _, ic := range allInstrumentationConfigs.Items {
		res, err := reconcileWorkload(ctx, c, ic.Name, ic.Namespace, dp)
		if err != nil || !res.IsZero() {
			return res, err
		}
	}

	return ctrl.Result{}, nil
}

func reconcileWorkload(ctx context.Context, c client.Client, icName string, namespace string, distroProvider *distros.Provider) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	workloadName, workloadKind, err := workload.ExtractWorkloadInfoFromRuntimeObjectName(icName)
	if err != nil {
		logger.Error(err, "error parsing workload info from runtime object name")
		return ctrl.Result{}, nil // return nil so not to retry
	}
	pw := k8sconsts.PodWorkload{
		Namespace: namespace,
		Kind:      workloadKind,
		Name:      workloadName,
	}

	ic := odigosv1.InstrumentationConfig{}
	err = c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: icName}, &ic)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// instrumentation config is deleted, trigger a rollout for the associated workload
			// this should happen once per workload, as the instrumentation config is deleted
			_, res, err := rollout.Do(ctx, c, nil, pw)
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
	rolloutChanged, res, err := rollout.Do(ctx, c, &ic, pw)

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
	distroPerLanguage := applyRulesForDistros(defaultDistrosPerLanguage, irls, distroProvider.Getter)

	containersConfig := make([]odigosv1.ContainerAgentConfig, 0, len(ic.Spec.Containers))
	for _, containerRuntimeDetails := range ic.Status.RuntimeDetailsByContainer {
		currentContainerConfig := containerInstrumentationConfig(containerRuntimeDetails.ContainerName, effectiveConfig, containerRuntimeDetails, distroPerLanguage, distroProvider.Getter)
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
		ic.Spec.AgentInjectionEnabled = true
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

func containerInstrumentationConfig(containerName string,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails odigosv1.RuntimeDetailsByContainer,
	distroPerLanguage map[common.ProgrammingLanguage]string,
	distroGetter *distros.Getter) odigosv1.ContainerAgentConfig {

	// check unknown language first. if language is not supported, we can skip the rest of the checks.
	if runtimeDetails.Language == common.UnknownProgrammingLanguage {
		return odigosv1.ContainerAgentConfig{
			ContainerName:      containerName,
			AgentEnabled:       false,
			AgentEnabledReason: odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
		}
	}

	// check if container is ignored by name, assuming IgnoredContainers is a short list.
	for _, ignoredContainer := range effectiveConfig.IgnoredContainers {
		if ignoredContainer == containerName {
			return odigosv1.ContainerAgentConfig{
				ContainerName:      containerName,
				AgentEnabled:       false,
				AgentEnabledReason: odigosv1.AgentEnabledReasonIgnoredContainer,
			}
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
	} else if runtimeDetails.RuntimeVersion == "" {
		// If the runtime does not have a version, we can't replace placeholders
		for _, staticVariable := range distro.EnvironmentVariables.StaticVariables {
			// This is a placeholder for the runtime version, disable the agent
			if strings.Contains(staticVariable.EnvValue, distroTypes.RuntimeVersionPlaceholderMajorMinor) {
				return odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
					AgentEnabledMessage: "runtime version is not available, but the distribution requires it to be set",
				}
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

		default:
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonMissingDistroParameter,
				AgentEnabledMessage: fmt.Sprintf("unsupported parameter '%s' for distro '%s'", parameterName, distroName),
			}
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
			}
		}
	}

	containerConfig := odigosv1.ContainerAgentConfig{
		ContainerName:  containerName,
		AgentEnabled:   true,
		OtelDistroName: distroName,
		DistroParams:   distroParameters,
	}

	return containerConfig
}

func applyRulesForDistros(defaultDistros map[common.ProgrammingLanguage]string,
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

	if len(ic.Status.RuntimeDetailsByContainer) == 0 {
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
