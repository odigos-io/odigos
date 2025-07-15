package instrumentor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	"github.com/odigos-io/odigos/instrumentor/report"
	"github.com/odigos-io/odigos/k8sutils/pkg/certs"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Instrumentor struct {
	mgr                controllerruntime.Manager
	logger             logr.Logger
	certReady          chan struct{}
	dp                 *distros.Provider
	webhooksRegistered *atomic.Bool
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

	// remove the deprecated webhook secret if it exists
	mgr.Add(&certs.SecretDeleteMigration{Client: mgr.GetClient(), Logger: opts.Logger, Secret: types.NamespacedName{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.DeprecatedInstrumentorWebhookSecretName,
	}})

	// setup the certificate rotator
	rotatorSetupFinished := make(chan struct{})
	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: env.GetCurrentNamespace(),
			Name:      k8sconsts.InstrumentorWebhookSecretName,
		},
		CertDir: filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"),
		IsReady: rotatorSetupFinished,
		CAName:  k8sconsts.InstrumentorCAName,
		Webhooks: []rotator.WebhookInfo{
			{Name: k8sconsts.InstrumentorMutatingWebhookName, Type: rotator.Mutating},
			{Name: k8sconsts.InstrumentorSourceMutatingWebhookName, Type: rotator.Mutating},
			{Name: k8sconsts.InstrumentorSourceValidatingWebhookName, Type: rotator.Validating},
		},
		DNSName: "serving-cert",
		ExtraDNSNames: []string{
			fmt.Sprintf("%s.%s.svc", k8sconsts.InstrumentorServiceName, env.GetCurrentNamespace()),
			fmt.Sprintf("%s.%s.svc.cluster.local", k8sconsts.InstrumentorServiceName, env.GetCurrentNamespace()),
		},
		EnableReadinessCheck: true,

		// marking the controller as the owner of the webhooks config updated fields (caBundle)
		// this helps to avoid CI/CD systems overwriting the controller set fields.
		FieldOwner: k8sconsts.InstrumentorWebhookFieldOwner,

		// we could set RequireLeaderElection to true here but that will make the readiness probe fail for non-leader
		// instances (since the IsReady channel will not be closed in non-leader instances).

		// these are the defaults, but we set them explicitly for clarity
		CaCertDuration:         10 * 365 * 24 * time.Hour, // 10 years
		ServerCertDuration:     1 * 365 * 24 * time.Hour,  // 1 year
		RotationCheckFrequency: 12 * time.Hour,            // 12 hours
		LookaheadInterval:      90 * 24 * time.Hour,       // 90 days
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

	webhooksRegistered := &atomic.Bool{}
	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		if !webhooksRegistered.Load() {
			return errors.New("webhooks not registered yet")
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to set up cert rotator check: %w", err)
	}

	return &Instrumentor{
		mgr:                mgr,
		logger:             opts.Logger,
		certReady:          rotatorSetupFinished,
		dp:                 dp,
		webhooksRegistered: webhooksRegistered,
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

	// register webhooks after the certificate is ready
	g.Go(func() error {
		select {
		case <-i.certReady:
		case <-groupCtx.Done():
			return nil
		}
		i.logger.V(0).Info("Cert rotator is ready")
		err := controllers.RegisterWebhooks(i.mgr, i.dp)
		if err != nil {
			return err
		}
		i.webhooksRegistered.Store(true)
		i.logger.V(0).Info("Webhooks registered")
		return nil
	})

	err := g.Wait()
	if err != nil {
		i.logger.Error(err, "Instrumentor exited with error")
	}
}
