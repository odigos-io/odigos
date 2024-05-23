package envOverwrite

import (
	"strings"

	"github.com/odigos-io/odigos/common"
)

type envValues struct {
	delim string
	values map[common.OtelSdk]string
}
// EnvValuesMap is a map of environment variables odigos uses for various languages and goals.
// The key is the environment variable name and the value is the value to be set or appended
// to the environment variable. We need to make sure that in case any of these environment
// variables is already set, we append the value to it instead of overwriting it.
var EnvValuesMap = map[string]envValues{
	"NODE_OPTIONS": {
		delim: " ",
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "--require /var/odigos/nodejs/autoinstrumentation.js",
			common.OtelSdkEbpfEnterprise: "--require /var/odigos/nodejs-ebpf/autoinstrumentation.js",
		},
	},
	"PYTHONPATH": {
		delim: ":",
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "/var/odigos/python:/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation",
			common.OtelSdkEbpfEnterprise: "/var/odigos/python-ebpf:/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation:/var/odigos/python",
		},
	},
	"JAVA_OPTS": {
		delim: " ",
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "-javaagent:/var/odigos/java/javaagent.jar",
			common.OtelSdkEbpfEnterprise: "-javaagent:/var/odigos/java-ebpf/dtrace-injector.jar",
			common.OtelSdkNativeEnterprise: "-javaagent:/var/odigos/java-ext-ebpf/javaagent.jar " +
			"-Dotel.javaagent.extensions=/var/odigos/java-ext-ebpf/otel_agent_extension.jar",
		},
	},
	"JAVA_TOOL_OPTIONS": {
		delim: " ",
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "-javaagent:/var/odigos/java/javaagent.jar",
			common.OtelSdkEbpfEnterprise: "-javaagent:/var/odigos/java-ebpf/dtrace-injector.jar",
			common.OtelSdkNativeEnterprise: "-javaagent:/var/odigos/java-ext-ebpf/javaagent.jar " +
			"-Dotel.javaagent.extensions=/var/odigos/java-ext-ebpf/otel_agent_extension.jar",
		},
	},
}

func ShouldPatch(envName string, observedValue string, sdk common.OtelSdk) bool {
	env, ok := EnvValuesMap[envName]
	if !ok {
		// Odigos does not manipulate this environment variable, so ignore it
		return false
	}

	val, ok := env.values[sdk]
	if !ok {
		// Odigos does not manipulate this environment variable for this SDK, so ignore it
		return false
	}

	if val == observedValue {
		// if the observed value is the same as the value odigos sets,
		// that means the user does not add any additional values,
		// so we should not add it to the deployment manifest
		return false
	}

	return true
}

func ShouldRevert(envName string, value string, sdk common.OtelSdk) bool {
	env, ok := EnvValuesMap[envName]
	if !ok {
		// We don't care about this environment variable
		return false
	}

	val, ok := env.values[sdk]
	if !ok {
		// We don't care about this environment variable for this SDK
		return false
	}

	if !strings.Contains(value, val) {
		// The environment variable is not patched
		return false
	}
	return true

}

func Patch(envName string, currentVal string, sdk common.OtelSdk) string {
	env, exists := EnvValuesMap[envName]
	if !exists {
		return ""
	}

	valToAppend, ok := env.values[sdk]
	if !ok {
		return ""
	}

	if currentVal == "" {
		return valToAppend
	}

	if strings.Contains(currentVal, valToAppend) {
		// The environment variable is already patched
		return currentVal
	}

	return currentVal + env.delim + valToAppend
}

func ValToAppend(envName string, sdk common.OtelSdk) (string, bool) {
	env, exists := EnvValuesMap[envName]
	if !exists {
		return "", false
	}

	valToAppend, ok := env.values[sdk]
	if !ok {
		return "", false
	}

	return valToAppend, true
}
