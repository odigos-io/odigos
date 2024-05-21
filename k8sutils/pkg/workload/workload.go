package workload

import (
	"errors"
	"strings"
)

// runtime name is a way to store workload specific CRs with odigos
// and give the k8s object a name which is unique and can be used to extract the workload name and kind
func GetRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}

func GetWorkloadInfoRuntimeName(name string) (workloadName string, workloadKind string, err error) {
	hyphenIndex := strings.Index(name, "-")
	if hyphenIndex == -1 {
		err = errors.New("invalid runtime name")
		return
	}

	workloadKind, err = kindFromLowercase(name[:hyphenIndex])
	if err != nil {
		return
	}
	workloadName = name[hyphenIndex+1:]
	return
}

func kindFromLowercase(lowercaseKind string) (string, error) {
	switch lowercaseKind {
	case "deployment":
		return "Deployment", nil
	case "statefulset":
		return "StatefulSet", nil
	case "daemonset":
		return "DaemonSet", nil
	default:
		return "", errors.New("unknown kind")
	}
}
