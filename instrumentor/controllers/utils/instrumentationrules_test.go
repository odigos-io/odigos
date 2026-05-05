package utils_test

// ****************
// IsWorkloadParticipatingInRule tests
// ****************

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	"github.com/odigos-io/odigos/instrumentor/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func Test_IsWorkloadParticipatingInRule_BothFields_ScopeNoMatch_IgnoresWorkloads(t *testing.T) {
	// Arrange: workload listed in deprecated workloads only; sourcesScopes does not match
	ns := testutil.NewMockNamespace()
	svcDep := testutil.NewMockTestDeployment(ns, "svc")
	otherDep := testutil.NewMockTestDeployment(ns, "other")
	pw := testutil.PodWorkloadFromDeployment(svcDep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScopeAndWorkloads(
		"ir", ns.Name,
		[]k8sconsts.SourcesScope{{
			WorkloadName:      otherDep.Name,
			WorkloadNamespace: otherDep.Namespace,
			WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		}},
		[]k8sconsts.PodWorkload{pw},
	)

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.False(t, isParticipatingInRule, "sourcesScopes wins; deprecated workloads must be ignored")
}

func Test_IsWorkloadParticipatingInRule_BothFields_ScopeMatch_IgnoresWorkloads(t *testing.T) {
	// Arrange: sourcesScopes matches actual workload; deprecated workloads lists wrong workload only
	ns := testutil.NewMockNamespace()
	svcDep := testutil.NewMockTestDeployment(ns, "svc")
	otherDep := testutil.NewMockTestDeployment(ns, "other")
	pw := testutil.PodWorkloadFromDeployment(svcDep)
	otherPw := testutil.PodWorkloadFromDeployment(otherDep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScopeAndWorkloads(
		"ir", ns.Name,
		[]k8sconsts.SourcesScope{{
			WorkloadName:      svcDep.Name,
			WorkloadNamespace: svcDep.Namespace,
			WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		}},
		[]k8sconsts.PodWorkload{otherPw},
	)

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.True(t, isParticipatingInRule, "sourcesScopes match is sufficient")
}

func Test_IsWorkloadParticipatingInRule_SourcesScopeOnly(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
	}})

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.True(t, isParticipatingInRule)
}

func Test_IsWorkloadParticipatingInRule_WorkloadsOnly(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleWithWorkloads("ir", ns.Name, []k8sconsts.PodWorkload{pw})

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.True(t, isParticipatingInRule)
}

func Test_IsWorkloadParticipatingInRule_SourcesScope_PartialNamespace_Match(t *testing.T) {
	// Arrange: only namespace set in scope — wildcard for name/kind
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadNamespace: dep.Namespace,
	}})

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.True(t, isParticipatingInRule)
}

func Test_IsWorkloadParticipatingInRule_SourcesScope_PartialNamespace_Miss(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadNamespace: "other-ns",
	}})

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.False(t, isParticipatingInRule)
}

func Test_IsWorkloadParticipatingInRule_SourcesScopeEmpty_Inactive(t *testing.T) {
	// Arrange: explicit empty sourcesScopes — inactive; workloads must not rescue
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	emptyScopes := []k8sconsts.SourcesScope{}
	rule := testutil.NewMockInstrumentationRuleWithSourcesScopeAndWorkloads("ir", ns.Name, emptyScopes, []k8sconsts.PodWorkload{pw})

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.False(t, isParticipatingInRule, "empty sourcesScopes means no participation")
}

func Test_IsWorkloadParticipatingInRule_AllWorkloadsWhenNoScope(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleAllWorkloads("ir", ns.Name)

	// Act
	isParticipatingInRule := utils.IsWorkloadParticipatingInRule(pw, rule, "", common.UnknownProgrammingLanguage)

	// Assert
	assert.True(t, isParticipatingInRule)
}

// ****************
// IsInstrumentationConfigParticipatingInRule tests
// ****************

func Test_IsInstrumentationConfigParticipatingInRule_Disabled_ShortCircuits(t *testing.T) {
	// Arrange: disabled must return false before any per-container work (callers rely on this; IC may list many containers)
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := &odigosv1alpha1.InstrumentationConfig{
		Spec: odigosv1alpha1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1alpha1.ContainerOverride{
				{ContainerName: "a"},
				{ContainerName: "b"},
				{ContainerName: "c"},
			},
		},
		Status: odigosv1alpha1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1alpha1.RuntimeDetailsByContainer{
				{ContainerName: "a", Language: common.GoProgrammingLanguage},
				{ContainerName: "b", Language: common.JavaProgrammingLanguage},
				{ContainerName: "c", Language: common.GoProgrammingLanguage},
			},
		},
	}
	rule := testutil.NewMockInstrumentationRuleDisabled("ir", ns.Name)

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.False(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_NilIC_DelegatesToWorkloadOnlyCheck(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	rule := testutil.NewMockInstrumentationRuleAllWorkloads("ir", ns.Name)

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, nil, rule)

	// Assert
	assert.True(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_NoContainerOverrides_Fallback(t *testing.T) {
	// Arrange: IC without containersOverrides — use status.runtimeDetailsByContainer for per-container scope (mock IC has container "test" + Go)
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := testutil.NewMockInstrumentationConfig(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.True(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_NoOverrides_StatusRuntime_ContainerScopeMiss(t *testing.T) {
	// Arrange: no ContainersOverrides; status lists container "test" but scope requires a different container name
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := testutil.NewMockInstrumentationConfig(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		ContainerName:     "not-the-test-container",
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.False(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_NoOverrides_StatusRuntime_ContainerAndLanguageMatch(t *testing.T) {
	// Arrange: scope pins container "test" and Go — matches NewMockInstrumentationConfig status
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := testutil.NewMockInstrumentationConfig(dep)
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		ContainerName:     "test",
		WorkloadLanguage:  common.GoProgrammingLanguage,
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.True(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_AnyContainerMatches(t *testing.T) {
	// Arrange: first container does not match scoped language; second does
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := &odigosv1alpha1.InstrumentationConfig{
		Spec: odigosv1alpha1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1alpha1.ContainerOverride{
				{ContainerName: "sidecar"},
				{ContainerName: "app"},
			},
		},
		Status: odigosv1alpha1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1alpha1.RuntimeDetailsByContainer{
				{ContainerName: "sidecar", Language: common.GoProgrammingLanguage},
				{ContainerName: "app", Language: common.JavaProgrammingLanguage},
			},
		},
	}
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		WorkloadLanguage:  common.JavaProgrammingLanguage,
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.True(t, got, "OR across containers: app satisfies Java scope")
}

func Test_IsInstrumentationConfigParticipatingInRule_AllContainersMiss(t *testing.T) {
	// Arrange
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := &odigosv1alpha1.InstrumentationConfig{
		Spec: odigosv1alpha1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1alpha1.ContainerOverride{
				{ContainerName: "app"},
			},
		},
		Status: odigosv1alpha1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1alpha1.RuntimeDetailsByContainer{
				{ContainerName: "app", Language: common.GoProgrammingLanguage},
			},
		},
	}
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		WorkloadLanguage:  common.JavaProgrammingLanguage,
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.False(t, got)
}

func Test_IsInstrumentationConfigParticipatingInRule_ContainerNameInScope(t *testing.T) {
	// Arrange: scope pins container name
	ns := testutil.NewMockNamespace()
	dep := testutil.NewMockTestDeployment(ns, "svc")
	pw := testutil.PodWorkloadFromDeployment(dep)
	ic := &odigosv1alpha1.InstrumentationConfig{
		Spec: odigosv1alpha1.InstrumentationConfigSpec{
			ContainersOverrides: []odigosv1alpha1.ContainerOverride{
				{ContainerName: "nginx"},
				{ContainerName: "app"},
			},
		},
		Status: odigosv1alpha1.InstrumentationConfigStatus{
			RuntimeDetailsByContainer: []odigosv1alpha1.RuntimeDetailsByContainer{
				{ContainerName: "nginx", Language: common.JavaProgrammingLanguage},
				{ContainerName: "app", Language: common.JavaProgrammingLanguage},
			},
		},
	}
	rule := testutil.NewMockInstrumentationRuleWithSourcesScope("ir", ns.Name, []k8sconsts.SourcesScope{{
		WorkloadName:      dep.Name,
		WorkloadNamespace: dep.Namespace,
		WorkloadKind:      string(k8sconsts.WorkloadKindDeployment),
		ContainerName:     "app",
	}})

	// Act
	got := utils.IsInstrumentationConfigParticipatingInRule(pw, ic, rule)

	// Assert
	assert.True(t, got)
}
