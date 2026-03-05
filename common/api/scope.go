package api

import (
	"github.com/odigos-io/odigos/common"
)

// SourcesScopeMatchesWorkload returns true if the scope matches the workload.
// Only workload identity is checked: WorkloadNamespace, WorkloadKind, WorkloadName.
// Empty fields in scope act as wildcards (match any value).
func SourcesScopeMatchesWorkload(scope SourcesScope, pw WorkloadRef) bool {
	if scope.WorkloadNamespace != "" && scope.WorkloadNamespace != pw.Namespace {
		return false
	}
	if scope.WorkloadKind != "" && scope.WorkloadKind != pw.Kind {
		return false
	}
	if scope.WorkloadName != "" && scope.WorkloadName != pw.Name {
		return false
	}
	return true
}

// SourcesScopeMatchesContainer returns true if the scope matches the given
// workload, container, and language. All non-empty scope fields must match (AND semantics).
// Empty fields in scope act as wildcards.
func SourcesScopeMatchesContainer(scope SourcesScope, pw WorkloadRef, containerName string, language common.ProgrammingLanguage) bool {
	if scope.WorkloadName != "" && scope.WorkloadName != pw.Name {
		return false
	}
	if scope.WorkloadKind != "" && scope.WorkloadKind != pw.Kind {
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

// AnySourceScopeMatchesContainer returns true if the list is empty (match all)
// or any scope matches the given workload/container/language.
func AnySourceScopeMatchesContainer(scopes []SourcesScope, pw WorkloadRef, containerName string, language common.ProgrammingLanguage) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		if SourcesScopeMatchesContainer(scope, pw, containerName, language) {
			return true
		}
	}
	return false
}

// AnySourceScopeMatchesWorkload returns true if the list is empty (match all)
// or any scope matches the given workload.
func AnySourceScopeMatchesWorkload(scopes []SourcesScope, pw WorkloadRef) bool {
	if len(scopes) == 0 {
		return true
	}
	for _, scope := range scopes {
		if SourcesScopeMatchesWorkload(scope, pw) {
			return true
		}
	}
	return false
}
