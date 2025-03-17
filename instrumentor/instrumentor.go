package instrumentor

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/report"
	"github.com/odigos-io/odigos/instrumentor/runtimemigration"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"golang.org/x/sync/errgroup"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Instrumentor struct {
	mgr    controllerruntime.Manager
	logger logr.Logger
}

func New(opts controllers.KubeManagerOptions, dp *distros.Provider) (*Instrumentor, error) {
	err := feature.Setup()
	if err != nil {
		fmt.Println("Setup failed")
		return nil, err
	}
	fmt.Println("Setup successful")

	mgr, err := controllers.CreateManager(opts)
	if err != nil {
		return nil, err
	}

	// This temporary migration step ensures the runtimeDetails migration in the instrumentationConfig is performed.
	// This code can be removed once the migration is confirmed to be successful.
	mgr.Add(&runtimemigration.MigrationRunnable{KubeClient: mgr.GetClient(), Logger: opts.Logger})

	// wire up the controllers and webhooks
	err = controllers.SetupWithManager(mgr, dp)
	if err != nil {
		return nil, err
	}

	// Add health and ready probes
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return mgr.GetWebhookServer().StartedChecker()(req)
	}); err != nil {
		return nil, fmt.Errorf("unable to set up ready check: %w", err)
	}

	return &Instrumentor{
		mgr:    mgr,
		logger: opts.Logger,
	}, nil
}

func (i *Instrumentor) Run(ctx context.Context, odigosTelemetryDisabled bool) {
	g, groupCtx := errgroup.WithContext(ctx)

	// Start pprof server
	g.Go(func() error {
		err := common.StartPprofServer(groupCtx, i.logger)
		if err != nil {
			i.logger.Error(err, "Failed to start pprof server")
		} else {
			i.logger.V(0).Info("Pprof server exited")
		}
		// if we fail to start the pprof server, don't return an error as it is not critical
		// and we can run the rest of the components
		return nil
	})

	if !odigosTelemetryDisabled {
		// Start telemetry report
		g.Go(func() error {
			report.Start(groupCtx, i.mgr.GetClient())
			i.logger.V(0).Info("Telemetry reporting exited")
			return nil
		})
	}

	// start kube manager
	g.Go(func() error {
		err := i.mgr.Start(groupCtx)
		if err != nil {
			i.logger.Error(err, "error starting kube manager")
		} else {
			i.logger.V(0).Info("Kube manager exited")
		}
		return err
	})

	err := g.Wait()
	if err != nil {
		i.logger.Error(err, "Instrumentor exited with error")
	}
}
