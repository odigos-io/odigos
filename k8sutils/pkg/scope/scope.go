package scope

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

// return true if the source matches and of the specific sources in the scope (OR semantics)
// empty list means "match all".
func sourceMatchScopeSources(source k8sconsts.PodWorkload, scopes *k8sconsts.SourcesScopes) bool {
	if len(scopes.Sources) == 0 {
		return true
	}
	for _, scope := range scopes.Sources {
		if scope == source {
			return true
		}
	}
	return false
}

// return true if the namespace matches and of the specific namespaces in the scope (OR semantics)
// empty list means "match all".
func sourceMatchScopeNamespaces(namespace string, scopes *k8sconsts.SourcesScopes) bool {
	if len(scopes.Namespaces) == 0 {
		return true
	}
	for _, scope := range scopes.Namespaces {
		if scope == namespace {
			return true
		}
	}
	return false
}

// return true if the language matches and of the specific languages in the scope (OR semantics)
// empty list means "match all".
func sourceMatchScopeLanguages(language common.ProgrammingLanguage, scopes *k8sconsts.SourcesScopes) bool {
	if len(scopes.Languages) == 0 {
		return true
	}
	for _, scope := range scopes.Languages {
		if scope == language {
			return true
		}
	}
	return false
}

// SourceScopeMatchesContainer returns true if the scope matches the given workload,
// and language. All non-empty scope fields must match (AND semantics).
// Empty fields in scope act as wildcards.
func SourceScopeMatchesContainer(
	scopes *k8sconsts.SourcesScopes,
	pw k8sconsts.PodWorkload,
	language common.ProgrammingLanguage,
) bool {
	// empty scope means match the entire cluster
	if scopes == nil {
		return true
	}

	// check if the source matches the relevant criterias (AND semantics between criterias)
	// e.g. for scope with namespace "foo" and language "java", the source must match both the namespace and the language

	matchSources := sourceMatchScopeSources(pw, scopes)
	if !matchSources {
		return false
	}

	matchNamespaces := sourceMatchScopeNamespaces(pw.Namespace, scopes)
	if !matchNamespaces {
		return false
	}

	matchLanguages := sourceMatchScopeLanguages(language, scopes)
	if !matchLanguages {
		return false
	}

	// all critireas are empty or matched, so the scope matches
	return true
}
