package workload_test

import (
	"testing"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/tj/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetWorkloadFromOwnerReferenceWithReplicaSet(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	name, kind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "mydeployment-1234",
		Kind: "ReplicaSet",
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "mydeployment", name)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, kind)
}

func TestGetWorkloadFromOwnerReferenceWithDaemonSet(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-ds",
		Kind: string(k8sconsts.WorkloadKindDaemonSet),
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "my-ds", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindDaemonSet, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithStatefulSet(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-ss",
		Kind: string(k8sconsts.WorkloadKindStatefulSet),
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "my-ss", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithDeployment(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-deployment",
		Kind: string(k8sconsts.WorkloadKindDeployment),
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "my-deployment", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithInvalidKind(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "my-deployment",
		Kind: "Invalid",
	}, pod)
	assert.NotNil(t, err)
	assert.Equal(t, "", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKind(""), workloadKind)
}

func TestGetWorkloadFromOwnerReferenceWithInvalidReplicaSet(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "default"},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "customreplicaset",
		Kind: "ReplicaSet",
	}, pod)
	assert.NotNil(t, err)
	assert.Equal(t, "", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKind(""), workloadKind)
}

func TestGetWorkloadFromOwnerReferenceReplicaSetOwnedByRollout(t *testing.T) {
	// Pod with Argo Rollouts label should be identified as a Rollout
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Labels: map[string]string{
				argorolloutsv1alpha1.DefaultRolloutUniqueLabelKey: "abc123",
			},
		},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "myrollout-abc123",
		Kind: "ReplicaSet",
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "myrollout", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindArgoRollout, workloadKind)
}

func TestGetWorkloadFromOwnerReferenceReplicaSetOwnedByDeployment(t *testing.T) {
	// Pod without Argo Rollouts label should be identified as a Deployment
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Labels:    map[string]string{},
		},
	}
	workloadName, workloadKind, err := workload.GetWorkloadFromOwnerReference(metav1.OwnerReference{
		Name: "mydeployment-7d4c8b5f9b",
		Kind: "ReplicaSet",
	}, pod)
	assert.Nil(t, err)
	assert.Equal(t, "mydeployment", workloadName)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, workloadKind)
}
