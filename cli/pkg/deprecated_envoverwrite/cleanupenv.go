package deprecated_envoverwrite

import (
	"strings"
)

// ========= NOTE =========
// EnvOverwrite is deprecated, odigos will not modify any environment for workload objects (deployments, daemonsets, etc.)
// since Jan 2025, and should revert any changes made to the environment variables for any version upgrade after that.
// The logic for uninstall is kept for some more time, just in case someone is still using
// a very old version of odigos, or that some of the values were not removed during upgrade.
// I find it hard to believe that this code is still needed at this point (June 2025), but just in case...
// This package should be removed once we completely sunset the envoverwrite logic (preferably in odigos 1.1)
// ========= NOTE =========

// due to a bug we had with the env overwriter logic,
// some patched values were recorded incorrectly into the workload annotation for original value.
// they include odigos values (/var/odigos/...) as if they were the original value in the manifest,
// and then used to revert odigos changes back to the original value, which is incorrect and can lead to issues.
// this function sanitizes env values by removing them, and returning a "clean" value back to the user.
// it's a temporary fix since the env overwriter logic is being removed.
// TODO: remove this function in odigos 1.1
func CleanupEnvValueFromOdigosAdditions(envVarName string, envVarValue string) string {

	type envOverwriteMetadata struct {
		delim                string
		possibleOdigosValues []string
	}

	nodeMetadata := envOverwriteMetadata{
		delim: " ",
		possibleOdigosValues: []string{
			"--require /var/odigos/nodejs/autoinstrumentation.js",
			"--require /var/odigos/nodejs-ebpf/autoinstrumentation.js",
		},
	}
	pythonMetadata := envOverwriteMetadata{
		delim: ":",
		possibleOdigosValues: []string{
			"/var/odigos/python",
			"/var/odigos/python/opentelemetry/instrumentation/auto_instrumentation",
			"/var/odigos/python-ebpf",
		},
	}
	javaMetadata := envOverwriteMetadata{
		delim: " ",
		possibleOdigosValues: []string{"-javaagent:/var/odigos/java/javaagent.jar",
			"-javaagent:/var/odigos/java-ebpf/dtrace-injector.jar",
			"-javaagent:/var/odigos/java-ext-ebpf/javaagent.jar",
			"-Dotel.javaagent.extensions=/var/odigos/java-ext-ebpf/otel_agent_extension.jar",
		},
	}

	envToMetadataMap := map[string]envOverwriteMetadata{
		"NODE_OPTIONS":      nodeMetadata,
		"PYTHONPATH":        pythonMetadata,
		"JAVA_OPTS":         javaMetadata,
		"JAVA_TOOL_OPTIONS": javaMetadata,
	}

	overwriteMetadata, exists := envToMetadataMap[envVarName]
	if !exists {
		// not managed by odigos, so no need to clean up
		// not expected to happen, but just in case
		return envVarValue
	}

	// if any of the possible values for this env exists, remove it
	for _, value := range overwriteMetadata.possibleOdigosValues {
		// try to remove each value with and without the delimiter.
		// if odigos value is the only one left, the delimiter will not be present.
		withSeparator := overwriteMetadata.delim + value
		envVarValue = strings.ReplaceAll(envVarValue, withSeparator, "")
		envVarValue = strings.ReplaceAll(envVarValue, value, "")
	}

	return envVarValue
}
