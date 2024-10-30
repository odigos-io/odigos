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

// returns the current value that should be populated in a specific environment variable.
// if we should not patch the value, returns nil.
// the are 2 parts to the environment value: odigos part and user part.
// either one can be set or empty.
// so we have 4 cases to handle:
func GetPatchedEnvValue(envName string, observedValue string, currentSdk *common.OtelSdk, language common.ProgrammingLanguage) *string {
	envMetadata, ok := EnvValuesMap[envName]
	if !ok {
		// Odigos does not manipulate this environment variable, so ignore it
		return nil
	}

	if envMetadata.programmingLanguage != language {
		// Odigos does not manipulate this environment variable for the given language, so ignore it
		return nil
	}

	if currentSdk == nil {
		// When we have no sdk injected, we should not inject any odigos values.
		return nil
	}

	desiredOdigosPart, ok := envMetadata.values[*currentSdk]
	if !ok {
		// No specific overwrite is required for this SDK
		return nil
	}

	// scenario 1: no user defined values and no odigos value
	// happens: might be the case right after the source is instrumented, and before the instrumentation is applied.
	// action: there are no user defined values, so no need to make any changes.
	if observedValue == "" {
		return nil
	}

	// scenario 2: no user defined values, only odigos value
	// happens: when the user did not set any value to this env (either via manifest or dockerfile)
	// action: we don't need to overwrite the value, just let odigos handle it
	for _, sdkEnvValue := range envMetadata.values {
		if sdkEnvValue == observedValue {
			return nil
		}
	}

	// Scenario 3: both odigos and user defined values are present
	// happens: when the user set some values to this env (either via manifest or dockerfile) and odigos instrumentation is applied.
	// action: we want to keep the user defined values and upsert the odigos value.
	for _, sdkEnvValue := range envMetadata.values {
		if strings.Contains(observedValue, sdkEnvValue) {
			if sdkEnvValue == desiredOdigosPart {
				// shortcut, the value is already patched
				// both the odigos part equals to the new value, and the user part we want to keep
				// Exception: if there is a webhook involved that inject the env value,
				// we need to remove duplicate values, otherwise it will grow indefinitely in each iteration
				parts := strings.Split(observedValue, envMetadata.delim)
				specialEnvValue := "-javaagent:/opt/sre-agent/sre-agent.jar"
				specialFound := false
				newValues := []string{}
				for _, part := range parts {
					if part == specialEnvValue {
						if specialFound {
							continue
						}
						specialFound = true
					}
					if part == "" {
						continue
					}
					newValues = append(newValues, part)
				}
				observedValue = strings.Join(newValues, envMetadata.delim)
				return &observedValue
			} else {
				// The environment variable is patched by some other odigos sdk.
				// replace just the odigos part with the new desired value.
				// this can happen when moving between SDKs.
				patchedEvnValue := strings.ReplaceAll(observedValue, sdkEnvValue, desiredOdigosPart)
				return &patchedEvnValue
			}
		}
	}

	// Scenario 4: only user defined values are present
	// happens: when the user set some values to this env (either via manifest or dockerfile) and odigos instrumentation not yet applied.
	// action: we want to keep the user defined values and append the odigos value.
	mergedEnvValue := observedValue + envMetadata.delim + desiredOdigosPart
	return &mergedEnvValue
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
