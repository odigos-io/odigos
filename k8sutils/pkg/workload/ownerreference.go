package workload

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetWorkloadFromOwnerReference retrieves both the workload name and workload kind
// from the provided owner reference.
func GetWorkloadFromOwnerReference(ownerReference metav1.OwnerReference) (workloadName string, workloadKind WorkloadKind, err error) {

	return GetWorkloadNameAndKind(ownerReference.Name, ownerReference.Kind)
}

func GetWorkloadNameAndKind(ownerName, ownerKind string) (string, WorkloadKind, error) {
	if ownerKind == "ReplicaSet" {
		return extractDeploymentInfo(ownerName)
	}
	return handleNonReplicaSet(ownerName, ownerKind)
}

// extractDeploymentInfo extracts deployment information from a ReplicaSet name
func extractDeploymentInfo(replicaSetName string) (string, WorkloadKind, error) {
	hyphenIndex := strings.LastIndex(replicaSetName, "-")
	if hyphenIndex == -1 {
		return "", "", fmt.Errorf("replicaset name '%s' does not contain a hyphen", replicaSetName)
	}

	deploymentName := replicaSetName[:hyphenIndex]
	return deploymentName, WorkloadKindDeployment, nil
}

// handleNonReplicaSet processes non-ReplicaSet workload types
func handleNonReplicaSet(ownerName, ownerKind string) (string, WorkloadKind, error) {
	workloadKind := WorkloadKindFromString(ownerKind)
	if workloadKind == "" {
		return "", "", ErrKindNotSupported
	}

	return ownerName, workloadKind, nil
}
func PodWorkloadObject(ctx context.Context, pod *corev1.Pod) (*PodWorkload, error) {
	for _, owner := range pod.OwnerReferences {
		workloadName, workloadKind, err := GetWorkloadFromOwnerReference(owner)
		if err != nil {
			if errors.Is(err, ErrKindNotSupported) {
				continue
			}
			return nil, IgnoreErrorKindNotSupported(err)
		}

		return &PodWorkload{
			Name:      workloadName,
			Kind:      workloadKind,
			Namespace: pod.Namespace,
		}, nil
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}
