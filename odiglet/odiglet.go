package odiglet

import (
	"context"
	"fmt"

	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/odigos-io/odigos/common"
	k8senv "github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/kube"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"golang.org/x/sync/errgroup"

	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime"
)

type Odiglet struct {
	clientset     *kubernetes.Clientset
	mgr           controllerruntime.Manager
	ebpfManager   commonInstrumentation.Manager
	configUpdates chan<- commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	lsf           instrumentation.OtelSdksLsf
}

const (
	configUpdatesBufferSize = 10
)

// New creates a new Odiglet instance.
func New(lsf instrumentation.OtelSdksLsf, factories map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory) (*Odiglet, error) {
	// Init Kubernetes API client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config for Kubernetes client %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client %w", err)
	}

	mgr, err := kube.CreateManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime manager %w", err)
	}

	configUpdates := make(chan commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup], configUpdatesBufferSize)
	ebpfManager, err := ebpf.NewManager(mgr.GetClient(), log.Logger, factories, configUpdates)
	if err != nil {
		return nil, fmt.Errorf("failed to create ebpf manager %w", err)
	}

	err = kube.SetupWithManager(mgr, nil, clientset, configUpdates)
	if err != nil {
		return nil, fmt.Errorf("failed to setup controller-runtime manager %w", err)
	}

	return &Odiglet{
		clientset:     clientset,
		mgr:           mgr,
		ebpfManager:   ebpfManager,
		configUpdates: configUpdates,
		lsf:           lsf,
	}, nil
}

// Run starts the Odiglet components and blocks until the context is cancelled, or a critical error occurs.
func (o *Odiglet) Run(ctx context.Context) {
	g, groupCtx := errgroup.WithContext(ctx)

	// Start pprof server
	g.Go(func() error {
		err := common.StartPprofServer(groupCtx, log.Logger)
		if err != nil {
			log.Logger.Error(err, "Failed to start pprof server")
		} else {
			log.Logger.V(0).Info("Pprof server exited")
		}
		// if we fail to start the pprof server, don't return an error as it is not critical
		// and we can run the rest of the components
		return nil
	})

	// Start device manager
	// the device manager library doesn't support passing a context,
	// however, internally it uses a context to cancel the device manager once SIGTERM or SIGINT is received.
	g.Go(func() error {
		err := runDeviceManager(o.clientset, o.lsf)
		log.Logger.V(0).Info("Device manager exited")
		return err
	})

	g.Go(func() error {
		err := o.ebpfManager.Run(ctx)
		if err != nil {
			log.Logger.Error(err, "Failed to run ebpf manager")
		}
		log.Logger.V(0).Info("eBPF manager exited")
		return err
	})

	// start OpAmp server
	odigosNs := k8senv.GetCurrentNamespace()
	g.Go(func() error {
		err := server.StartOpAmpServer(ctx, log.Logger, o.mgr, o.clientset, env.Current.NodeName, odigosNs)
		if err != nil {
			log.Logger.Error(err, "Failed to start opamp server")
		}
		log.Logger.V(0).Info("OpAmp server exited")
		return err
	})

	// start kube manager
	g.Go(func() error {
		err := o.mgr.Start(ctx)
		if err != nil {
			log.Logger.Error(err, "error starting kube manager")
		} else {
			log.Logger.V(0).Info("Kube manager exited")
		}
		// the manager is stopped, it is now safe to close the config updates channel
		if o.configUpdates != nil {
			close(o.configUpdates)
		}
		return err
	})

	err := g.Wait()
	if err != nil {
		log.Logger.Error(err, "Odiglet exited with error")
	}
}

func runDeviceManager(clientset *kubernetes.Clientset, otelSdkLsf instrumentation.OtelSdksLsf) error {
	log.Logger.V(0).Info("Starting device manager")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lister, err := instrumentation.NewLister(ctx, clientset, otelSdkLsf)
	if err != nil {
		return fmt.Errorf("failed to create device manager lister %w", err)
	}

	manager := dpm.NewManager(lister)
	manager.Run()
	return nil
}