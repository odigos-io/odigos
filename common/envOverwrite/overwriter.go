package envOverwrite

import "strings"

// EnvValues is a map of environment variables odigos uses for various languages and goals.
// The key is the environment variable name and the value is the value to be set or appended
// to the environment variable. We need to make sure that in case any of these environment
// variables is already set, we append the value to it instead of overwriting it.
var EnvValues = map[string]struct{
	Value string
	Delim string
}{
	"NODE_OPTIONS": {
		Value: "--require /var/odigos/nodejs/autoinstrumentation.js",
		Delim: " ",
	},
	"PYTHONPATH": {
		Value: "/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation:/var/odigos/python",
		Delim: ":",
	},
}

func ShouldPatch(envName string, value string) bool {
	_, ok := EnvValues[envName]
	if !ok {
		// We don't care about this environment variable
		return false
	}

	if strings.Contains(value, EnvValues[envName].Value) {
		// The environment variable is already patched
		return false
	}
	return true
}

func ShouldRevert(envName string, value string) bool {
	_, ok := EnvValues[envName]
	if !ok {
		// We don't care about this environment variable
		return false
	}

	if !strings.Contains(value, EnvValues[envName].Value) {
		// The environment variable is not patched
		return false
	}
	return true

}

func Patch(envName string, currentVal string) string {
	env, exists := EnvValues[envName]
	if !exists {
		return ""
	}

	if currentVal == "" {
		return env.Value
	}

	return currentVal + env.Delim + env.Value
}
