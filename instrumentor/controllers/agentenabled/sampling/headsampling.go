package sampling

import (
	"slices"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	v1 "k8s.io/api/core/v1"
)

// used to return the result of computing the paths and rule names for kubelet health probes auto-rule.
type kubeletProbePathAndName struct {
	Path     string
	RuleName string
}

func getPercentageOrZero(percentage *float64) float64 {
	if percentage != nil {
		return *percentage
	}
	return 0.0
}

func calculateHeadSamplingPercentage(effectiveConfig *common.OdigosConfiguration) float64 {
	if effectiveConfig.Sampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling == nil {
		return 0.0 // default if unset.
	} else if effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage == nil {
		return 0.0 // default if unset.
	}
	return *effectiveConfig.Sampling.K8sHealthProbesSampling.KeepPercentage
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

	percentageAtMost := calculateHeadSamplingPercentage(effectiveConfig)

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

// givin a specific container in a workload, matched to a distro, calculate it's head sampling based on odigos config and sampling rules.
func CalculateHeadSamplingConfig(distro *distro.OtelDistro, workloadObj workload.Workload, containerName string, effectiveConfig *common.OdigosConfiguration, samplingRules *[]odigosv1.Sampling, pw k8sconsts.PodWorkload) *odigosv1.HeadSamplingConfig {

	// only calculate head sampling config if the distro supports it
	if distro.Traces == nil || distro.Traces.HeadSampling == nil || !distro.Traces.HeadSampling.Supported {
		return nil
	}

	kubeletHealthProbesRules := calculateKubeletHealthProbesSamplingRules(effectiveConfig, workloadObj, containerName)
	customSamplingRules := getRelevantNoisyOperations(samplingRules, pw, containerName, distro)

	// if no rules are found, disable the head sampling (unused)
	if len(customSamplingRules) == 0 && len(kubeletHealthProbesRules) == 0 {
		return nil
	}

	noisyOperations := append(kubeletHealthProbesRules, customSamplingRules...)

	// sort them so the output is deterministic, otherwise, different order of
	// sampling rules (coming from list operations) will result in continuous changes in resource.
	// lower percentage first, just so it's more organized when looking.
	slices.SortFunc(noisyOperations, func(a, b commonapisampling.NoisyOperation) int {
		aPercentage := getPercentageOrZero(a.PercentageAtMost)
		bPercentage := getPercentageOrZero(b.PercentageAtMost)
		if aPercentage != bPercentage {
			return int(aPercentage - bPercentage)
		}
		return strings.Compare(a.Id, b.Id)
	})

	return &odigosv1.HeadSamplingConfig{
		NoisyOperations: noisyOperations,
	}
}

func getRelevantNoisyOperations(samplingRules *[]odigosv1.Sampling, pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro) []commonapisampling.NoisyOperation {
	noisyOperations := []commonapisampling.NoisyOperation{}

	for _, samplingRule := range *samplingRules {

		if samplingRule.Spec.Disabled {
			continue
		}

		for _, noisyOperation := range samplingRule.Spec.NoisyOperations {

			// only take into account operations that are relevant to the current source and container
			if IsServiceInRuleScope(noisyOperation.SourceScopes, pw, containerName, distro.Language) {
				id := odigosv1.ComputeNoisyOperationHash(&noisyOperation)
				noisyOperations = append(noisyOperations, commonapisampling.NoisyOperation{
					Id:               id,
					Name:             noisyOperation.Name,
					Disabled:         noisyOperation.Disabled,
					Operation:        noisyOperation.Operation,
					PercentageAtMost: noisyOperation.PercentageAtMost,
				})
			}
		}
	}
	return noisyOperations
}
