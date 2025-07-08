package workload

import (
	"context"
	"errors"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// PodWorkloadObjectOrError is the same as PodWorkloadObject but returns an error if the workload is not found.
func PodWorkloadObjectOrError(ctx context.Context, pod *corev1.Pod) (*k8sconsts.PodWorkload, error) {
	pw, err := PodWorkloadObject(ctx, pod)
	if err != nil {
		return nil, err
	}

	if pw == nil {
		return nil, fmt.Errorf("workload not found for pod %s/%s", pod.Namespace, pod.Name)
	}

	return pw, nil
}

// PodWorkload returns the workload object that manages the provided pod.
// If the pod is not owned by a controller, it returns a nil workload with no error.
func PodWorkloadObject(ctx context.Context, pod *corev1.Pod) (*k8sconsts.PodWorkload, error) {
	for _, owner := range pod.OwnerReferences {
		workloadName, workloadKind, err := GetWorkloadFromOwnerReference(owner)
		if err != nil {
			if errors.Is(err, ErrKindNotSupported) {
				continue
			}
			return nil, IgnoreErrorKindNotSupported(err)
		}

		return &k8sconsts.PodWorkload{
			Name:      workloadName,
			Kind:      workloadKind,
			Namespace: pod.Namespace,
		}, nil
	}

	// Pod does not necessarily have to be managed by a controller
	return nil, nil
}

// GetWorkloadFromOwnerReference retrieves both the workload name and workload kind
// from the provided owner reference.
func GetWorkloadFromOwnerReference(ownerReference metav1.OwnerReference) (workloadName string, workloadKind k8sconsts.WorkloadKind, err error) {
	return GetWorkloadNameAndKind(ownerReference.Name, ownerReference.Kind)
}

func GetWorkloadNameAndKind(ownerName, ownerKind string) (string, k8sconsts.WorkloadKind, error) {
	switch ownerKind {
	case "ReplicaSet":
		return extractInfoWithSuffix(ownerName, k8sconsts.WorkloadKindDeployment)
	case "Job":
		return extractInfoWithSuffix(ownerName, k8sconsts.WorkloadKindCronJob)
	default:
		return extractInfoWithoutSuffix(ownerName, ownerKind)
	}
}

// extractInfoWithSuffix strips Kubernetes-generated suffixes from owner reference names.
// ReplicaSets and Jobs get unique suffixes appended (e.g., "app-name-7d4c8b5f9b").
// This extracts the base name by removing everything after the last hyphen,
// enabling grouping of resources by their logical application identity..
func extractInfoWithSuffix(fullName string, kind k8sconsts.WorkloadKind) (string, k8sconsts.WorkloadKind, error) {
	hyphenIndex := strings.LastIndex(fullName, "-")
	if hyphenIndex == -1 {
		return "", "", fmt.Errorf("%s name '%s' does not contain a hyphen", kind, fullName)
	}
	return fullName[:hyphenIndex], kind, nil
}

// extractInfoWithoutSuffix processes non-suffix-based workload names
func extractInfoWithoutSuffix(ownerName, ownerKind string) (string, k8sconsts.WorkloadKind, error) {
	workloadKind := WorkloadKindFromString(ownerKind)
	if workloadKind == "" {
		return "", "", ErrKindNotSupported
	}
	return ownerName, workloadKind, nil
}
