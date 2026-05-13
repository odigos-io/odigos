package scope

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

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
