package utils

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// naive implementation, can be optimized.
// assumption is that the list of workloads is small
func IsWorkloadParticipatingInRule(
	workload k8sconsts.PodWorkload,
	rule *odigosv1alpha1.InstrumentationRule,
	containerName *string,
	language *common.ProgrammingLanguage,
) bool {
	if rule.Spec.Disabled {
		return false
	}
	if rule.Spec.SourcesScopes == nil {
		return true
	}

	// check if we have SourcesScopes in the instrumentation rule - check those first
	if rule.Spec.SourcesScopes != nil {
		scopes := *rule.Spec.SourcesScopes
		if len(scopes) == 0 {
			return false
		}
		name := ""
		if containerName != nil {
			name = *containerName
		}
		lang := common.UnknownProgrammingLanguage
		if language != nil {
			lang = *language
		}
		return scope.AnySourceScopeMatchesContainer(scopes, workload, name, lang)
	}
	return false
}

// Resolves whether the rule applies to the workload for at least one container on the InstrumentationConfig.
func IsInstrumentationConfigParticipatingInRule(
	workload k8sconsts.PodWorkload,
	ic *odigosv1alpha1.InstrumentationConfig,
	rule *odigosv1alpha1.InstrumentationRule,
) bool {
	if rule.Spec.Disabled {
		return false
	}

	// If we don't have overrides, iterate on runtime details by container and extract details from there
	if len(ic.Spec.ContainersOverrides) == 0 {
		// No runtime details yet - try to match the workload only
		if len(ic.Status.RuntimeDetailsByContainer) == 0 {
			return IsWorkloadParticipatingInRule(workload, rule, nil, nil)
		}
		for i := range ic.Status.RuntimeDetailsByContainer {
			rd := &ic.Status.RuntimeDetailsByContainer[i]
			lang := common.UnknownProgrammingLanguage
			if rd.Language != "" {
				lang = rd.Language
			}
			if IsWorkloadParticipatingInRule(workload, rule, &rd.ContainerName, &lang) {
				return true
			}
		}
		return false
	}

	// For overrides, do the same but with the override structs
	detailsByContainer := ic.RuntimeDetailsByContainer()
	for i := range ic.Spec.ContainersOverrides {
		containerName := ic.Spec.ContainersOverrides[i].ContainerName
		runtimeDetails := detailsByContainer[containerName]
		lang := common.UnknownProgrammingLanguage
		if runtimeDetails != nil && runtimeDetails.Language != "" {
			lang = runtimeDetails.Language
		}
		if IsWorkloadParticipatingInRule(workload, rule, &containerName, &lang) {
			return true
		}
	}
	return false
}
