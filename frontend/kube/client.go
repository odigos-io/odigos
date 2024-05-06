package kube

import (
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/typed/actions/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
)

var DefaultClient *Client

func SetDefaultClient(client *Client) {
	DefaultClient = client
}

type Client struct {
	kubernetes.Interface
	OdigosClient  odigosv1alpha1.OdigosV1alpha1Interface
	ActionsClient actionsv1alpha1.ActionsV1alpha1Interface
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

	odigosClient, err := odigosv1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	actionsClient, err := actionsv1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Interface:     clientset,
		OdigosClient:  odigosClient,
		ActionsClient: actionsClient,
	}, nil
}
