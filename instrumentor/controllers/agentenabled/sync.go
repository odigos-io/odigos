package agentenabled

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	actionsv1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
	"github.com/odigos-io/odigos/common"
	commonconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/rollout"
	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled/signalconfig"
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
			_, res, err := rollout.Do(ctx, c, nil, pw, conf, distroProvider)
			return res, err
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
	rolloutChanged, res, err := rollout.Do(ctx, c, &ic, pw, conf, distroProvider)

	if rolloutChanged || agentEnabledChanged {
		updateErr := c.Status().Update(ctx, &ic)
		if updateErr != nil {
			// if the update fails, we should not return an error, but rather log it and retry later.
			return utils.K8SUpdateErrorHandler(updateErr)
		}
	}

	return res, err
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
	cg, irls, agentLevelActions, workloadObj, err := getRelevantResources(ctx, c, pw)
	if err != nil {
		// error of fetching one of the resources, retry
		return nil, err
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
	distroPerLanguage := calculateDefaultDistroPerLanguage(defaultDistrosPerLanguage, irls, distroProvider.Getter)

	// If the source was already marked for instrumentation, but has caused a CrashLoopBackOff or ImagePullBackOff we'd like to stop
	// instrumentating it and to disable future instrumentation of this service
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
	containersConfig := make([]odigosv1.ContainerAgentConfig, 0, len(ic.Spec.Containers))
	runtimeDetailsByContainer := ic.RuntimeDetailsByContainer()
	podManifestInjectionOptional := true

	for containerName, containerRuntimeDetails := range runtimeDetailsByContainer {
		// at this point, containerRuntimeDetails can be nil, indicating we have no runtime details for this container
		// from automatic runtime detection or overrides.
		containerOverride := ic.GetOverridesForContainer(containerName)
		currentContainerConfig := calculateContainerInstrumentationConfig(containerName, effectiveConfig, containerRuntimeDetails, distroPerLanguage, distroProvider.Getter, rollbackOccurred, existingBackoffReason, cg, irls, containerOverride, agentLevelActions, workloadObj, pw)
		containersConfig = append(containersConfig, currentContainerConfig)
		// if at least one container has agent enabled, and pod manifest injection is required,
		// then the overall pod manifest injection is required.
		if currentContainerConfig.AgentEnabled && !currentContainerConfig.PodManifestInjectionOptional {
			podManifestInjectionOptional = false
		}
	}
	ic.Spec.Containers = containersConfig
	ic.Spec.PodManifestInjectionOptional = podManifestInjectionOptional

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
		updateInstrumentationConfigAgentsMetaHash(ic, "")
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
// returns "nil" on success, and a container agent config with reason and message if not supported.
func isLoaderInjectionSupportedByRuntimeDetails(containerName string, runtimeDetails *odigosv1.RuntimeDetailsByContainer) *odigosv1.ContainerAgentConfig {
	// check for conditions to inject ldpreload when it is the only method configured.
	secureExecution := runtimeDetails.SecureExecutionMode == nil || *runtimeDetails.SecureExecutionMode
	if secureExecution {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: "container is running in secure execution mode and injection method is set to 'loader'",
		}
	}

	odigosLoaderPath := filepath.Join(k8sconsts.OdigosAgentsDirectory, commonconsts.OdigosLoaderDirName, commonconsts.OdigosLoaderName)
	ldPreloadVal, ldPreloadFoundInInspection := getEnvVarFromList(runtimeDetails.EnvVars, commonconsts.LdPreloadEnvVarName)
	ldPreloadUnsetOrExpected := !ldPreloadFoundInInspection || strings.Contains(ldPreloadVal, odigosLoaderPath)
	if !ldPreloadUnsetOrExpected {
		return &odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
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
	containerName string,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	distro *distro.OtelDistro,
) (*common.EnvInjectionDecision, *odigosv1.ContainerAgentConfig) {
	if effectiveConfig.AgentEnvVarsInjectionMethod == nil {
		// this should never happen, as the config is reconciled with default value.
		// it is only here as a safety net.
		return nil, &odigosv1.ContainerAgentConfig{
			ContainerName:       containerName,
			AgentEnabled:        false,
			AgentEnabledReason:  odigosv1.AgentEnabledReasonInjectionConflict,
			AgentEnabledMessage: "no injection method configured for odigos agent",
		}
	}

	// If we should try loader, check for this first
	distroSupportsLoader := distro.RuntimeAgent != nil && distro.RuntimeAgent.LdPreloadInjectionSupported
	loaderRequested := (*effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod || *effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderFallbackToPodManifestInjectionMethod)

	if distroSupportsLoader && loaderRequested {
		err := isLoaderInjectionSupportedByRuntimeDetails(containerName, runtimeDetails)
		if err != nil {
			// loader is requested by config and distro, but not supported by the runtime details.
			if *effectiveConfig.AgentEnvVarsInjectionMethod == common.LoaderEnvInjectionMethod {
				// config requires us to use loader when it is supported by distro,
				// thus we can't use it and need fail the injection.
				return nil, err
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

// filterUrlTemplateRulesForContainer filters template rules to only include those relevant to the container.
// A rule group is applied if ALL set filters match (AND logic).
// If no filters are set in a group, it's considered global and applies to all containers.
func filterUrlTemplateRulesForContainer(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) *odigosv1.UrlTemplatizationConfig {
	var rules []string
	participating := false

	for _, action := range *agentLevelActions {
		// Safety check: actions were already filtered to only include template actions.
		if action.Spec.URLTemplatization == nil {
			continue
		}

		for _, rulesGroup := range action.Spec.URLTemplatization.TemplatizationRulesGroups {
			if templatizationRulesGroupMatchesContainer(rulesGroup, language, pw) {
				participating = true
				for _, rule := range rulesGroup.TemplatizationRules {
					rules = append(rules, rule.Template)
				}
			}
		}
	}

	// container can participate in templatization and have no rule.
	// if at least one rule group matches, the container participates.
	if !participating {
		return nil
	}

	return &odigosv1.UrlTemplatizationConfig{
		Rules: rules,
	}
}

func filterIgnoreHealthChecksForContainer(agentLevelActions *[]odigosv1.Action, language common.ProgrammingLanguage) []actionsv1.IgnoreHealthChecksConfig {
	ignoredHealthChecksConfigs := []actionsv1.IgnoreHealthChecksConfig{}
	for _, ignoreHealthCheck := range *agentLevelActions {
		if ignoreHealthCheck.Spec.Samplers != nil && ignoreHealthCheck.Spec.Samplers.IgnoreHealthChecks != nil {
			ignoredHealthChecksConfigs = append(ignoredHealthChecksConfigs, *ignoreHealthCheck.Spec.Samplers.IgnoreHealthChecks)
		}
	}
	return ignoredHealthChecksConfigs
}

// templatizationRulesGroupMatchesContainer checks if a rules group matches the container based on all set filters.
// Returns true if all explicitly-set filters match (AND logic), or if no filters are set (global rule).
func templatizationRulesGroupMatchesContainer(rulesGroup actions.UrlTemplatizationRulesGroup, language common.ProgrammingLanguage, pw k8sconsts.PodWorkload) bool {
	// Filter by programming language
	if rulesGroup.FilterProgrammingLanguage != nil {
		if *rulesGroup.FilterProgrammingLanguage != language {
			return false
		}
	}

	// Filter by k8s namespace
	if rulesGroup.FilterK8sNamespace != "" {
		if rulesGroup.FilterK8sNamespace != pw.Namespace {
			return false
		}
	}

	// Filter by k8s workload kind
	if rulesGroup.FilterK8sWorkloadKind != nil {
		if *rulesGroup.FilterK8sWorkloadKind != pw.Kind {
			return false
		}
	}

	// Filter by k8s workload name
	if rulesGroup.FilterK8sWorkloadName != "" {
		if rulesGroup.FilterK8sWorkloadName != pw.Name {
			return false
		}
	}

	return true
}

func calculateContainerInstrumentationConfig(containerName string,
	effectiveConfig *common.OdigosConfiguration,
	runtimeDetails *odigosv1.RuntimeDetailsByContainer,
	distroPerLanguage map[common.ProgrammingLanguage]string,
	distroGetter *distros.Getter,
	rollbackOccurred bool,
	existingBackoffReason odigosv1.AgentEnabledReason,
	nodeCollectorsGroup *odigosv1.CollectorsGroup,
	irls *[]odigosv1.InstrumentationRule,
	containerOverride *odigosv1.ContainerOverride,
	agentLevelActions *[]odigosv1.Action,
	workloadObj workload.Workload,
	pw k8sconsts.PodWorkload,
) odigosv1.ContainerAgentConfig {
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

	filteredTemplateRules := filterUrlTemplateRulesForContainer(agentLevelActions, runtimeDetails.Language, pw)
	ignoreHealthChecks := filterIgnoreHealthChecksForContainer(agentLevelActions, runtimeDetails.Language)

	d, err := resolveContainerDistro(containerName, containerOverride, runtimeDetails.Language, distroPerLanguage, distroGetter)
	if err != nil {
		return *err
	}
	distroName := d.Name

	tracesEnabled, metricsEnabled, logsEnabled := signalconfig.GetEnabledSignalsForContainer(nodeCollectorsGroup, irls)

	// at this time, we don't populate the signals specific configs, but we will do it soon
	tracesConfig, err := signalconfig.CalculateTracesConfig(tracesEnabled, effectiveConfig, containerName, runtimeDetails.Language, filteredTemplateRules, ignoreHealthChecks, irls, agentLevelActions, workloadObj, d)
	if err != nil {
		return *err
	}
	metricsConfig, err := signalconfig.CalculateMetricsConfig(metricsEnabled, effectiveConfig, d, containerName)
	if err != nil {
		return *err
	}
	logsConfig, err := signalconfig.CalculateLogsConfig(logsEnabled, effectiveConfig, containerName)
	if err != nil {
		return *err
	}

	envInjectionDecision, unsupportedDetails := getEnvInjectionDecision(containerName, effectiveConfig, runtimeDetails, d)
	if unsupportedDetails != nil {
		// if we have a container agent config with reason and message, we return it.
		// this is a failure to inject the agent, and we should not proceed with other checks.
		return *unsupportedDetails
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

	// check if the runtime version is in supported range if it is provided
	if runtimeDetails.RuntimeVersion != "" && len(d.RuntimeEnvironments) == 1 {
		constraint, err := version.NewConstraint(d.RuntimeEnvironments[0].SupportedVersions)
		if err != nil {
			return odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedRuntimeVersion,
				AgentEnabledMessage: fmt.Sprintf("failed to parse supported versions constraint: %s", d.RuntimeEnvironments[0].SupportedVersions),
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
				AgentEnabledMessage: fmt.Sprintf("%s runtime not supported by OpenTelemetry. supported versions: '%s', found: %s", d.RuntimeEnvironments[0].Name, constraint, detectedVersion),
			}
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

	podManifestInjectionRequired := distro.IsRestartRequired(d, effectiveConfig)

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
				ContainerName:                containerName,
				AgentEnabled:                 true,
				PodManifestInjectionOptional: !podManifestInjectionRequired,
				AgentEnabledReason:           odigosv1.AgentEnabledReasonEnabledSuccessfully,
				AgentEnabledMessage:          fmt.Sprintf("we are operating alongside the %s, which is not the recommended configuration. We suggest disabling the %s for optimal performance.", runtimeDetails.OtherAgent.Name, runtimeDetails.OtherAgent.Name),
				OtelDistroName:               distroName,
				DistroParams:                 distroParameters,
				EnvInjectionMethod:           envInjectionDecision,
				Traces:                       tracesConfig,
				Metrics:                      metricsConfig,
				Logs:                         logsConfig,
			}
		}
	}

	return odigosv1.ContainerAgentConfig{
		ContainerName:                containerName,
		AgentEnabled:                 true,
		PodManifestInjectionOptional: !podManifestInjectionRequired,
		OtelDistroName:               distroName,
		DistroParams:                 distroParameters,
		EnvInjectionMethod:           envInjectionDecision,
		Traces:                       tracesConfig,
		Metrics:                      metricsConfig,
		Logs:                         logsConfig,
	}
}

func calculateDefaultDistroPerLanguage(defaultDistros map[common.ProgrammingLanguage]string,
	instrumentationRules *[]odigosv1.InstrumentationRule, dg *distros.Getter,
) map[common.ProgrammingLanguage]string {
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

// givin the container relevant resources, resolve the otel distro to use for the container.
// the function will:
//  1. check for distro in container override, validate and return it if found.
//  2. check the default distros for the language and return the distro to use if found.
//  3. if the distro cannot be resolved for any reason, the function will return an error as
//     ContainerAgentConfig with the appropriate reason and message for the failure.
func resolveContainerDistro(
	containerName string,
	containerOverride *odigosv1.ContainerOverride,
	containerLanguage common.ProgrammingLanguage,
	distroPerLanguage map[common.ProgrammingLanguage]string,
	distroGetter *distros.Getter,
) (*distro.OtelDistro, *odigosv1.ContainerAgentConfig) {
	// check if the distro name is specifically overridden.
	// this can happen for languages that support multiple distros,
	// and the user want to specify a specific distro for this specific workload, and not the default one.
	if containerOverride != nil && containerOverride.OtelDistroName != nil {

		overwriteDistroName := *containerOverride.OtelDistroName
		distro := distroGetter.GetDistroByName(overwriteDistroName)
		if distro == nil { // not expected to happen, here for safety net
			message := fmt.Sprintf("requested otel distro %s is not available in this odigos tier", overwriteDistroName)
			return nil, &odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
				AgentEnabledMessage: message,
			}
		}

		// verify the distro matches the language, since it might be overridden by the container override.
		if distro.Language != containerLanguage {
			return nil, &odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonUnsupportedProgrammingLanguage,
				AgentEnabledMessage: fmt.Sprintf("requested otel distro %s does not support language %s", overwriteDistroName, containerLanguage),
			}
		}

		return distro, nil

	} else { // use the default distro for the language

		distroName, ok := distroPerLanguage[containerLanguage]
		if !ok {
			if containerLanguage == common.UnknownProgrammingLanguage {
				return nil, &odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
					AgentEnabledMessage: "runtime language/platform cannot be detected, no instrumentation agent is available. use the container override to manually specify the programming language.",
				}
			} else {
				return nil, &odigosv1.ContainerAgentConfig{
					ContainerName:       containerName,
					AgentEnabled:        false,
					AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
					AgentEnabledMessage: fmt.Sprintf("support for %s is coming soon. no instrumentation agent available at the moment", containerLanguage),
				}
			}
		}

		distro := distroGetter.GetDistroByName(distroName)
		if distro == nil { // not expected to happen, here for safety net
			message := fmt.Sprintf("otel distro %s is not available in this odigos tier", distroName)
			return nil, &odigosv1.ContainerAgentConfig{
				ContainerName:       containerName,
				AgentEnabled:        false,
				AgentEnabledReason:  odigosv1.AgentEnabledReasonNoAvailableAgent,
				AgentEnabledMessage: message,
			}
		}

		return distro, nil
	}
}
