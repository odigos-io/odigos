package utils

import (
	"errors"
	"fmt"
	"strings"
)

// WorkloadKey is a string in the format <namespace>/<workload-kind>/<workload-name>.
// it is used in order to have one string fast access into the workload cache map.
type WorkloadKey string

// name of a datastream
type DatastreamName string

func WorkloadKeyFromParts(ns string, workloadKind string, workloadName string) WorkloadKey {
	return WorkloadKey(fmt.Sprintf("%s/%s/%s", ns, workloadKind, workloadName))
}

func InstrumentationConfigToWorkloadKey(ns string, icName string) (WorkloadKey, error) {
	// do not use the existing function from k8sutils,
	// in order to avoid pulling in it's dependencies and create conflicts with collector dependencies
	parts := strings.SplitN(icName, "-", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid workload runtime object name, missing hyphen")
	}

	workloadKey := WorkloadKeyFromParts(ns, parts[1], parts[0])
	return workloadKey, nil
}
