package traces

import (
	"net/url"
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

// used to return the result of computing the paths and rule names for kubelet health probes auto-rule.
type kubeletProbePathAndName struct {
	Path        string
	QueryParams []commonapisampling.QueryParamMatcher
	RuleName    string
}

func isK8sHealthProbesSamplingEnabled(effectiveConfig *common.OdigosConfiguration) bool {
	// only add health probe sampling rules when explicitly enabled
	return effectiveConfig.Sampling != nil && effectiveConfig.Sampling.K8sHealthProbesSampling != nil && effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled != nil && *effectiveConfig.Sampling.K8sHealthProbesSampling.Enabled
}

func queryParamsMatch(a, b []commonapisampling.QueryParamMatcher) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name {
			return false
		}

		// Both pointers nil: treat as equal
		if a[i].ValueExact == nil && b[i].ValueExact == nil {
			continue
		}

		// One is nil, other not: not equal
		if (a[i].ValueExact == nil && b[i].ValueExact != nil) || (a[i].ValueExact != nil && b[i].ValueExact == nil) {
			return false
		}

		// Both not nil: compare value
		if *a[i].ValueExact != *b[i].ValueExact {
			return false
		}
	}
	return true
}

// parseHTTPGetPath splits a k8s HTTPGet probe path into path and query param matchers.
// probe paths are relative (e.g. "/healthz" or "/health?type=readiness").
func parseHTTPGetPath(rawPath string) (string, []commonapisampling.QueryParamMatcher) {
	if rawPath == "" {
		return "", nil
	}

	parsed, err := url.Parse(rawPath)
	if err != nil {
		return rawPath, nil
	}

	path := parsed.Path
	if path == "" {
		path = rawPath
	}

	if parsed.RawQuery == "" {
		return path, nil
	}

	queryParams := make([]commonapisampling.QueryParamMatcher, 0, len(parsed.Query()))
	for name, values := range parsed.Query() {
		for _, value := range values {
			queryParams = append(queryParams, commonapisampling.QueryParamMatcher{
				Name:       name,
				ValueExact: &value,
			})
		}
	}

	slices.SortFunc(queryParams, func(a, b commonapisampling.QueryParamMatcher) int {
		if cmp := strings.Compare(a.Name, b.Name); cmp != 0 {
			return cmp
		}
		// Handle nil ValueExact pointers. Assume nil < non-nil.
		switch {
		case a.ValueExact == nil && b.ValueExact == nil:
			return 0
		case a.ValueExact == nil:
			return -1
		case b.ValueExact == nil:
			return 1
		default:
			return strings.Compare(*a.ValueExact, *b.ValueExact)
		}
	})

	return path, queryParams
}

// for kubelet health probes auto-rule, while iterating over the probes,
// add the path and name to the list, and update the rule name if the path and query params already exist.
func addProbePathAndName(pathsAndNames []kubeletProbePathAndName, path string, queryParams []commonapisampling.QueryParamMatcher, name string) []kubeletProbePathAndName {

	// update existing entry if found.
	for i, pathAndName := range pathsAndNames {
		if pathAndName.Path == path && queryParamsMatch(pathAndName.QueryParams, queryParams) {
			pathsAndNames[i].RuleName += "," + name
			return pathsAndNames
		}
	}

	// add new entry if not found.
	pathsAndNames = append(pathsAndNames, kubeletProbePathAndName{
		Path:        path,
		QueryParams: queryParams,
		RuleName:    name,
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
		path, queryParams := parseHTTPGetPath(c.StartupProbe.HTTPGet.Path)
		pathsAndNames = addProbePathAndName(pathsAndNames, path, queryParams, "StartupProbe")
	}
	if c.LivenessProbe != nil && c.LivenessProbe.HTTPGet != nil {
		path, queryParams := parseHTTPGetPath(c.LivenessProbe.HTTPGet.Path)
		pathsAndNames = addProbePathAndName(pathsAndNames, path, queryParams, "LivenessProbe")
	}
	if c.ReadinessProbe != nil && c.ReadinessProbe.HTTPGet != nil {
		path, queryParams := parseHTTPGetPath(c.ReadinessProbe.HTTPGet.Path)
		pathsAndNames = addProbePathAndName(pathsAndNames, path, queryParams, "ReadinessProbe")
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
				Route:       pathAndName.Path,
				Method:      "GET",
				QueryParams: pathAndName.QueryParams,
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

func noisyOperationContainsHttpQueryParams(noisyOperation *commonapisampling.NoisyOperation) bool {
	return noisyOperation != nil && noisyOperation.Operation != nil && noisyOperation.Operation.HttpServer != nil && len(noisyOperation.Operation.HttpServer.QueryParams) > 0
}

func CalculateSamplingCategoryRulesForContainer(samplingRules *[]odigosv1.Sampling, language common.ProgrammingLanguage,
	pw k8sconsts.PodWorkload, containerName string, distro *distro.OtelDistro, workloadObj workload.Workload, effectiveConfig *common.OdigosConfiguration) ([]apisampling.NoisyOperation, []apisampling.HighlyRelevantOperation, []apisampling.CostReductionRule) {

	var filteredNoisyOps []apisampling.NoisyOperation
	var filteredRelevantOps []apisampling.HighlyRelevantOperation
	var filteredCostRules []apisampling.CostReductionRule

	distroSupportsHttpQueryParams := distro.Traces != nil && distro.Traces.HeadSampling != nil && distro.Traces.HeadSampling.HttpQueryParamsSupported

	// compute auto sampling rules
	if isK8sHealthProbesSamplingEnabled(effectiveConfig) {
		kubeletHealthProbesSamplingRules := calculateKubeletHealthProbesSamplingRules(effectiveConfig, workloadObj, containerName)
		for _, rule := range kubeletHealthProbesSamplingRules {
			ruleContainsHttpQueryParams := noisyOperationContainsHttpQueryParams(&rule)
			if ruleContainsHttpQueryParams && !distroSupportsHttpQueryParams {
				// filter out rule which are not supported by the distro.
				// in the future, we should somehow communicate this to the user, and not just silently ignore it.
				// for now, we just silently ignore it.
				continue
			}
			filteredNoisyOps = append(filteredNoisyOps, rule)
		}
	}

	for _, samplingRule := range *samplingRules {
		// Filter and convert NoisyOperations, HighlyRelevantOperations, CostReductionRules.
		// Exclude SourceScopes and Notes from the rules because we want the instrumentationConfig to be more lightweight.

		for _, noisyOp := range samplingRule.Spec.NoisyOperations {
			if scope.SourceScopeMatchesContainer(noisyOp.SourceScopes, pw, language) {

				noisyOperationCommonApi := apisampling.NoisyOperation{
					Id:               odigosv1.ComputeNoisyOperationHash(&noisyOp),
					Name:             noisyOp.Name,
					Disabled:         noisyOp.Disabled,
					Operation:        noisyOp.Operation,
					PercentageAtMost: noisyOp.PercentageAtMost,
				}

				ruleContainsHttpQueryParams := noisyOperationContainsHttpQueryParams(&noisyOperationCommonApi)
				if ruleContainsHttpQueryParams && !distroSupportsHttpQueryParams {
					// filter out rule which are not supported by the distro.
					// in the future, we should somehow communicate this to the user, and not just silently ignore it.
					// for now, we just silently ignore it.
					continue
				}

				filteredNoisyOps = append(filteredNoisyOps, noisyOperationCommonApi)
			}
		}

		// Filter and convert HighlyRelevantOperations - exclude SourceScopes and Notes
		for _, relevantOp := range samplingRule.Spec.HighlyRelevantOperations {
			if scope.SourceScopeMatchesContainer(relevantOp.SourceScopes, pw, language) {
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
			if scope.SourceScopeMatchesContainer(costRule.SourceScopes, pw, language) {
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
