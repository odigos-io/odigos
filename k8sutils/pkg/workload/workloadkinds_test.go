package workload_test

import (
	"errors"
	"strings"
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestWorkloadKindLowerCaseFromKind(t *testing.T) {
	dep := workload.WorkloadKindLowerCaseFromKind(k8sconsts.WorkloadKindDeployment)
	assert.Equal(t, k8sconsts.WorkloadKindLowerCaseDeployment, dep)
	ds := workload.WorkloadKindLowerCaseFromKind(k8sconsts.WorkloadKindDaemonSet)
	assert.Equal(t, k8sconsts.WorkloadKindLowerCaseDaemonSet, ds)
	ss := workload.WorkloadKindLowerCaseFromKind(k8sconsts.WorkloadKindStatefulSet)
	assert.Equal(t, k8sconsts.WorkloadKindLowerCaseStatefulSet, ss)
	invalid := workload.WorkloadKindLowerCaseFromKind("Invalid")
	assert.Equal(t, k8sconsts.WorkloadKindLowerCase(""), invalid)
}

func TestWorkloadKindFromLowerCase(t *testing.T) {
	dep := workload.WorkloadKindFromLowerCase(k8sconsts.WorkloadKindLowerCaseDeployment)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, dep)
	ds := workload.WorkloadKindFromLowerCase(k8sconsts.WorkloadKindLowerCaseDaemonSet)
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, ds)
	ss := workload.WorkloadKindFromLowerCase(k8sconsts.WorkloadKindLowerCaseStatefulSet)
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, ss)
	invalid := workload.WorkloadKindFromLowerCase("Invalid")
	assert.Equal(t, k8sconsts.WorkloadKind(""), invalid)
}

func TestWorkloadKindFromString(t *testing.T) {
	depLower := workload.WorkloadKindFromString("deployment")
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, depLower)
	depPascal := workload.WorkloadKindFromString("Deployment")
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, depPascal)

	dsLower := workload.WorkloadKindFromString("daemonset")
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, dsLower)
	dsPascal := workload.WorkloadKindFromString("DaemonSet")
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, dsPascal)

	ssLower := workload.WorkloadKindFromString("statefulset")
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, ssLower)
	ssPascal := workload.WorkloadKindFromString("StatefulSet")
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, ssPascal)

	invalid := workload.WorkloadKindFromString("Invalid")
	assert.Equal(t, k8sconsts.WorkloadKind(""), invalid)
}

func TestClientObjectFromWorkloadKind(t *testing.T) {
	dep := workload.ClientObjectFromWorkloadKind(k8sconsts.WorkloadKindDeployment)
	assert.Equal(t, &appsv1.Deployment{}, dep)
	ds := workload.ClientObjectFromWorkloadKind(k8sconsts.WorkloadKindDaemonSet)
	assert.Equal(t, &appsv1.DaemonSet{}, ds)
	ss := workload.ClientObjectFromWorkloadKind(k8sconsts.WorkloadKindStatefulSet)
	assert.Equal(t, &appsv1.StatefulSet{}, ss)
	invalid := workload.ClientObjectFromWorkloadKind("Invalid")
	assert.Equal(t, nil, invalid)
}

// Every workload kind odigos knows about, in both spellings. Each helper below
// reimplements this list as its own switch statement, and none of them reference
// each other, so a kind missing from one is silent at runtime: the workload
// resolves to the empty kind or to a nil object instead of being reported.
var allWorkloadKinds = []struct {
	pascalCase k8sconsts.WorkloadKind
	lowerCase  k8sconsts.WorkloadKindLowerCase
}{
	{k8sconsts.WorkloadKindDeployment, k8sconsts.WorkloadKindLowerCaseDeployment},
	{k8sconsts.WorkloadKindDaemonSet, k8sconsts.WorkloadKindLowerCaseDaemonSet},
	{k8sconsts.WorkloadKindStatefulSet, k8sconsts.WorkloadKindLowerCaseStatefulSet},
	{k8sconsts.WorkloadKindNamespace, k8sconsts.WorkloadKindLowerCaseNamespace},
	{k8sconsts.WorkloadKindStaticPod, k8sconsts.WorkloadKindLowerCaseStaticPod},
	{k8sconsts.WorkloadKindCronJob, k8sconsts.WorkloadKindLowerCaseCronJob},
	{k8sconsts.WorkloadKindJob, k8sconsts.WorkloadKindLowerCaseJob},
	{k8sconsts.WorkloadKindDeploymentConfig, k8sconsts.WorkloadKindLowerCaseDeploymentConfig},
	{k8sconsts.WorkloadKindArgoRollout, k8sconsts.WorkloadKindLowerCaseArgoRollout},
}

func TestWorkloadKindMappingsCoverEveryKind(t *testing.T) {
	for _, kind := range allWorkloadKinds {
		t.Run(string(kind.pascalCase), func(t *testing.T) {
			assert.Equal(t, kind.lowerCase, workload.WorkloadKindLowerCaseFromKind(kind.pascalCase))
			assert.Equal(t, kind.pascalCase, workload.WorkloadKindFromLowerCase(kind.lowerCase))

			// The lower case spelling is exactly the pascal case one lowercased,
			// which is what lets WorkloadKindFromString accept either form. The
			// lower case form is also the prefix of instrumentation config object
			// names, so it is persisted in the cluster and cannot drift.
			assert.Equal(t, strings.ToLower(string(kind.pascalCase)), string(kind.lowerCase))

			assert.Equal(t, kind.pascalCase, workload.WorkloadKindFromString(string(kind.pascalCase)))
			assert.Equal(t, kind.pascalCase, workload.WorkloadKindFromString(string(kind.lowerCase)))
		})
	}
}

// Job is deliberately not a valid workload kind: odigos instruments the CronJob
// that owns the pods rather than the Job itself, and ObjectToWorkload has no Job
// case either. Spelling the exception out means a kind accidentally dropped from
// the switch fails here instead of silently making its Sources unacceptable.
func TestIsValidWorkloadKind(t *testing.T) {
	for _, kind := range allWorkloadKinds {
		t.Run(string(kind.pascalCase), func(t *testing.T) {
			want := kind.pascalCase != k8sconsts.WorkloadKindJob
			assert.Equal(t, want, workload.IsValidWorkloadKind(kind.pascalCase))
		})
	}

	assert.False(t, workload.IsValidWorkloadKind(""))
	assert.False(t, workload.IsValidWorkloadKind("Pod"))
	assert.False(t, workload.IsValidWorkloadKind("ReplicaSet"))
	// only the pascal case spelling is a workload kind.
	assert.False(t, workload.IsValidWorkloadKind("deployment"))
}

// The returned object decides which resource type the controller fetches from the
// api server, so two kinds mapped to the same type would read the wrong resource.
func TestClientObjectFromWorkloadKindForEveryKind(t *testing.T) {
	tests := []struct {
		kind k8sconsts.WorkloadKind
		want client.Object
	}{
		{kind: k8sconsts.WorkloadKindDeployment, want: &appsv1.Deployment{}},
		{kind: k8sconsts.WorkloadKindDaemonSet, want: &appsv1.DaemonSet{}},
		{kind: k8sconsts.WorkloadKindStatefulSet, want: &appsv1.StatefulSet{}},
		{kind: k8sconsts.WorkloadKindNamespace, want: &corev1.Namespace{}},
		{kind: k8sconsts.WorkloadKindStaticPod, want: &corev1.Pod{}},
		{kind: k8sconsts.WorkloadKindCronJob, want: &batchv1.CronJob{}},
		{kind: k8sconsts.WorkloadKindJob, want: &batchv1.Job{}},
		{kind: k8sconsts.WorkloadKindDeploymentConfig, want: &openshiftappsv1.DeploymentConfig{}},
		{kind: k8sconsts.WorkloadKindArgoRollout, want: &argorolloutsv1alpha1.Rollout{}},
	}
	assert.Equal(t, len(allWorkloadKinds), len(tests))

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			assert.Equal(t, tt.want, workload.ClientObjectFromWorkloadKind(tt.kind))
		})
	}
}

// Namespace is the one kind with no list type: it scopes source level
// configuration rather than being something odigos enumerates workloads of.
func TestClientListObjectFromWorkloadKind(t *testing.T) {
	tests := []struct {
		kind k8sconsts.WorkloadKind
		want client.ObjectList
	}{
		{kind: k8sconsts.WorkloadKindDeployment, want: &appsv1.DeploymentList{}},
		{kind: k8sconsts.WorkloadKindDaemonSet, want: &appsv1.DaemonSetList{}},
		{kind: k8sconsts.WorkloadKindStatefulSet, want: &appsv1.StatefulSetList{}},
		{kind: k8sconsts.WorkloadKindNamespace, want: nil},
		{kind: k8sconsts.WorkloadKindStaticPod, want: &corev1.PodList{}},
		{kind: k8sconsts.WorkloadKindCronJob, want: &batchv1.CronJobList{}},
		{kind: k8sconsts.WorkloadKindJob, want: &batchv1.JobList{}},
		{kind: k8sconsts.WorkloadKindDeploymentConfig, want: &openshiftappsv1.DeploymentConfigList{}},
		{kind: k8sconsts.WorkloadKindArgoRollout, want: &argorolloutsv1alpha1.RolloutList{}},
	}
	assert.Equal(t, len(allWorkloadKinds), len(tests))

	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			got := workload.ClientListObjectFromWorkloadKind(tt.kind)
			if tt.want == nil {
				assert.Nil(t, got)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}

	assert.Nil(t, workload.ClientListObjectFromWorkloadKind("Invalid"))
}

func TestIsErrorKindNotSupported(t *testing.T) {
	assert.True(t, workload.IsErrorKindNotSupported(workload.ErrKindNotSupported))
	assert.False(t, workload.IsErrorKindNotSupported(errors.New("workload kind not supported")))
	assert.False(t, workload.IsErrorKindNotSupported(errors.New("some other error")))
	assert.False(t, workload.IsErrorKindNotSupported(nil))

	// odiglet and the ownerreference helpers match the sentinel with errors.Is,
	// so it has to stay a comparable sentinel value.
	assert.True(t, errors.Is(workload.ErrKindNotSupported, workload.ErrKindNotSupported))
}

// Callers walk pods of kinds odigos does not instrument, and use this to drop the
// unsupported-kind error while still surfacing every other failure.
func TestIgnoreErrorKindNotSupported(t *testing.T) {
	assert.Nil(t, workload.IgnoreErrorKindNotSupported(workload.ErrKindNotSupported))
	assert.Nil(t, workload.IgnoreErrorKindNotSupported(nil))

	other := errors.New("api server is unreachable")
	assert.Same(t, other, workload.IgnoreErrorKindNotSupported(other))

	lookalike := errors.New("workload kind not supported")
	assert.Same(t, lookalike, workload.IgnoreErrorKindNotSupported(lookalike))
}
