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

func GetRelevantEnvVarsKeys() []string {
	keys := make([]string, 0, len(EnvValuesMap))
	for key := range EnvValuesMap {
		keys = append(keys, key)
	}
	return keys
}

// returns the current value that should be populated in a specific environment variable.
// if we should not patch the value, returns nil.
// the are 2 parts to the environment value: odigos part and user part.
// either one can be set or empty.
// so we have 4 cases to handle:
func GetPatchedEnvValue(envName string, observedValue string, currentSdk *common.OtelSdk, language common.ProgrammingLanguage, currentContainerManifestValue *string, annotationOriginalValue *string, annotationOriginalValueFound bool) (*string, *string) {
	envMetadata, ok := EnvValuesMap[envName]
	if !ok {
		// Odigos does not manipulate this environment variable, so ignore it
		return nil, nil
	}

	if envMetadata.programmingLanguage != language {
		// Odigos does not manipulate this environment variable for the given language, so ignore it
		return nil, nil
	}

	if currentSdk == nil {
		// When we have no sdk injected, we should not inject any odigos values.
		return nil, nil
	}

	desiredOdigosPart, ok := envMetadata.values[*currentSdk]
	if !ok {
		// No specific overwrite is required for this SDK
		return nil, nil
	}

	// check if we already processed this env since it's annotation
	if annotationOriginalValueFound {
		if currentContainerManifestValue == nil {
			return &desiredOdigosPart, nil
		}

		currentManifestContainsOdigosValue := strings.Contains(*currentContainerManifestValue, desiredOdigosPart)
		if currentManifestContainsOdigosValue {
			// our goal is to be in the environment. if we are already there, so no overwrite is required.
			// Note 1: assuming that the desired odigos part will not change! (since we are comparing to it).
			// Note 2: if the runtime detected value changed from dockerfile, we will not pick it up here.
			userEnvValue := strings.Replace(*currentContainerManifestValue, envMetadata.delim+desiredOdigosPart, "", -1)
			return currentContainerManifestValue, &userEnvValue
		} else {
			// assuming that the manifest value set by the user is what should be used and ignore runtime details.
			// just add odigos value
			// TODO: update annotation with the new value
			newValue := *currentContainerManifestValue + envMetadata.delim + desiredOdigosPart
			return &newValue, currentContainerManifestValue
		}
	} else {
		// else means there is no annotation on the workload.
		if currentContainerManifestValue != nil {
			userNewValAndOdigos := *currentContainerManifestValue + envMetadata.delim + desiredOdigosPart
			return &userNewValAndOdigos, currentContainerManifestValue
		}
	}

	// scenario 1: no user defined values and no odigos value
	// happens: might be the case right after the source is instrumented, and before the instrumentation is applied.
	// action: there are no user defined values, so no need to make any changes.
	if observedValue == "" {
		return nil, nil
	}

	// scenario 2: no user defined values, only odigos value
	// happens: when the user did not set any value to this env (either via manifest or dockerfile)
	// action: we don't need to overwrite the value, just let odigos handle it
	for _, sdkEnvValue := range envMetadata.values {
		if sdkEnvValue == observedValue {
			return nil, nil
		}
	}

	// temporary fix clean up observed value from the known webhook injected value
	parts := strings.Split(observedValue, envMetadata.delim)
	ignoreEnvValue := "-javaagent:/opt/sre-agent/sre-agent.jar"
	newValues := []string{}
	for _, part := range parts {
		if part == ignoreEnvValue {
			continue
		}
		if strings.TrimSpace(part) == "" {
			continue
		}
		newValues = append(newValues, part)
	}
	observedValue = strings.Join(newValues, envMetadata.delim)

	// Scenario 3: both odigos and user defined values are present
	// happens: when the user set some values to this env (either via manifest or dockerfile) and odigos instrumentation is applied.
	// action: we want to keep the user defined values and upsert the odigos value.
	for _, sdkEnvValue := range envMetadata.values {
		if strings.Contains(observedValue, sdkEnvValue) {
			if sdkEnvValue == desiredOdigosPart {
				// shortcut, the value is already patched
				// both the odigos part equals to the new value, and the user part we want to keep
				// Exception: for a value that is injected by a webhook, we don't want to add it to
				// the deployment, as the webhook will manage when it is needed.
				return &observedValue, currentContainerManifestValue
			} else {
				// The environment variable is patched by some other odigos sdk.
				// replace just the odigos part with the new desired value.
				// this can happen when moving between SDKs.
				patchedEvnValue := strings.ReplaceAll(observedValue, sdkEnvValue, desiredOdigosPart)
				return &patchedEvnValue, currentContainerManifestValue
			}
		}
	}

	// Scenario 4: only user defined values are present
	// happens: when the user set some values to this env (either via manifest or dockerfile) and odigos instrumentation not yet applied.
	// action: we want to keep the user defined values and append the odigos value.
	if observedValue == "" {
		return &desiredOdigosPart, currentContainerManifestValue
	} else {
		// no user defined values, just append the odigos value
		mergedEnvValue := observedValue + envMetadata.delim + desiredOdigosPart
		return &mergedEnvValue, currentContainerManifestValue
	}
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
