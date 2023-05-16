package main

import (
	"context"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	"github.com/keyval-dev/odigos/odiglet/pkg/instrumentation"
	"github.com/keyval-dev/odigos/odiglet/pkg/kube"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
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

	go startDeviceManager(clientset)

	if err := kube.StartReconciling(); err != nil {
		log.Logger.Error(err, "Failed to start reconciling")
		os.Exit(-1)
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
