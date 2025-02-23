package detector

import (
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	detector "github.com/odigos-io/runtime-detector"
)

func DefaultK8sDetectorOptions(logger logr.Logger) []detector.DetectorOption {
	sLogger := slog.New(logr.ToSlogHandler(logger))

	opts := []detector.DetectorOption{
		detector.WithLogger(sLogger),
		detector.WithEnvironments(relevantEnvVars()...),
		detector.WithEnvPrefixFilter(k8sconsts.OdigosEnvVarPodName),
		detector.WithExePathsToFilter("/usr/bin/bash", "/bin/bash", "/bin/sh", "/usr/bin/sh", "/bin/busybox", "/usr/bin/dash", "/sbin/tini", "/usr/bin/tini"),
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
	envs = append(envs, k8sconsts.OdigosInjectedEnvVars()...)

	return envs
}
