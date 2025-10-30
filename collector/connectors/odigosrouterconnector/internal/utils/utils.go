package utils

import (
	"errors"
	"fmt"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv1_26 "go.opentelemetry.io/otel/semconv/v1.26.0"
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
	workloadKind := parts[0]
	workloadName := parts[1]

	workloadKey := WorkloadKeyFromParts(ns, workloadKind, workloadName)
	return workloadKey, nil
}

func ResourceAttributesToWorkloadKey(attrs pcommon.Map) *WorkloadKey {
	nsAttr, ok := attrs.Get(string(semconv1_26.K8SNamespaceNameKey))
	if !ok {
		return nil
	}
	ns := nsAttr.Str()

	name, kind := getDynamicNameAndKind(attrs)
	if name == "" || kind == "" {
		return nil
	}

	workloadKey := WorkloadKeyFromParts(ns, kind, name)
	return &workloadKey
}

// getDynamicNameAndKind extracts the workload name and kind from a resource's attributes.
// It searches for known Kubernetes keys such as deployment, statefulset, and daemonset,
// and returns the first matched workload name and its corresponding kind.
// If none are found, it returns empty strings for both.
func getDynamicNameAndKind(attrs pcommon.Map) (name string, kind string) {
	if name, ok := attrs.Get(string(semconv1_26.K8SDeploymentNameKey)); ok {
		return name.Str(), "deployment"
	}
	if name, ok := attrs.Get(string(semconv1_26.K8SStatefulSetNameKey)); ok {
		return name.Str(), "statefulset"
	}
	if name, ok := attrs.Get(string(semconv1_26.K8SDaemonSetNameKey)); ok {
		return name.Str(), "daemonset"
	}
	if name, ok := attrs.Get(string(semconv1_26.K8SCronJobNameKey)); ok {
		return name.Str(), "cronjob"
	}
	if name, ok := attrs.Get(string(semconv1_26.K8SJobNameKey)); ok {
		return name.Str(), "job"
	}
	return "", ""
}
