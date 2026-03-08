package scope

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonapi "github.com/odigos-io/odigos/common/api"
	"github.com/odigos-io/odigos/common"
)

// SourcesScopeMatchesContainer returns true if the scope matches the given workload, container,
// and language. All non-empty scope fields must match (AND semantics).
// Empty fields in scope act as wildcards.
func SourcesScopeMatchesContainer(scope commonapi.SourcesScope, pw k8sconsts.PodWorkload, containerName string, language common.ProgrammingLanguage) bool {
	if scope.WorkloadName != "" && scope.WorkloadName != pw.Name {
		return false
	}
	if scope.WorkloadKind != "" && scope.WorkloadKind != string(pw.Kind) {
		return false
	}
	if scope.WorkloadNamespace != "" && scope.WorkloadNamespace != pw.Namespace {
		return false
	}
	if scope.ContainerName != "" && scope.ContainerName != containerName {
		return false
	}
	if scope.WorkloadLanguage != "" && scope.WorkloadLanguage != language {
		return false
	}
	return true
}

// AnySourceScopeMatchesContainer returns true if scopes is empty (match all) or any scope
// matches the given workload, container, and language.
func AnySourceScopeMatchesContainer(scopes []commonapi.SourcesScope, pw k8sconsts.PodWorkload, containerName string, language common.ProgrammingLanguage) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, s := range scopes {
		if SourcesScopeMatchesContainer(s, pw, containerName, language) {
			return true
		}
	}
	return false
}
