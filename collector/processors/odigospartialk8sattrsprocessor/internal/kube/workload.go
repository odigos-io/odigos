package kube

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// getWorkloadNameAndKind resolves the workload name and kind from owner reference information.
// This is a simplified version of k8sutils/pkg/workload.GetWorkloadNameAndKind that doesn't
// require the heavy dependencies from the full k8sutils package.
func getWorkloadNameAndKind(ownerName, ownerKind string, pod *corev1.Pod) (string, WorkloadKind, error) {
	switch ownerKind {
	case "ReplicaSet":
		return determineReplicaSetOwner(ownerName, pod)
	case "ReplicationController":
		return extractInfoWithSuffix(ownerName, WorkloadKindDeploymentConfig)
	case "Job":
		return extractInfoWithSuffix(ownerName, WorkloadKindCronJob)
	default:
		return extractInfoWithoutSuffix(ownerName, ownerKind)
	}
}

// determineReplicaSetOwner checks if a ReplicaSet is owned by a Deployment or Argo Rollout
func determineReplicaSetOwner(ownerName string, pod *corev1.Pod) (string, WorkloadKind, error) {
	// If we find a label associated with Argo rollouts, it is a Rollout kind
	if _, ok := pod.Labels[argoRolloutUniqueLabelKey]; ok {
		return extractInfoWithSuffix(ownerName, WorkloadKindArgoRollout)
	}
	// Default to Deployment kind
	return extractInfoWithSuffix(ownerName, WorkloadKindDeployment)
}

// extractInfoWithSuffix strips Kubernetes-generated suffixes from owner reference names.
// ReplicaSets and Jobs get unique suffixes appended (e.g., "app-name-7d4c8b5f9b").
// This extracts the base name by removing everything after the last hyphen.
func extractInfoWithSuffix(fullName string, kind WorkloadKind) (string, WorkloadKind, error) {
	hyphenIndex := strings.LastIndex(fullName, "-")
	if hyphenIndex == -1 {
		return "", "", fmt.Errorf("%s name '%s' does not contain a hyphen", kind, fullName)
	}
	return fullName[:hyphenIndex], kind, nil
}

// extractInfoWithoutSuffix handles workload kinds that don't have generated suffixes
func extractInfoWithoutSuffix(ownerName, ownerKind string) (string, WorkloadKind, error) {
	kind := WorkloadKind(ownerKind)
	switch kind {
	case WorkloadKindDeployment, WorkloadKindDaemonSet, WorkloadKindStatefulSet, WorkloadKindJob:
		return ownerName, kind, nil
	default:
		return "", "", fmt.Errorf("unknown workload kind: %s", ownerKind)
	}
}
