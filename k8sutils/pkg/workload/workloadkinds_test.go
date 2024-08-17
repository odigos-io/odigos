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
