package envOverwrite

import "strings"

var envValues = map[string]string{
	"NODE_OPTIONS": "--require /var/odigos/nodejs/autoinstrumentation.js",
}

func ShouldOverwrite(envName string) bool {
	_, ok := envValues[envName]
	return ok
}

func Patch(envName string, currentVal string) string {
	val, exists := envValues[envName]
	if !exists {
		return ""
	}

	if currentVal == "" {
		return val
	}

	return currentVal + " " + val
}

func Revert(envName string, currentVal string) string {
	val, exists := envValues[envName]
	if !exists {
		return ""
	}

	return strings.Replace(currentVal, " "+val, "", 1)
}
