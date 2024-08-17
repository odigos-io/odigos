package workload_test

import (
	"testing"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	appsv1 "k8s.io/api/apps/v1"
)

func TestWorkloadKindLowerCaseFromKind(t *testing.T) {
	dep := workload.WorkloadKindLowerCaseFromKind(workload.WorkloadKindDeployment)
	assert.Equal(t, workload.WorkloadKindLowerCaseDeployment, dep)
	ds := workload.WorkloadKindLowerCaseFromKind(workload.WorkloadKindDaemonSet)
	assert.Equal(t, workload.WorkloadKindLowerCaseDaemonSet, ds)
	ss := workload.WorkloadKindLowerCaseFromKind(workload.WorkloadKindStatefulSet)
	assert.Equal(t, workload.WorkloadKindLowerCaseStatefulSet, ss)
	invalid := workload.WorkloadKindLowerCaseFromKind("Invalid")
	assert.Equal(t, workload.WorkloadKindLowerCase(""), invalid)
}

func TestWorkloadKindFromLowerCase(t *testing.T) {
	dep := workload.WorkloadKindFromLowerCase(workload.WorkloadKindLowerCaseDeployment)
	assert.Equal(t, workload.WorkloadKindDeployment, dep)
	ds := workload.WorkloadKindFromLowerCase(workload.WorkloadKindLowerCaseDaemonSet)
	assert.Equal(t, workload.WorkloadKindDaemonSet, ds)
	ss := workload.WorkloadKindFromLowerCase(workload.WorkloadKindLowerCaseStatefulSet)
	assert.Equal(t, workload.WorkloadKindStatefulSet, ss)
	invalid := workload.WorkloadKindFromLowerCase("Invalid")
	assert.Equal(t, workload.WorkloadKind(""), invalid)
}

func TestWorkloadKindFromString(t *testing.T) {
	depLower := workload.WorkloadKindFromString("deployment")
	assert.Equal(t, workload.WorkloadKindDeployment, depLower)
	depPascal := workload.WorkloadKindFromString("Deployment")
	assert.Equal(t, workload.WorkloadKindDeployment, depPascal)

	dsLower := workload.WorkloadKindFromString("daemonset")
	assert.Equal(t, workload.WorkloadKindDaemonSet, dsLower)
	dsPascal := workload.WorkloadKindFromString("DaemonSet")
	assert.Equal(t, workload.WorkloadKindDaemonSet, dsPascal)

	ssLower := workload.WorkloadKindFromString("statefulset")
	assert.Equal(t, workload.WorkloadKindStatefulSet, ssLower)
	ssPascal := workload.WorkloadKindFromString("StatefulSet")
	assert.Equal(t, workload.WorkloadKindStatefulSet, ssPascal)

	invalid := workload.WorkloadKindFromString("Invalid")
	assert.Equal(t, workload.WorkloadKind(""), invalid)
}

func TestWorkloadKindFromClientObject(t *testing.T) {
	dep := workload.WorkloadKindFromClientObject(&appsv1.Deployment{})
	assert.Equal(t, workload.WorkloadKindDeployment, dep)
	ds := workload.WorkloadKindFromClientObject(&appsv1.DaemonSet{})
	assert.Equal(t, workload.WorkloadKindDaemonSet, ds)
	ss := workload.WorkloadKindFromClientObject(&appsv1.StatefulSet{})
	assert.Equal(t, workload.WorkloadKindStatefulSet, ss)
	invalid := workload.WorkloadKindFromClientObject(&appsv1.ReplicaSet{})
	assert.Equal(t, workload.WorkloadKind(""), invalid)
}

func TestClientObjectFromWorkloadKind(t *testing.T) {
	dep := workload.ClientObjectFromWorkloadKind(workload.WorkloadKindDeployment)
	assert.Equal(t, &appsv1.Deployment{}, dep)
	ds := workload.ClientObjectFromWorkloadKind(workload.WorkloadKindDaemonSet)
	assert.Equal(t, &appsv1.DaemonSet{}, ds)
	ss := workload.ClientObjectFromWorkloadKind(workload.WorkloadKindStatefulSet)
	assert.Equal(t, &appsv1.StatefulSet{}, ss)
	invalid := workload.ClientObjectFromWorkloadKind("Invalid")
	assert.Equal(t, nil, invalid)
}
