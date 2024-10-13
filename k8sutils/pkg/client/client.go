package client

import (
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetClientConfig(kc string) (*rest.Config, error) {
	var kubeConfig *rest.Config
	var err error

	if IsRunningInKubernetes() {
		kubeConfig, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kc)
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

func GetCurrentClusterDetails(kc string) ClusterDetails {
	config, err := clientcmd.LoadFromFile(kc)
	if err != nil {
		return ClusterDetails{}
	}

	ctx := config.CurrentContext
	cluster := ""
	if val, ok := config.Contexts[ctx]; ok {
		cluster = val.Cluster
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
