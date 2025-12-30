package client

import (
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClientConfigWithContext(kc string, context string) (*rest.Config, error) {
	var kubeConfig *rest.Config
	var err error

	if IsRunningInKubernetes() {
		// Running inside a Kubernetes cluster
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Loading kubeconfig from file with optional context override
		loadingRules := &clientcmd.ClientConfigLoadingRules{
			ExplicitPath: kc, // Path to kubeconfig
		}

		configOverrides := &clientcmd.ConfigOverrides{}
		if context != "" {
			configOverrides.CurrentContext = context
		}

		// Build the kubeconfig
		kubeConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules,
			configOverrides,
		).ClientConfig()

		if err != nil {
			return nil, err
		}
	}
	return kubeConfig, nil
}

func IsRunningInKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

type ClusterDetails struct {
	CurrentContext string
	ClusterName    string
	ServerEndpoint string
}

func GetCurrentClusterDetails(kc string, kContext string) ClusterDetails {
	config, err := clientcmd.LoadFromFile(kc)
	if err != nil {
		return ClusterDetails{}
	}

	var ctx string
	if kContext != "" {
		ctx = kContext
	} else {
		ctx = config.CurrentContext
	}
	cluster := ""
	if val, ok := config.Contexts[ctx]; ok {
		cluster = val.Cluster
	} else if kContext != "" { // If context is provided, but not found in kubeconfig
		fmt.Printf("Context %s not found in kubeconfig, bailing\n", kContext)
		os.Exit(1)
	}

	server := ""
	if val, ok := config.Clusters[cluster]; ok {
		server = val.Server
	}

	return ClusterDetails{
		CurrentContext: ctx,
		ClusterName:    cluster,
		ServerEndpoint: server,
	}
}

func GetK8sClientset() (*kubernetes.Clientset, error) {
	// Init Kubernetes API client
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

// IsResourceAvailable checks if the external resource is available in the cluster
// using the RESTMapper to avoid permission errors on clusters that don't have the resource installed.
func IsResourceAvailable(mapper meta.RESTMapper, gvk schema.GroupVersionKind) bool {
	_, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	return err == nil
}
