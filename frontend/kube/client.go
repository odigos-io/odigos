package kube

import (
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/typed/actions/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var DefaultClient *Client

func SetDefaultClient(client *Client) {
	DefaultClient = client
}

type Client struct {
	kubernetes.Interface
	OdigosClient   odigosv1alpha1.OdigosV1alpha1Interface
	ActionsClient  actionsv1alpha1.ActionsV1alpha1Interface
	MetadataClient metadata.Interface
}

func CreateClient(kubeConfig string, kContext string) (*Client, error) {
	config, err := k8sutils.GetClientConfigWithContext(kubeConfig, kContext)
	if err != nil {
		return nil, err
	}

	config.QPS = k8sconsts.K8sClientDefaultQPS
	config.Burst = k8sconsts.K8sClientDefaultBurst

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

	metadataClient, err := metadata.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Interface:      clientset,
		OdigosClient:   odigosClient,
		ActionsClient:  actionsClient,
		MetadataClient: metadataClient,
	}, nil
}
