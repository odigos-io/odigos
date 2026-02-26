package detector

import (
	"os"
	"strconv"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	detector "github.com/odigos-io/runtime-detector"
)

const (
	durationFilterMillisEnvKey = "ODIGOS_PROCESS_DURATION_FILTER_MILLIS"
)

func DefaultK8sDetectorOptions(appendEnvVarNames []string) []detector.DetectorOption {
	logger := commonlogger.Logger().With("subsystem", "detector")

	opts := []detector.DetectorOption{
		detector.WithLogger(logger),
		detector.WithEnvironments(relevantEnvVars(appendEnvVarNames)...),
		detector.WithEnvPrefixFilter(k8sconsts.OdigosEnvVarPodName),
		detector.WithExePathsToFilter(
			"/usr/bin/bash",
			"/bin/bash",
			"/usr/local/sbin/bash",
			"/usr/local/bin/bash",
			"/bin/sh",
			"/usr/bin/sh",
			"/bin/busybox",
			"/usr/bin/dash",
			"/sbin/tini",
			"/usr/bin/tini",
		),
		detector.WithProcFSPath(process.HostProcDir()),
	}

	if val, ok := os.LookupEnv(durationFilterMillisEnvKey); ok {
		valI, err := strconv.Atoi(val)
		if err != nil {
			logger.Error("Failed to parse ODIGOS_PROCESS_DURATION_FILTER_MILLIS env var, ignoring", "err", err, "value", val)
			return opts
		}

		d := time.Duration(valI) * time.Millisecond
		logger.Info("Using duration filter from env var", "value", d)
		opts = append(opts, detector.WithMinDuration(d))
	}

	return opts
}

func relevantEnvVars(appendEnvVarNames []string) []string {
	// env vars related to language versions
	versionEnvs := process.LangsVersionEnvs

	envs := make([]string, 0, len(versionEnvs))
	for env := range versionEnvs {
		envs = append(envs, env)
	}

	// env vars that Odigos is using for adding dependencies
	envs = append(envs, appendEnvVarNames...)

	// env vars that Odigos is injecting to the relevant containers
	envs = append(envs, k8sconsts.OdigosInjectedEnvVars()...)

	return envs
}
