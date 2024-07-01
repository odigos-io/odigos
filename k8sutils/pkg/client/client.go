package client

import (
	"os"

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
