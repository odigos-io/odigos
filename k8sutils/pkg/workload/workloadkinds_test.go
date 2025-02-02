package workload_test

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
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

func TestWorkloadKindFromClientObject(t *testing.T) {
	dep := workload.WorkloadKindFromClientObject(&appsv1.Deployment{})
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, dep)
	ds := workload.WorkloadKindFromClientObject(&appsv1.DaemonSet{})
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, ds)
	ss := workload.WorkloadKindFromClientObject(&appsv1.StatefulSet{})
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, ss)
	invalid := workload.WorkloadKindFromClientObject(&appsv1.ReplicaSet{})
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
