package envOverwrite

import (
	"strings"

	"github.com/odigos-io/odigos/common"
)

type envValues struct {
	delim               string
	programmingLanguage common.ProgrammingLanguage
	values              map[common.OtelSdk]string
}

// EnvValuesMap is a map of environment variables odigos uses for various languages and goals.
// The key is the environment variable name and the value is the value to be set or appended
// to the environment variable. We need to make sure that in case any of these environment
// variables is already set, we append the value to it instead of overwriting it.
//
// Note: The values here needs to be in sync with the paths used in the odigos images.
// If the paths are changed in the odigos images, the values here should be updated accordingly.
var EnvValuesMap = map[string]envValues{
	"NODE_OPTIONS": {
		delim:               " ",
		programmingLanguage: common.JavascriptProgrammingLanguage,
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "--require /var/odigos/nodejs/autoinstrumentation.js",
			common.OtelSdkEbpfEnterprise:  "--require /var/odigos/nodejs-ebpf/autoinstrumentation.js",
		},
	},
	"PYTHONPATH": {
		delim:               ":",
		programmingLanguage: common.PythonProgrammingLanguage,
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "/var/odigos/python:/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation",
			common.OtelSdkEbpfEnterprise:  "/var/odigos/python-ebpf:/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation:/var/odigos/python",
		},
	},
	"JAVA_OPTS": {
		delim:               " ",
		programmingLanguage: common.JavaProgrammingLanguage,
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "-javaagent:/var/odigos/java/javaagent.jar",
			common.OtelSdkEbpfEnterprise:  "-javaagent:/var/odigos/java-ebpf/dtrace-injector.jar",
			common.OtelSdkNativeEnterprise: "-javaagent:/var/odigos/java-ext-ebpf/javaagent.jar " +
				"-Dotel.javaagent.extensions=/var/odigos/java-ext-ebpf/otel_agent_extension.jar",
		},
	},
	"JAVA_TOOL_OPTIONS": {
		delim:               " ",
		programmingLanguage: common.JavaProgrammingLanguage,
		values: map[common.OtelSdk]string{
			common.OtelSdkNativeCommunity: "-javaagent:/var/odigos/java/javaagent.jar",
			common.OtelSdkEbpfEnterprise:  "-javaagent:/var/odigos/java-ebpf/dtrace-injector.jar",
			common.OtelSdkNativeEnterprise: "-javaagent:/var/odigos/java-ext-ebpf/javaagent.jar " +
				"-Dotel.javaagent.extensions=/var/odigos/java-ext-ebpf/otel_agent_extension.jar",
		},
	},
}

// EnvVarsForLanguage is a map of environment variables that are relevant for each language.
var EnvVarsForLanguage = map[common.ProgrammingLanguage][]string{
	common.JavascriptProgrammingLanguage: {"NODE_OPTIONS"},
	common.PythonProgrammingLanguage:     {"PYTHONPATH"},
	common.JavaProgrammingLanguage:       {"JAVA_OPTS", "JAVA_TOOL_OPTIONS"},
}

func GetRelevantEnvVarsKeys() []string {
	keys := make([]string, 0, len(EnvValuesMap))
	for key := range EnvValuesMap {
		keys = append(keys, key)
	}
	return keys
}

func GetPossibleValuesPerEnv(env string) map[common.OtelSdk]string {
	return EnvValuesMap[env].values
}

// due to a bug we had with the env overwriter logic,
// some patched values were recorded incorrectly into the workload annotation for original value.
// they include odigos values (/var/odigos/...) as if they were the original value in the manifest,
// and then used to revert odigos changes back to the original value, which is incorrect and can lead to issues.
// this function sanitizes env values by removing them, and returning a "clean" value back to the user.
// it's a temporary fix since the env overwriter logic is being removed.
// TODO: remove this function in odigos 1.1
func CleanupEnvValueFromOdigosAdditions(envVarName string, envVarValue string) string {
	overwriteMetadata, exists := EnvValuesMap[envVarName]
	if !exists {
		// not managed by odigos, so no need to clean up
		// not expected to happen, but just in case
		return envVarValue
	}

	// if any of the possible values for this env exists, remove it
	for _, value := range overwriteMetadata.values {
		withSeparator := overwriteMetadata.delim + value
		envVarValue = strings.Replace(envVarValue, withSeparator, "", -1)
	}

	// remove any odigos special values if they exist
	if envVarName == "JAVA_OPTS" || envVarName == "JAVA_TOOL_OPTIONS" {
		envVarValue = strings.Replace(envVarValue, "-javaagent:/opt/sre-agent/sre-agent.jar", "", -1)
		envVarValue = strings.Replace(envVarValue, "newrelic/bootstrap", "", -1)
	}

	return envVarValue
}

func AppendOdigosAdditionsToEnvVar(envName string, envFromContainerRuntimeValue string, desiredOdigosAddition string) *string {
	envValues, ok := EnvValuesMap[envName]
	if !ok {
		// Odigos does not manipulate this environment variable, so ignore it
		return nil
	}

	// In case observedValue is exists but empty, we just need to set the desiredOdigosAddition without delim before
	if strings.TrimSpace(envFromContainerRuntimeValue) == "" {
		return &desiredOdigosAddition
	} else {
		// In case observedValue is not empty, we need to append the desiredOdigosAddition with the delim
		mergedEnvValue := envFromContainerRuntimeValue + envValues.delim + desiredOdigosAddition
		return &mergedEnvValue
	}
}
