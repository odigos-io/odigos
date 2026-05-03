package traces

import (
	"slices"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	apisampling "github.com/odigos-io/odigos/common/api/sampling"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	v1 "k8s.io/api/core/v1"
)

func DistroSupportsHeadSampling(distro *distro.OtelDistro) bool {
	return distro.Traces != nil && distro.Traces.HeadSampling != nil && distro.Traces.HeadSampling.Supported
}

// isServiceInRuleScope returns true if the list is empty (match all) or any scope matches the given workload/container/language.
func isServiceInRuleScope(services []scope.SourcesScope, pw k8sconsts.PodWorkload, containerName string, containerLanguage common.ProgrammingLanguage) bool {
	return scope.AnySourceScopeMatchesContainer(services, pw, containerName, containerLanguage)
}

// used to return the result of computing the paths and rule names for kubelet health probes auto-rule.
type kubeletProbePathAndName struct {
	Path     string
	RuleName string
}

func isK8sHealthProbesSamplingEnabled(effectiveConfig *common.OdigosConfiguration) bool {
	// only add health probe sampling rules when explicitly enabled
	return effectiveConfig.Sampling != nil && effectiveConfig.Sampling.K8sHealthProbesSampling != nil && effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled != nil && *effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled
}

// for kubelet health probes auto-rule, while iterating over the probes,
// add the path and name to the list, and update the rule name if the path already exists.
func addProbePathAndName(pathsAndNames []kubeletProbePathAndName, path string, name string) []kubeletProbePathAndName {

	// update existing entry if found.
	for i, pathAndName := range pathsAndNames {
		if pathAndName.Path == path {
			pathsAndNames[i].RuleName += "," + name
			return pathsAndNames
		}
	}

	// add new entry if not found.
	pathsAndNames = append(pathsAndNames, kubeletProbePathAndName{
		Path:     path,
		RuleName: name,
	})

	return pathsAndNames
}

// given a workload object, and a container name,
// calculate the http get path for each health-probe configured.
// returns: map where key is a path, and value is a list of probe names that use
func calculateKubeletHttpGetProbePaths(workloadObj workload.Workload, containerName string) []kubeletProbePathAndName {

	// this list can have at most 3 elements, so no problem iterating over it.
	// avoid using a map since iterating it can range the keys in any order,
	// and we want this config to be idempotent.
	pathsAndNames := []kubeletProbePathAndName{}

	var c *v1.Container
	for _, container := range workloadObj.PodSpec().Containers {
		if container.Name == containerName {
			c = &container
			break
		}
	}

	if c == nil {
		return nil
	}

	if c.StartupProbe != nil && c.StartupProbe.HTTPGet != nil {
		pathsAndNames = addProbePathAndName(pathsAndNames, c.StartupProbe.HTTPGet.Path, "StartupProbe")
	}
	if c.LivenessProbe != nil && c.LivenessProbe.HTTPGet != nil {
		pathsAndNames = addProbePathAndName(pathsAndNames, c.LivenessProbe.HTTPGet.Path, "LivenessProbe")
	}
	if c.ReadinessProbe != nil && c.ReadinessProbe.HTTPGet != nil {
		pathsAndNames = addProbePathAndName(pathsAndNames, c.ReadinessProbe.HTTPGet.Path, "ReadinessProbe")
	}
	return pathsAndNames
}

func getPercentageOrZero(percentage *float64) float64 {
	if percentage != nil {
		return *percentage
	}
	return 0.0
}

func calculateK8sHealthProbeSamplingPercentage(effectiveConfig *common.OdigosConfiguration) float64 {
	if effectiveConfig.Sampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage == nil {
		return 0.0 // default if unset.
	}
	return *effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage
}

func calculateKubeletHealthProbesSamplingRules(effectiveConfig *common.OdigosConfiguration, workloadObj workload.Workload, containerName string) []commonapisampling.NoisyOperation {

	// only add health probe sampling rules when explicitly enabled
	if effectiveConfig.Sampling == nil || effectiveConfig.Sampling.K8sHealthProbesSampling == nil || effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled == nil || !*effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled {
		return nil
	}

	if workloadObj == nil {
		return nil
	}

	kubeletPathAndNames := calculateKubeletHttpGetProbePaths(workloadObj, containerName)
	if len(kubeletPathAndNames) == 0 {
		return nil
	}

	percentageAtMost := calculateK8sHealthProbeSamplingPercentage(effectiveConfig)

	noisyOperations := make([]commonapisampling.NoisyOperation, 0, len(kubeletPathAndNames))
	for _, pathAndName := range kubeletPathAndNames {

		operation := &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
				Route:  pathAndName.Path,
				Method: "GET",
			},
		}

		id := odigosv1.ComputeNoisyOperationHash(&odigosv1.NoisyOperation{
			// avoid setting a scope here, so all of these paths in all containers will have the same rule id.
			Operation: operation,
		})

		noisyOperations = append(noisyOperations, commonapisampling.NoisyOperation{
			Id:               id,
			Name:             "kubelet health probe: " + pathAndName.RuleName,
			Operation:        operation,
			PercentageAtMost: &percentageAtMost,
		})
	}

	return noisyOperations
}

func CalculateSamplingCategoryRulesForContainer(samplingRules *[]odigosv1.Sampling, language common.ProgrammingLanguage,
	pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro, workloadObj workload.Workload, effectiveConfig *common.OdigosConfiguration) ([]apisampling.NoisyOperation, []apisampling.HighlyRelevantOperation, []apisampling.CostReductionRule) {

	var filteredNoisyOps []apisampling.NoisyOperation
	var filteredRelevantOps []apisampling.HighlyRelevantOperation
	var filteredCostRules []apisampling.CostReductionRule

	// compute auto sampling rules
	if isK8sHealthProbesSamplingEnabled(effectiveConfig) {
		filteredNoisyOps = append(filteredNoisyOps, calculateKubeletHealthProbesSamplingRules(effectiveConfig, workloadObj, containerName)...)
	}

	for _, samplingRule := range *samplingRules {
		// Filter and convert NoisyOperations, HighlyRelevantOperations, CostReductionRules.
		// Exclude SourceScopes and Notes from the rules because we want the instrumentationConfig to be more lightweight.

		for _, noisyOp := range samplingRule.Spec.NoisyOperations {
			if isServiceInRuleScope(noisyOp.SourceScopes, pw, containerName, language) {
				filteredNoisyOps = append(filteredNoisyOps, apisampling.NoisyOperation{
					Id:               odigosv1.ComputeNoisyOperationHash(&noisyOp),
					Name:             noisyOp.Name,
					Disabled:         noisyOp.Disabled,
					Operation:        noisyOp.Operation,
					PercentageAtMost: noisyOp.PercentageAtMost,
				})
			}
		}

		// Filter and convert HighlyRelevantOperations - exclude SourceScopes and Notes
		for _, relevantOp := range samplingRule.Spec.HighlyRelevantOperations {
			if isServiceInRuleScope(relevantOp.SourceScopes, pw, containerName, language) {
				filteredRelevantOps = append(filteredRelevantOps, apisampling.HighlyRelevantOperation{
					Id:                odigosv1.ComputeHighlyRelevantOperationHash(&relevantOp),
					Name:              relevantOp.Name,
					Disabled:          relevantOp.Disabled,
					Error:             relevantOp.Error,
					DurationAtLeastMs: relevantOp.DurationAtLeastMs,
					Operation:         relevantOp.Operation,
					PercentageAtLeast: relevantOp.PercentageAtLeast,
				})
			}
		}

		for _, costRule := range samplingRule.Spec.CostReductionRules {
			if isServiceInRuleScope(costRule.SourceScopes, pw, containerName, language) {
				filteredCostRules = append(filteredCostRules, apisampling.CostReductionRule{
					Id:               odigosv1.ComputeCostReductionRuleHash(&costRule),
					Name:             costRule.Name,
					Disabled:         costRule.Disabled,
					Operation:        costRule.Operation,
					PercentageAtMost: costRule.PercentageAtMost,
				})
			}
		}
	}

	// sort the results so the output is deterministic, otherwise, different order of
	// sampling rules (coming from list operations) will result in continuous changes in k8s resource content.
	// lower percentage first, just so it's more organized when looking.
	slices.SortFunc(filteredNoisyOps, func(a, b apisampling.NoisyOperation) int {
		aPercentage := getPercentageOrZero(a.PercentageAtMost)
		bPercentage := getPercentageOrZero(b.PercentageAtMost)
		if aPercentage != bPercentage {
			return int(aPercentage - bPercentage)
		}
		return strings.Compare(a.Id, b.Id)
	})

	slices.SortFunc(filteredRelevantOps, func(a, b apisampling.HighlyRelevantOperation) int {
		aPercentage := getPercentageOrZero(a.PercentageAtLeast)
		bPercentage := getPercentageOrZero(b.PercentageAtLeast)
		if aPercentage != bPercentage {
			return int(aPercentage - bPercentage)
		}
		return strings.Compare(a.Id, b.Id)
	})

	slices.SortFunc(filteredCostRules, func(a, b apisampling.CostReductionRule) int {
		aPercentage := a.PercentageAtMost
		bPercentage := b.PercentageAtMost
		if aPercentage != bPercentage {
			return int(aPercentage - bPercentage)
		}
		return strings.Compare(a.Id, b.Id)
	})

	return filteredNoisyOps, filteredRelevantOps, filteredCostRules
}
