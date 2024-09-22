package main

import (
	"context"
	"os"

	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/fs"

	"github.com/kubevirt/device-plugin-manager/pkg/dpm"
	"github.com/odigos-io/odigos/common"
	k8senv "github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/instrumentlang"
	"github.com/odigos-io/odigos/odiglet/pkg/kube"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	_ "net/http/pprof"
)

func odigletInitPhase() {
	err := fs.CopyAgentsDirectoryToHost()
	if err != nil {
		log.Logger.Error(err, "Failed to copy agents directory to host")
		os.Exit(-1)
	}
	os.Exit(0)
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

	ctx := signals.SetupSignalHandler()

	go common.StartPprofServer(log.Logger)

	go startDeviceManager(clientset)

	mgr, err := kube.CreateManager()
	if err != nil {
		log.Logger.Error(err, "Failed to create controller-runtime manager")
		os.Exit(-1)
	}

	odigosNs := k8senv.GetCurrentNamespace()
	err = server.StartOpAmpServer(ctx, log.Logger, mgr, clientset, env.Current.NodeName, odigosNs)
	if err != nil {
		log.Logger.Error(err, "Failed to start opamp server")
	}

	ebpfDirectors, err := initEbpf(ctx, mgr.GetClient(), mgr.GetScheme())
	if err != nil {
		log.Logger.Error(err, "Failed to init eBPF director")
		os.Exit(-1)
	}

	err = kube.SetupWithManager(mgr, ebpfDirectors, clientset)
	if err != nil {
		log.Logger.Error(err, "Failed to setup controller-runtime manager")
		os.Exit(-1)
	}

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

func initEbpf(ctx context.Context, client client.Client, scheme *runtime.Scheme) (ebpf.DirectorsMap, error) {
	goInstrumentationFactory := ebpf.NewGoInstrumentationFactory(client)
	goDirector := ebpf.NewEbpfDirector(ctx, client, scheme, common.GoProgrammingLanguage, goInstrumentationFactory)
	goDirectorKey := ebpf.DirectorKey{
		Language: common.GoProgrammingLanguage,
		OtelSdk:  common.OtelSdkEbpfCommunity,
	}

	return ebpf.DirectorsMap{
		goDirectorKey: goDirector,
	}, nil
}
