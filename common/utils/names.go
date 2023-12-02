package utils

import (
	"errors"
	"strings"
)

func GetRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}

func GetTargetFromRuntimeName(name string) (string, string, error) {
	hyphenIndex := strings.Index(name, "-")
	if hyphenIndex == -1 {
		return "", "", errors.New("invalid runtime name")
	}

	workloadKind, err := kindFromLowercase(name[:hyphenIndex])
	if err != nil {
		return "", "", err
	}
	workloadName := name[hyphenIndex+1:]

	return workloadName, workloadKind, nil
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
