package utils

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
)

// IsContainerParticipatingInRule reports whether the rule applies to the given container of a
// workload. A rule applies when it is enabled and its sources scope matches the workload +
// container + language. A nil SourcesScopes pointer on the rule is treated as "match all";
// an empty (non-nil) slice is also "match all" (see AnySourceScopeMatchesContainer).
// Callers pass concrete values: "" for containerName and common.UnknownProgrammingLanguage for
// language when the container's identity is not yet known. These defaults are matched
// literally, so scopes pinning ContainerName or WorkloadLanguage will not match unresolved
// containers.
func IsContainerParticipatingInRule(
	workload k8sconsts.PodWorkload,
	rule *odigosv1alpha1.InstrumentationRule,
	containerName string,
	language common.ProgrammingLanguage,
) bool {
	if rule.Spec.Disabled {
		return false
	}
	var scopes []scope.SourcesScope
	if rule.Spec.SourcesScopes != nil {
		scopes = *rule.Spec.SourcesScopes
	}
	return scope.AnySourceScopeMatchesContainer(scopes, workload, containerName, language)
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

	if len(ic.Spec.ContainersOverrides) == 0 {
		// No runtime details yet - match only on workload-level scope fields by passing the
		// "unknown" defaults for container name and language.
		if len(ic.Status.RuntimeDetailsByContainer) == 0 {
			return IsContainerParticipatingInRule(workload, rule, "", common.UnknownProgrammingLanguage)
		}
		for i := range ic.Status.RuntimeDetailsByContainer {
			rd := &ic.Status.RuntimeDetailsByContainer[i]
			if IsContainerParticipatingInRule(workload, rule, rd.ContainerName, rd.Language) {
				return true
			}
		}
		return false
	}

	detailsByContainer := ic.RuntimeDetailsByContainer()
	for i := range ic.Spec.ContainersOverrides {
		containerName := ic.Spec.ContainersOverrides[i].ContainerName
		language := common.UnknownProgrammingLanguage
		if runtimeDetails := detailsByContainer[containerName]; runtimeDetails != nil {
			language = runtimeDetails.Language
		}
		if IsContainerParticipatingInRule(workload, rule, containerName, language) {
			return true
		}
	}
	return false
}
