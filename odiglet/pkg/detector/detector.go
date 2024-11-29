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

const (
	ProcessExecEvent = detector.ProcessExecEvent
	ProcessExitEvent = detector.ProcessExitEvent
)

type Detector struct {
	Detector *detector.Detector
	procEvents chan<- ProcessEvent
}

func NewDetector(ctx context.Context, logger logr.Logger, events chan<- ProcessEvent) (*Detector, error) {
	detector, err := newDetector(ctx, logger, events)
	if err != nil {
		return nil, err
	}

	return &Detector{
		Detector: detector,
		procEvents: events,
	}, nil
}

func (d *Detector) Run(ctx context.Context) error {
	var runError error
	done := make(chan struct{})

	go func() {
		defer close(done)
		runError = d.Detector.Run(ctx)
	}()

	<-done
	return runError
}

func newDetector(ctx context.Context, logger logr.Logger, events chan<- ProcessEvent) (*detector.Detector, error) {
	sLogger := slog.New(logr.ToSlogHandler(logger))

	opts := []detector.DetectorOption{
		detector.WithLogger(sLogger),
		detector.WithEnvironments(relevantEnvVars()...),
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
