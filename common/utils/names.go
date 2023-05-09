package utils

import "strings"

func GetRuntimeObjectName(name string, kind string) string {
	return strings.ToLower(kind + "-" + name)
}
