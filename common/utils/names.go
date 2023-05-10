package utils

import (
	"errors"
	"strings"
)

func GetRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}

func GetTargetFromRuntimeName(name string) (string, string, error) {
	parts := strings.Split(name, "-")
	if len(parts) != 2 {
		return "", "", errors.New("invalid runtime name")
	}

	return parts[1], parts[0], nil
}
