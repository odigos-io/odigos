package kube

import (
	"context"
	"fmt"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/keyval-dev/odigos/cli/pkg/generated/clientset/versioned/typed/odigos/v1alpha1"
	odigoslabels "github.com/keyval-dev/odigos/cli/pkg/labels"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	kubernetes.Interface
	Dynamic       *dynamic.DynamicClient
	ApiExtensions apiextensionsclient.Interface
	OdigosClient  v1alpha1.OdigosV1alpha1Interface
}

func CreateClient(cmd *cobra.Command) (*Client, error) {
	kc := cmd.Flag("kubeconfig").Value.String()
	config, err := clientcmd.BuildConfigFromFlags("", kc)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	extendClientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	odigosClient, err := v1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Interface:     clientset,
		Dynamic:       dynamicClient,
		ApiExtensions: extendClientset,
		OdigosClient:  odigosClient,
	}, nil
}

func PrintClientErrorAndExit(err error) {
	fmt.Printf("\033[31mERROR\033[0m Could not connect to Kubernetes cluster\n%s\n", err)
	os.Exit(-1)
}

func (c *Client) ApplyResource(ctx context.Context, ns string, odigosVersion string, obj interface{}, typemeta metav1.TypeMeta, objectmeta metav1.ObjectMeta) error {

	// for each resource, add a label with odigos version.
	// we can use this label to later delete all resources
	// which are not part of the up-to-date odigos version.
	// this works because we update an existing label object,
	// so by modifying `objectmeta`, we also modify `obj` object itself.
	currentLabels := objectmeta.GetLabels()
	if currentLabels == nil {
		fmt.Printf("ERROR: resource %s/%s has no labels\n", typemeta.Kind, objectmeta.Name)
		os.Exit(1)
	}
	currentLabels[odigoslabels.OdigosSystemVersionLabelKey] = odigosVersion
	currentLabels[odigoslabels.OdigosSystemLabelKey] = odigoslabels.OdigosSystemLabelValue

	depBytes, _ := yaml.Marshal(obj)

	force := true
	patchOptions := metav1.PatchOptions{
		FieldManager: "odigos",
		Force:        &force,
	}

	resourceName := objectmeta.Name
	_, err := c.Dynamic.Resource(TypeMetaToDynamicResource(typemeta)).Namespace(ns).Patch(ctx, resourceName, k8stypes.ApplyPatchType, depBytes, patchOptions)
	return err
}

func (c *Client) DeleteOldOdigosSystemObjects(ctx context.Context, resourceAndNamespace ResourceAndNs, odigosVersion string) error {
	systemObject, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemLabelKey, selection.Equals, []string{odigoslabels.OdigosSystemLabelValue})
	notLatestVersion, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemVersionLabelKey, selection.NotEquals, []string{odigosVersion})
	labelSelector := k8slabels.NewSelector().Add(*systemObject).Add(*notLatestVersion).String()
	resource := resourceAndNamespace.Resource
	ns := resourceAndNamespace.Namespace
	return c.Dynamic.Resource(resource).Namespace(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
}
