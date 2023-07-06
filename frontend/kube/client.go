package kube

import (
	"github.com/keyval-dev/odigos/frontend/generated/clientset/versioned/typed/odigos/v1alpha1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var DefaultClient *Client

func SetDefaultClient(client *Client) {
	DefaultClient = client
}

type Client struct {
	kubernetes.Interface
	OdigosClient v1alpha1.OdigosV1alpha1Interface
}

func CreateClient(kubeConfig string) (*Client, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	odigosClient, err := v1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Interface:    clientset,
		OdigosClient: odigosClient,
	}, nil
}
