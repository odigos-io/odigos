package main

import (
	"context"
	"os"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/odiglet/pkg/ebpf"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation"
	"github.com/keyval-dev/odigos/odiglet/pkg/kube"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	if err := log.Init(); err != nil {
		panic(err)
	}
	log.Logger.V(0).Info("Starting odiglet")

	// Load env
	if err := env.Load(); err != nil {
		log.Logger.Error(err, "Failed to load env")
		os.Exit(1)
	}

	// Init Kubernetes API client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Logger.Error(err, "Failed to init Kubernetes API client")
		os.Exit(-1)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Logger.Error(err, "Failed to init Kubernetes API client")
		os.Exit(-1)
	}

	ebpfDirectors, err := initEbpf()
	if err != nil {
		log.Logger.Error(err, "Failed to init eBPF director")
		os.Exit(-1)
	}

	go startDeviceManager(clientset)

	mgr, err := kube.CreateManager()
	if err != nil {
		log.Logger.Error(err, "Failed to create controller-runtime manager")
		os.Exit(-1)
	}

	err = kube.SetupWithManager(mgr, ebpfDirectors)
	if err != nil {
		log.Logger.Error(err, "Failed to setup controller-runtime manager")
		os.Exit(-1)
	}

	ctx := signals.SetupSignalHandler()
	err = kube.StartManager(ctx, mgr)
	if err != nil {
		log.Logger.Error(err, "Failed to start controller-runtime manager")
		os.Exit(-1)
	}

	<-ctx.Done()
	for _, director := range ebpfDirectors {
		director.Shutdown()
	}
}

func startDeviceManager(clientset *kubernetes.Clientset) {
	log.Logger.V(0).Info("Starting device manager")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lister, err := instrumentation.NewLister(ctx, clientset)
	if err != nil {
		log.Logger.Error(err, "Failed to create new lister")
		os.Exit(-1)
	}

	manager := dpm.NewManager(lister)
	manager.Run()
}

func initEbpf() (map[common.ProgrammingLanguage]ebpf.Director, error) {
	goDirector, err := ebpf.NewInstrumentationDirectorGo()
	if err != nil {
		return nil, err
	}

	return map[common.ProgrammingLanguage]ebpf.Director{
		common.GoProgrammingLanguage: goDirector,
	}, nil
}
