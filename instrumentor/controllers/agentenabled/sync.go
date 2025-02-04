package agentenabled

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
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
	Reason odigosv1.AgentInjectionReason

	// Human-readable message for the condition. it will show up in the ui and tools,
	// and should describe any additional context for the condition in free-form text.
	Message string
}

func reconcileAll(ctx context.Context, c client.Client) (ctrl.Result, error) {

	allInstrumentationConfigs := odigosv1.InstrumentationConfigList{}
	listErr := c.List(ctx, &allInstrumentationConfigs)
	if listErr != nil {
		return ctrl.Result{}, listErr
	}

	var err error
	for _, ic := range allInstrumentationConfigs.Items {
		_, workloadErr := reconcileWorkload(ctx, c, ic.Name, ic.Namespace)
		if workloadErr != nil {
			err = errors.Join(err, workloadErr)
		}
	}

	return ctrl.Result{}, err
}

func reconcileWorkload(ctx context.Context, c client.Client, icName string, namespace string) (ctrl.Result, error) {

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
		// if instrumentation config not found, we have nothing to updated.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling workload for InstrumentationConfig object agent enabling", "name", ic.Name, "namespace", ic.Namespace, "instrumentationConfig", ic)

	condition, err := updateInstrumentationConfigSpec(ctx, c, pw, &ic)
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

	changed := meta.SetStatusCondition(&ic.Status.Conditions, cond)
	if changed {
		err = c.Status().Update(ctx, &ic)
		if err != nil {
			return utils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, nil
}

// this function receives a workload object, and updates the instrumentation config object ptr.
// if the function returns without an error, it means the instrumentation config object was updated.
// caller should persist the object to the API server.
// if the function returns without an error, it also returns an agentInjectedStatusCondition object
// which records what should be written to the status.conditions field of the instrumentation config
// and later be used for viability and monitoring purposes.
func updateInstrumentationConfigSpec(ctx context.Context, c client.Client, pw k8sconsts.PodWorkload, ic *odigosv1.InstrumentationConfig) (*agentInjectedStatusCondition, error) {

	cg, irls, effectiveConfig, err := getRelevantResources(ctx, c, pw)
	if err != nil {
		// error of fetching one of the resources, retry
		return nil, err
	}

	// check if we are waiting for some transient prerequisites to be completed before injecting the agent
	prerequisiteCompleted, reason, message := isReadyForInstrumentation(cg, ic)
	if !prerequisiteCompleted {
		ic.Spec.AgentInjectionEnabled = false
		ic.Spec.Containers = []odigosv1.ContainerConfig{}
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionUnknown,
			Reason:  reason,
			Message: message,
		}, nil
	}

	tier := env.GetOdigosTierFromEnv()
	defaultDistrosPerLanguage := distros.GetDefaultDistroNames(tier)
	distroPerLanguage := applyRulesForDistros(defaultDistrosPerLanguage, irls)

	containersConfig := make([]odigosv1.ContainerConfig, 0, len(ic.Spec.Containers))
	for _, containerRuntimeDetails := range ic.Status.RuntimeDetailsByContainer {
		currentContainerConfig := containerInstrumentationConfig(containerRuntimeDetails.ContainerName, effectiveConfig, containerRuntimeDetails, distroPerLanguage)
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
		if containerConfig.Instrumented {
			instrumentedContainerNames = append(instrumentedContainerNames, containerConfig.ContainerName)
		}
		if odigosv1.AgentInjectionReasonPriority(containerConfig.InstrumentationReason) > odigosv1.AgentInjectionReasonPriority(aggregatedCondition.Reason) {
			// set to the most specific (highest priority) reason from multiple containers.
			aggregatedCondition = containerConfigToStatusCondition(containerConfig)
		}
	}
	if len(instrumentedContainerNames) > 0 {
		// if any instrumented containers are found, the pods webhook should process pods for this workload.
		// set the AgentInjectionEnabled to true to signal that.
		ic.Spec.AgentInjectionEnabled = true
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionTrue,
			Reason:  odigosv1.AgentInjectionReasonInjectedSuccessfully,
			Message: fmt.Sprintf("agent injected successfully to %d containers: %v", len(instrumentedContainerNames), instrumentedContainerNames),
		}, nil
	} else {
		// if none of the containers are instrumented, we can set the status to false
		// to signal to the webhook that those pods should not be processed.
		ic.Spec.AgentInjectionEnabled = false
		return aggregatedCondition, nil
	}
}

func containerConfigToStatusCondition(containerConfig odigosv1.ContainerConfig) *agentInjectedStatusCondition {
	if containerConfig.Instrumented {
		// no expecting to hit this case, but for completeness
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionTrue,
			Reason:  odigosv1.AgentInjectionReasonInjectedSuccessfully,
			Message: fmt.Sprintf("agent injected successfully to container %s", containerConfig.ContainerName),
		}
	} else {
		return &agentInjectedStatusCondition{
			Status:  metav1.ConditionFalse,
			Reason:  containerConfig.InstrumentationReason,
			Message: containerConfig.InstrumentationMessage,
		}
	}
}

func containerInstrumentationConfig(containerName string,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails odigosv1.RuntimeDetailsByContainer,
	distroPerLanguage map[common.ProgrammingLanguage]string) odigosv1.ContainerConfig {

	// check unknown language first. if language is not supported, we can skip the rest of the checks.
	if runtimeDetails.Language == common.UnknownProgrammingLanguage {
		return odigosv1.ContainerConfig{
			ContainerName:         containerName,
			Instrumented:          false,
			InstrumentationReason: odigosv1.AgentInjectionReasonUnsupportedProgrammingLanguage,
		}
	}

	// check if container is ignored by name, assuming IgnoredContainers is a short list.
	for _, ignoredContainer := range effectiveConfig.IgnoredContainers {
		if ignoredContainer == containerName {
			return odigosv1.ContainerConfig{
				ContainerName:         containerName,
				Instrumented:          false,
				InstrumentationReason: odigosv1.AgentInjectionReasonIgnoredContainer,
			}
		}
	}

	// check for deprecated "ignored" language
	// TODO: remove this in odigos v1.1
	if runtimeDetails.Language == common.IgnoredProgrammingLanguage {
		return odigosv1.ContainerConfig{
			ContainerName:         containerName,
			Instrumented:          false,
			InstrumentationReason: odigosv1.AgentInjectionReasonIgnoredContainer,
		}
	}

	// get relevant distroName for the detected language
	distroName, ok := distroPerLanguage[runtimeDetails.Language]
	if !ok {
		return odigosv1.ContainerConfig{
			ContainerName:         containerName,
			Instrumented:          false,
			InstrumentationReason: odigosv1.AgentInjectionReasonNoAvailableAgent,
		}
	}

	distro := distros.GetDistroByName(distroName)
	if distro == nil {
		return odigosv1.ContainerConfig{
			ContainerName:         containerName,
			Instrumented:          false,
			InstrumentationReason: odigosv1.AgentInjectionReasonNoAvailableAgent,
		}
	}

	// check if the runtime version is in supported range if it is provided
	if runtimeDetails.RuntimeVersion != "" && len(distro.RuntimeEnvironments) == 1 {
		constraint, err := version.NewConstraint(distro.RuntimeEnvironments[0].SupportedVersions)
		if err != nil {
			return odigosv1.ContainerConfig{
				ContainerName:          containerName,
				Instrumented:           false,
				InstrumentationReason:  odigosv1.AgentInjectionReasonUnsupportedRuntimeVersion,
				InstrumentationMessage: fmt.Sprintf("failed to parse supported versions constraint: %s", distro.RuntimeEnvironments[0].SupportedVersions),
			}
		}
		detectedVersion, err := version.NewVersion(runtimeDetails.RuntimeVersion)
		if err != nil {
			return odigosv1.ContainerConfig{
				ContainerName:          containerName,
				Instrumented:           false,
				InstrumentationReason:  odigosv1.AgentInjectionReasonUnsupportedRuntimeVersion,
				InstrumentationMessage: fmt.Sprintf("failed to parse runtime version: %s", runtimeDetails.RuntimeVersion),
			}
		}
		if !constraint.Check(detectedVersion) {
			return odigosv1.ContainerConfig{
				ContainerName:          containerName,
				Instrumented:           false,
				InstrumentationReason:  odigosv1.AgentInjectionReasonUnsupportedRuntimeVersion,
				InstrumentationMessage: fmt.Sprintf("%s runtime not supported by OpenTelemetry. supported versions: '%s', found: %s", distro.RuntimeEnvironments[0].Name, constraint, detectedVersion),
			}
		}
	}

	// check for presence of other agents
	if runtimeDetails.OtherAgent != nil {
		if effectiveConfig.AllowConcurrentAgents == nil || !*effectiveConfig.AllowConcurrentAgents {
			return odigosv1.ContainerConfig{
				ContainerName:          containerName,
				Instrumented:           false,
				InstrumentationReason:  odigosv1.AgentInjectionReasonOtherAgentDetected,
				InstrumentationMessage: fmt.Sprintf("odigos disabled due to other instrumentation agent '%s' detected running in the container", runtimeDetails.OtherAgent.Name),
			}
		} else {
			return odigosv1.ContainerConfig{
				ContainerName:          containerName,
				Instrumented:           true,
				InstrumentationReason:  odigosv1.AgentInjectionReasonInjectedSuccessfully,
				InstrumentationMessage: fmt.Sprintf("container is running with other instrumentation agent '%s' and concurrent agents are allowed", runtimeDetails.OtherAgent.Name),
			}
		}
	}

	containerConfig := odigosv1.ContainerConfig{
		ContainerName:  containerName,
		Instrumented:   true,
		OtelDistroName: distroName,
	}

	return containerConfig
}

func applyRulesForDistros(defaultDistros map[common.ProgrammingLanguage]string,
	instrumentationRules *[]odigosv1.InstrumentationRule) map[common.ProgrammingLanguage]string {

	for _, rule := range *instrumentationRules {
		if rule.Spec.OtelSdks == nil {
			continue
		}
		// TODO: change this from otel sdks to distros and use distro name
	}

	return defaultDistros
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
func isReadyForInstrumentation(cg *odigosv1.CollectorsGroup, ic *odigosv1.InstrumentationConfig) (bool, odigosv1.AgentInjectionReason, string) {

	// Check if the node collector is ready
	isNodeCollectorReady, message := isNodeCollectorReady(cg)
	if !isNodeCollectorReady {
		return false, odigosv1.AgentInjectionReasonWaitingForNodeCollector, message
	}

	if len(ic.Status.RuntimeDetailsByContainer) == 0 {
		return false, odigosv1.AgentInjectionReasonWaitingForRuntimeInspection, "waiting for runtime inspection to complete"
	}

	// report success if both prerequisites are completed
	return true, odigosv1.AgentInjectionReasonInjectedSuccessfully, ""
}

func isNodeCollectorReady(cg *odigosv1.CollectorsGroup) (bool, string) {
	if cg == nil {
		return false, "node collector deployment not yet created"
	}

	if !cg.Status.Ready {
		return false, "node collector is not yet ready to receive telemetry from instrumented workloads"
	}

	// node collector is ready to receive telemetry
	return true, ""
}
