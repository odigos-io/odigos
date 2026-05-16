package utils

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// Resolves whether the rule applies to the workload for at least one container on the InstrumentationConfig.
func IsInstrumentationConfigParticipatingInRule(
	workload k8sconsts.PodWorkload,
	ic *odigosv1alpha1.InstrumentationConfig,
	rule *odigosv1alpha1.InstrumentationRule,
) bool {
	if rule.Spec.Disabled {
		return false
	}

	if len(ic.Spec.ContainersOverrides) == 0 {
		if len(ic.Status.RuntimeDetailsByContainer) == 0 {
			return scope.SourceScopeMatchesContainer(rule.Spec.SourcesScopes, workload, common.UnknownProgrammingLanguage)
		}
		for i := range ic.Status.RuntimeDetailsByContainer {
			if scope.SourceScopeMatchesContainer(rule.Spec.SourcesScopes, workload, ic.Status.RuntimeDetailsByContainer[i].Language) {
				return true
			}
		}
		return false
	}

	detailsByContainer := ic.RuntimeDetailsByContainer()
	for i := range ic.Spec.ContainersOverrides {
		containerName := ic.Spec.ContainersOverrides[i].ContainerName
		language := common.UnknownProgrammingLanguage
		if runtimeDetails := detailsByContainer[containerName]; runtimeDetails != nil && runtimeDetails.Language != "" {
			language = runtimeDetails.Language
		}
		if scope.SourceScopeMatchesContainer(rule.Spec.SourcesScopes, workload, language) {
			return true
		}
	}
	return false
}
