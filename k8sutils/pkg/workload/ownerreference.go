package workload

import (
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GetWorkloadFromOwnerReference retrieves both the workload name and workload kind
// from the provided owner reference.
func GetWorkloadFromOwnerReference(ownerReference metav1.OwnerReference) (workloadName string, workloadKind WorkloadKind, err error) {

	workloadName, workloadKind, err = GetWorkloadNameAndKind(ownerReference.Name, ownerReference.Kind)
	if err != nil {
		return "", "", err
	}

	return workloadName, workloadKind, nil
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
