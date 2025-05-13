package instrumentor

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/report"
	"github.com/odigos-io/odigos/instrumentor/runtimemigration"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
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
		return nil, err
	}

	mgr, err := controllers.CreateManager(opts)
	if err != nil {
		return nil, err
	}

	// This temporary migration step ensures the runtimeDetails migration in the instrumentationConfig is performed.
	// This code can be removed once the migration is confirmed to be successful.
	mgr.Add(&runtimemigration.MigrationRunnable{KubeClient: mgr.GetClient(), Logger: opts.Logger})

	rotatorSetupFinished := make(chan struct{})
	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: env.GetCurrentNamespace(),
			Name:      k8sconsts.InstrumentorWebhookSecretName,
		},
		CertDir: filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"),
		IsReady: rotatorSetupFinished,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to add cert rotator: %w", err)
	}

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

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		select {
		case <-rotatorSetupFinished:
			return nil
		default:
			return fmt.Errorf("cert rotator not ready")
		}
	}); err != nil {
		return nil, fmt.Errorf("unable to set up cert rotator check: %w", err)
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
