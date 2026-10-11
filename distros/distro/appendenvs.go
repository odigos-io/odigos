package distro

import "strings"

const OriginalEnvValuePlaceholder = "{{ORIGINAL_ENV_VALUE}}"

// Givin a set of distros, return a map of all the names of the environment variables
// that participate in append mechanism (PYTHONPATH, JAVA_TOOL_OPTIONS, NODE_OPTIONS, etc.)
// This is used at runtime detection and for enterprise loader to filter relevant env vars
// for further patching downstream.
func GetAppendEnvVarNames(distros []*OtelDistro) map[string]struct{} {
	appendEnvVarNames := map[string]struct{}{}
	for _, distro := range distros {
		for _, envVar := range distro.EnvironmentVariables.AppendOdigosVariables {
			envName := envVar.EnvName
			appendEnvVarNames[envName] = struct{}{}
		}
	}
	return appendEnvVarNames
}

// EvaluateReplacePattern resolves an AppendOdigosEnvironmentVariable ReplacePattern into a
// concrete env var value by substituting the two well-known placeholders:
//   - {{ODIGOS_AGENTS_DIR}} → agentsDir  (e.g. "/var/odigos" on k8s)
//   - {{ORIGINAL_ENV_VALUE}} → originalValue (the pre-existing value of the env var)
//
// When originalValue is empty the placeholder and its adjacent delimiter (space or colon)
// are stripped so the result never starts with a stray delimiter character.
func EvaluateReplacePattern(pattern, originalValue, agentsDir string) string {
	result := strings.ReplaceAll(pattern, AgentPlaceholderDirectory, agentsDir)
	if strings.TrimSpace(originalValue) == "" {
		result = strings.ReplaceAll(result, OriginalEnvValuePlaceholder, "")
		result = strings.TrimLeft(result, " :")
	} else {
		result = strings.ReplaceAll(result, OriginalEnvValuePlaceholder, originalValue)
	}
	return result
}
