package kube

import (
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var DefaultClient *Client

func SetDefaultClient(client *Client) {
	DefaultClient = client
}

type Client struct {
	kubernetes.Interface
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

	return &Client{
		Interface: clientset,
	}, nil
}
