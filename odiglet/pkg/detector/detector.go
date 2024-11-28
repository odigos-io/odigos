package detector

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common/envOverwrite"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"github.com/odigos-io/runtime-detector"
)

type ProcessEvent = detector.ProcessEvent

func StartRuntimeDetector(ctx context.Context, logger logr.Logger, events chan ProcessEvent) error {
	detector, err := newDetector(ctx, logger, events)
	if err != nil {
		return fmt.Errorf("failed to create runtime detector: %w", err)
	}

	var wg sync.WaitGroup
	var runError error

	wg.Add(1)
	go func() {
		defer wg.Done()
		readProcEventsLoop(logger, events)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		runError = detector.Run(ctx)
	}()

	wg.Wait()
	return runError
}

func newDetector(ctx context.Context, logger logr.Logger, events chan ProcessEvent) (*detector.Detector, error) {
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

func readProcEventsLoop(l logr.Logger, events chan ProcessEvent) {
	l = l.WithName("process detector")
	for e := range events {
		switch e.EventType {
		case detector.ProcessExecEvent:
			l.Info("detected new process",
				"pid", e.PID,
				"cmd", e.ExecDetails.CmdLine,
				"exeName", e.ExecDetails.ExeName,
				"exeLink", e.ExecDetails.ExeLink,
				"envs", e.ExecDetails.Environments,
				"container PID", e.ExecDetails.ContainerProcessID,
			)
		case detector.ProcessExitEvent:
			l.Info("detected process exit",
				"pid", e.PID,
			)
		}
	}
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
