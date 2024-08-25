package workload

import (
	"errors"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetWorkloadFromOwnerReference(ownerReference metav1.OwnerReference) (workloadName string, workloadKind WorkloadKind, err error) {
	ownerName := ownerReference.Name
	ownerKind := ownerReference.Kind
	if ownerKind == "ReplicaSet" {
		// ReplicaSet name is in the format <deployment-name>-<random-string>
		hyphenIndex := strings.LastIndex(ownerName, "-")
		if hyphenIndex == -1 {
			// It is possible for a user to define a bare ReplicaSet without a deployment, currently not supporting this
			err = errors.New("replicaset name does not contain a hyphen")
			return
		}
		// Extract deployment name from ReplicaSet name
		workloadName = ownerName[:hyphenIndex]
		workloadKind = WorkloadKindDeployment
		return
	}

	workloadKind = WorkloadKindFromString(ownerKind)
	if workloadKind == "" {
		err = ErrKindNotSupported
		return
	}
	workloadName = ownerName

	err = nil
	return
}
