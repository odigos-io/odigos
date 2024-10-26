package workload

import (
	"context"
	"errors"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// this file contains utils related to odigos workload runtime object names.
// the format is <workload-kind>-<workload-name>
// where the workload kind is lower case string (deployment, daemonset, statefulset)
// and then a hyphen and the workload name
// example: deployment-myapp

func CalculateWorkloadRuntimeObjectName[T string | WorkloadKind | WorkloadKindLowerCase](workloadName string, workloadKind T) string {
	return strings.ToLower(string(workloadKind) + "-" + workloadName)
}

func ExtractWorkloadInfoFromRuntimeObjectName(runtimeObjectName string) (workloadName string, workloadKind WorkloadKind, err error) {
	parts := strings.SplitN(runtimeObjectName, "-", 2)
	if len(parts) != 2 {
		err = errors.New("invalid workload runtime object name, missing hyphen")
		return
	}

	//convert the lowercase kind to pascal case and validate it
	workloadKindLowerCase := WorkloadKindLowerCase(parts[0])
	workloadKind = WorkloadKindFromLowerCase(workloadKindLowerCase)
	if workloadKind == "" {
		err = ErrKindNotSupported
		return
	}

	workloadName = parts[1]

	return
}

func GetRuntimeDetailsForPod(ctx context.Context, kubeClient client.Client, pod *corev1.Pod) (*odigosv1.InstrumentedApplication, error) {
	var workloadName string
	var workloadKind WorkloadKind
	for _, owner := range pod.OwnerReferences {
		wn, wk, err := GetWorkloadFromOwnerReference(owner)
		if IgnoreErrorKindNotSupported(err) != nil {
			return nil, err
		}

		workloadName = wn
		workloadKind = wk
		break
	}

	instrumentedApplicationName := CalculateWorkloadRuntimeObjectName(workloadName, workloadKind)

	var runtimeDetails odigosv1.InstrumentedApplication
	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: pod.Namespace,
		Name:      instrumentedApplicationName,
	}, &runtimeDetails)
	if err != nil {
		return nil, err
	}

	return &runtimeDetails, nil
}
