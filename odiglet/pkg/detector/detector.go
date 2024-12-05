package detector

import (
	"context"
	"log/slog"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"github.com/odigos-io/runtime-detector"
)

type ProcessEvent = detector.ProcessEvent

type Detector = detector.Detector

const (
	ProcessExecEvent = detector.ProcessExecEvent
	ProcessExitEvent = detector.ProcessExitEvent
)

func NewK8SProcDetector(ctx context.Context, logger logr.Logger, events chan<- ProcessEvent) (*detector.Detector, error) {
	sLogger := slog.New(logr.ToSlogHandler(logger))

	opts := []detector.DetectorOption{
		detector.WithLogger(sLogger),
		detector.WithEnvironments(relevantEnvVars()...),
		detector.WithEnvPrefixFilter(consts.OdigosEnvVarPodName),
	}
	detector, err := detector.NewDetector(ctx, events, opts...)

	if err != nil {
		return nil, err
	}

	return detector, nil
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
