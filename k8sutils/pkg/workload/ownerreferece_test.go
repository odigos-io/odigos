package workload_test

import (
	"testing"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetWorkloadFromOwnerReferenceWithReplicaSet(t *testing.T) {
	name, kind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "mydeployment-1234",
		Kind: "ReplicaSet",
	})
	assert.Nil(t, err)
	assert.Equal(t, "mydeployment", name)
	assert.Equal(t, workload.WorkloadKindDeployment, kind)
}

func TestGetWorkloadFromOwnerReferenceWithDaemonSet(t *testing.T) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-ds",
		Kind: string(workload.WorkloadKindDaemonSet),
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-ds", workloadName)
	assert.Equal(t, workload.WorkloadKindDaemonSet, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithStatefulSet(t *testing.T) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-ss",
		Kind: string(workload.WorkloadKindStatefulSet),
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-ss", workloadName)
	assert.Equal(t, workload.WorkloadKindStatefulSet, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithDeployment(t *testing.T) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-deployment",
		Kind: string(workload.WorkloadKindDeployment),
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-deployment", workloadName)
	assert.Equal(t, workload.WorkloadKindDeployment, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithInvalidKind(t *testing.T) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-deployment",
		Kind: "Invalid",
	})
	assert.NotNil(t, err)
	assert.Equal(t, "", workloadName)
	assert.Equal(t, workload.WorkloadKind(""), workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithInvalidReplicaSet(t *testing.T) {
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "customreplicaset",
		Kind: "ReplicaSet",
	})
	assert.NotNil(t, err)
	assert.Equal(t, "", workloadName)
	assert.Equal(t, workload.WorkloadKind(""), workloadKind)
}
