package distro

// Givin a set of distros, return a map of all the names of the environment variables
// that participate in append mechanism (PYTHONPATH, JAVA_TOOL_OPTIONS, NODE_OPTIONS, etc.)
// This is used at runtime detection and for enterprise loader to filter relevant env vars
// for further patching downstream.
func GetAppendEnvVarNames(distros []*OtelDistro) map[string]struct{} {
	appendEnvVarNames := map[string]struct{}{}
	for _, distro := range distros {
		if distro.RuntimeAgent != nil {
			for _, envVar := range distro.RuntimeAgent.EnvironmentVariables {
				envName := envVar.EnvName
				appendEnvVarNames[envName] = struct{}{}
			}
		}
	}
	return appendEnvVarNames
}
