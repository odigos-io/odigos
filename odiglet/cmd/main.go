package main

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/fs"

	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/odigos-io/odigos/common"
	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	k8senv "github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/instrumentlang"
	"github.com/odigos-io/odigos/odiglet/pkg/kube"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	_ "net/http/pprof"
)

func odigletInitPhase() {
	if err := log.Init(); err != nil {
		panic(err)
	}
	err := fs.CopyAgentsDirectoryToHost()
	if err != nil {
		log.Logger.Error(err, "Failed to copy agents directory to host")
		os.Exit(-1)
	}
	os.Exit(0)
}

type odiglet struct {
	clientset     *kubernetes.Clientset
	mgr           ctrl.Manager
	ebpfManager   commonInstrumentation.Manager
	configUpdates chan<- commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	criWrapper    *criwrapper.CriClient
}

const (
	configUpdatesBufferSize = 10
)

func newOdiglet() (*odiglet, error) {
	// Init Kubernetes API client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to create in-cluster config for Kubernetes client %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to create Kubernetes client %w", err)
	}

	criWrapper := criwrapper.CriClient{Logger: log.Logger}

	mgr, err := kube.CreateManager()
	if err != nil {
		return nil, fmt.Errorf("Failed to create controller-runtime manager %w", err)
	}

	configUpdates := make(chan commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup], configUpdatesBufferSize)
	ebpfManager, err := ebpf.NewManager(
		mgr.GetClient(),
		log.Logger,
		map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory{
			commonInstrumentation.OtelDistribution{
				Language: common.GoProgrammingLanguage,
				OtelSdk:  common.OtelSdkEbpfCommunity,
			}: sdks.NewGoInstrumentationFactory(),
		},
		configUpdates,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create ebpf manager %w", err)
	}

	kubeManagerOptions := kube.KubeManagerOptions{
		Mgr:           mgr,
		EbpfDirectors: nil,
		Clientset:     clientset,
		ConfigUpdates: configUpdates,
		CriClient:     &criWrapper,
	}

	err = kube.SetupWithManager(kubeManagerOptions)
	if err != nil {
		return nil, fmt.Errorf("Failed to setup controller-runtime manager %w", err)
	}

	return &odiglet{
		clientset:     clientset,
		mgr:           mgr,
		ebpfManager:   ebpfManager,
		configUpdates: configUpdates,
		criWrapper:    &criWrapper,
	}, nil
}

func (o *odiglet) run(ctx context.Context) {
	var wg sync.WaitGroup

	if err := o.criWrapper.Connect(); err != nil {
		log.Logger.Error(err, "Failed to connect to CRI runtime")
	}

	defer o.criWrapper.Close()

	// Start pprof server
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := common.StartPprofServer(ctx, log.Logger)
		if err != nil {
			log.Logger.Error(err, "Failed to start pprof server")
		} else {
			log.Logger.V(0).Info("Pprof server exited")
		}
	}()

	// Start device manager
	// the device manager library doesn't support passing a context,
	// however, internally it uses a context to cancel the device manager once SIGTERM or SIGINT is received.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runDeviceManager(o.clientset)
		log.Logger.V(0).Info("Device manager exited")
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := o.ebpfManager.Run(ctx)
		if err != nil {
			log.Logger.Error(err, "Failed to run ebpf manager")
		}
		log.Logger.V(0).Info("eBPF manager exited")
	}()

	// start OpAmp server
	odigosNs := k8senv.GetCurrentNamespace()
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := server.StartOpAmpServer(ctx, log.Logger, o.mgr, o.clientset, env.Current.NodeName, odigosNs)
		if err != nil {
			log.Logger.Error(err, "Failed to start opamp server")
		}
		log.Logger.V(0).Info("OpAmp server exited")
	}()

	// start kube manager
	wg.Add(1)
	go func() {
		defer wg.Done()
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
	}()

	<-ctx.Done()
	wg.Wait()
}

func main() {
	// If started in init mode
	if len(os.Args) == 2 && os.Args[1] == "init" {
		odigletInitPhase()
	}

	if err := log.Init(); err != nil {
		panic(err)
	}

	log.Logger.V(0).Info("Starting odiglet")

	// Load env
	if err := env.Load(); err != nil {
		log.Logger.Error(err, "Failed to load env")
		os.Exit(1)
	}

	o, err := newOdiglet()
	if err != nil {
		log.Logger.Error(err, "Failed to initialize odiglet")
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()
	o.run(ctx)

	log.Logger.V(0).Info("odiglet exiting")
}

func runDeviceManager(clientset *kubernetes.Clientset) {
	log.Logger.V(0).Info("Starting device manager")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	otelSdkLsf := map[common.ProgrammingLanguage]map[common.OtelSdk]instrumentation.LangSpecificFunc{
		common.GoProgrammingLanguage: {
			common.OtelSdkEbpfCommunity: instrumentlang.Go,
		},
		common.JavaProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Java,
		},
		common.PythonProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Python,
		},
		common.JavascriptProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.NodeJS,
		},
		common.DotNetProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.DotNet,
		},
		common.NginxProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Nginx,
		},
	}

	lister, err := instrumentation.NewLister(ctx, clientset, otelSdkLsf)
	if err != nil {
		log.Logger.Error(err, "Failed to create new lister")
		os.Exit(-1)
	}

	manager := dpm.NewManager(lister)
	manager.Run()
}
