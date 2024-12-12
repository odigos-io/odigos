package detector

import (
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"github.com/odigos-io/runtime-detector"
)

func K8sDetectorOptions(logger logr.Logger) []detector.DetectorOption {
	sLogger := slog.New(logr.ToSlogHandler(logger))

	opts := []detector.DetectorOption{
		detector.WithLogger(sLogger),
		detector.WithEnvironments(relevantEnvVars()...),
		detector.WithEnvPrefixFilter(consts.OdigosEnvVarPodName),
	}

	return opts
}

func relevantEnvVars() []string {
	// env vars related to language versions
	versionEnvs := process.LangsVersionEnvs

	envs := make([]string, 0, len(versionEnvs))
	for env := range versionEnvs {
		envs = append(envs, env)
	}

	// env vars that Odigos is using for adding dependencies
	envs = append(envs, envOverwrite.GetRelevantEnvVarsKeys()...)

	// env vars that Odigos is injecting to the relevant containers
	envs = append(envs, consts.OdigosInjectedEnvVars()...)

	return envs
}
