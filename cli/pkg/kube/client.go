package kube

import (
	"fmt"
	"github.com/keyval-dev/odigos/cli/pkg/generated/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

type Client struct {
	kubernetes.Interface
	ApiExtensions apiextensionsclient.Interface
	OdigosClient  v1alpha1.OdigosV1alpha1Interface
}

func CreateClient(cmd *cobra.Command) *Client {
	kc := cmd.Flag("kubeconfig").Value.String()
	config, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		printClientError(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		printClientError(err)
	}

	extendClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		printClientError(err)
	}

	odigosClient, err := v1alpha1.NewForConfig(config)
	if err != nil {
		printClientError(err)
	}

	return &Client{
		Interface:     clientset,
		ApiExtensions: extendClientset,
		OdigosClient:  odigosClient,
	}
}

func printClientError(err error) {
	fmt.Printf("\033[31mERROR\033[0m Could not connect to Kubernetes cluster\n%s\n", err)
	os.Exit(-1)
}
