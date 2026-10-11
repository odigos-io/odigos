package scope_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/scope"
	"github.com/stretchr/testify/assert"
)

func testPodWorkload() k8sconsts.PodWorkload {
	return k8sconsts.PodWorkload{
		Name:      "svc",
		Namespace: "default",
		Kind:      k8sconsts.WorkloadKindDeployment,
	}
}

func Test_SourceScopeMatchesContainer_NilScope_MatchesEverything(t *testing.T) {
	// Arrange: nil scope acts as a cluster-wide wildcard
	pw := testPodWorkload()

	// Act
	got := scope.SourceScopeMatchesContainer(nil, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got, "nil scope must match every container")
}

func Test_SourceScopeMatchesContainer_EmptyScope_MatchesEverything(t *testing.T) {
	// Arrange: an explicit empty scope (no criteria) is also a wildcard
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.JavaProgrammingLanguage)

	// Assert
	assert.True(t, got, "empty scope criteria must match every container")
}

func Test_SourceScopeMatchesContainer_SourceMatch(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{pw},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_SourceMismatch_DifferentName(t *testing.T) {
	// Arrange: same namespace and kind, different name
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{{
			Name:      "other",
			Namespace: pw.Namespace,
			Kind:      pw.Kind,
		}},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got, "non-empty Sources requires an exact PodWorkload match")
}

func Test_SourceScopeMatchesContainer_SourceMismatch_DifferentKind(t *testing.T) {
	// Arrange: same name and namespace, different kind
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{{
			Name:      pw.Name,
			Namespace: pw.Namespace,
			Kind:      k8sconsts.WorkloadKindStatefulSet,
		}},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got)
}

func Test_SourceScopeMatchesContainer_SourceMatch_AnyOfMany(t *testing.T) {
	// Arrange: source list has multiple entries; one matches
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources: []k8sconsts.PodWorkload{
			{Name: "other", Namespace: pw.Namespace, Kind: pw.Kind},
			pw,
			{Name: "another", Namespace: pw.Namespace, Kind: pw.Kind},
		},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got, "Sources match is OR semantics across the list")
}

func Test_SourceScopeMatchesContainer_NamespaceMatch(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Namespaces: []string{pw.Namespace},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_NamespaceMismatch(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Namespaces: []string{"other-ns"},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got)
}

func Test_SourceScopeMatchesContainer_NamespaceMatch_AnyOfMany(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Namespaces: []string{"other-ns", pw.Namespace, "yet-another"},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_LanguageMatch(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Languages: []common.ProgrammingLanguage{common.GoProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_LanguageMismatch(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got)
}

func Test_SourceScopeMatchesContainer_LanguageMatch_AnyOfMany(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Languages: []common.ProgrammingLanguage{
			common.JavaProgrammingLanguage,
			common.PythonProgrammingLanguage,
			common.GoProgrammingLanguage,
		},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_AllCriteriaMatch(t *testing.T) {
	// Arrange: every criterion is populated and matches
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources:    []k8sconsts.PodWorkload{pw},
		Namespaces: []string{pw.Namespace},
		Languages:  []common.ProgrammingLanguage{common.GoProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}

func Test_SourceScopeMatchesContainer_AndSemantics_SourceMatch_NamespaceMiss(t *testing.T) {
	// Arrange: source matches but namespace list excludes the workload's namespace
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources:    []k8sconsts.PodWorkload{pw},
		Namespaces: []string{"other-ns"},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got, "all non-empty criteria must match (AND semantics)")
}

func Test_SourceScopeMatchesContainer_AndSemantics_SourceMatch_LanguageMiss(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Sources:   []k8sconsts.PodWorkload{pw},
		Languages: []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got)
}

func Test_SourceScopeMatchesContainer_AndSemantics_NamespaceMatch_LanguageMiss(t *testing.T) {
	// Arrange
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Namespaces: []string{pw.Namespace},
		Languages:  []common.ProgrammingLanguage{common.JavaProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.False(t, got)
}

func Test_SourceScopeMatchesContainer_NamespaceWildcard_LanguageMatch(t *testing.T) {
	// Arrange: empty namespace list acts as wildcard, language criterion still applies
	pw := testPodWorkload()
	scopes := &k8sconsts.SourcesScopes{
		Languages: []common.ProgrammingLanguage{common.GoProgrammingLanguage},
	}

	// Act
	got := scope.SourceScopeMatchesContainer(scopes, pw, common.GoProgrammingLanguage)

	// Assert
	assert.True(t, got)
}
