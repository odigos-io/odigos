package scope

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

// SourcesScope is defined in api/k8sconsts to avoid circular module dependencies
// (k8sutils imports api, so api cannot import k8sutils).
// This alias keeps all existing callers working without changes.
type SourcesScope = k8sconsts.SourcesScope

// AnySourceScopeMatchesContainer returns true if the scope matches the given workload, container,
// and language. All non-empty scope fields must match (AND semantics).
// Empty fields in scope act as wildcards.
func AnySourceScopeMatchesContainer(
	scopes []SourcesScope,
	pw k8sconsts.PodWorkload,
	containerName string,
	language common.ProgrammingLanguage,
) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		if scope.WorkloadName != "" && scope.WorkloadName != pw.Name {
			continue
		}
		if scope.WorkloadKind != "" && scope.WorkloadKind != string(pw.Kind) {
			continue
		}
		if scope.WorkloadNamespace != "" && scope.WorkloadNamespace != pw.Namespace {
			continue
		}
		if scope.ContainerName != "" && scope.ContainerName != containerName {
			continue
		}
		if scope.WorkloadLanguage != "" && scope.WorkloadLanguage != language {
			continue
		}
		return true
	}
	return false
}
