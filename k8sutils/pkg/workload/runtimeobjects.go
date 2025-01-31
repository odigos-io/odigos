package workload

import (
	"errors"
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// this file contains utils related to odigos workload runtime object names.
// the format is <workload-kind>-<workload-name>
// where the workload kind is lower case string (deployment, daemonset, statefulset)
// and then a hyphen and the workload name
// example: deployment-myapp

func CalculateWorkloadRuntimeObjectName[T string | k8sconsts.WorkloadKind | k8sconsts.WorkloadKindLowerCase](
	workloadName string, workloadKind T) string {
	return strings.ToLower(string(workloadKind) + "-" + workloadName)
}

func ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName string) (workloadName string, workloadKind k8sconsts.WorkloadKind, err error) {
	parts := strings.SplitN(runtimeObjectName, "-", 2)
	if len(parts) != 2 {
		err = errors.New("invalid workload runtime object name, missing hyphen")
		return
	}

	// convert the lowercase kind to pascal case and validate it
	workloadKindLowerCase := k8sconsts.WorkloadKindLowerCase(parts[0])
	workloadKind = WorkloadKindFromLowerCase(workloadKindLowerCase)
	if workloadKind == "" {
		err = ErrKindNotSupported
		return
	}

	workloadName = parts[1]

	return
}
