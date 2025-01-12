package kube

import (
	"context"
	"fmt"
	"os"
	"strconv"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigoslabels "github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

type Client struct {
	kubernetes.Interface
	Clientset     *kubernetes.Clientset
	Dynamic       *dynamic.DynamicClient
	ApiExtensions apiextensionsclient.Interface
	OdigosClient  v1alpha1.OdigosV1alpha1Interface
	Config        *rest.Config
}

// Identical to the Object interface defined in controller-runtime: https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/client#Object
// This is a workaround to avoid importing controller-runtime in the cli package
type Object interface {
	metav1.Object
	runtime.Object
}

// GetCLIClientOrExit returns the current kube client from cmd.Context() if one exists.
// otherwise it creates a new client and returns it.
func GetCLIClientOrExit(cmd *cobra.Command) *Client {
	// we can check the cmd context for client, but currently avoiding that due to circular dependencies
	client, err := createClient(cmd)
	if err != nil {
		PrintClientErrorAndExit(err)
	}
	return client
}

func createClient(cmd *cobra.Command) (*Client, error) {
	kc := cmd.Flag("kubeconfig").Value.String()
	kContext := cmd.Flag("kube-context").Value.String()
	config, err := k8sutils.GetClientConfigWithContext(kc, kContext)
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
		Clientset:     clientset,
		Dynamic:       dynamicClient,
		ApiExtensions: extendClientset,
		OdigosClient:  odigosClient,
		Config:        config,
	}, nil
}

func PrintClientErrorAndExit(err error) {
	fmt.Printf("\033[31mERROR\033[0m Could not connect to Kubernetes cluster\n%s\n", err)
	os.Exit(-1)
}

func (c *Client) ApplyResources(ctx context.Context, configVersion int, objs []Object) error {
	for _, obj := range objs {
		err := c.ApplyResource(ctx, configVersion, obj)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) ApplyResource(ctx context.Context, configVersion int, obj Object) error {

	labels := obj.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}
	labels[odigoslabels.OdigosSystemLabelKey] = odigoslabels.OdigosSystemLabelValue
	labels[odigoslabels.OdigosSystemConfigLabelKey] = strconv.Itoa(configVersion)
	obj.SetLabels(labels)

	depBytes, _ := yaml.Marshal(obj)

	force := true
	patchOptions := metav1.PatchOptions{
		FieldManager: "odigos",
		Force:        &force,
	}

	resourceName := obj.GetName()
	gvk := obj.GetObjectKind().GroupVersionKind()
	ns := obj.GetNamespace()
	resource := TypeMetaToDynamicResource(gvk)
	_, err := c.Dynamic.Resource(resource).Namespace(ns).Patch(ctx, resourceName, k8stypes.ApplyPatchType, depBytes, patchOptions)
	return err
}

func (c *Client) DeleteOldOdigosSystemObjects(ctx context.Context, resourceAndNamespace ResourceAndNs, configVersion int, k8sVersion *version.Version) error {
	systemObject, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemLabelKey, selection.Equals, []string{odigoslabels.OdigosSystemLabelValue})
	notLatestVersion, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemConfigLabelKey, selection.NotEquals, []string{strconv.Itoa(configVersion)})
	labelSelector := k8slabels.NewSelector().Add(*systemObject).Add(*notLatestVersion).String()
	resource := resourceAndNamespace.Resource
	ns := resourceAndNamespace.Namespace
	// DeleteCollection is only available in k8s 1.23 and above, for older versions we need to list and delete each resource
	if k8sVersion != nil && k8sVersion.GreaterThan(version.MustParse("1.23")) {
		return c.Dynamic.Resource(resource).Namespace(ns).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
			LabelSelector: labelSelector,
		})
	} else {
		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
		}
		resourceList, err := c.Dynamic.Resource(resource).Namespace(ns).List(ctx, listOptions)
		if err != nil {
			return fmt.Errorf("failed to list resources: %w", err)
		}

		// Delete each resource individually
		for _, item := range resourceList.Items {
			err = c.Dynamic.Resource(resource).Namespace(ns).Delete(ctx, item.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete resource %s: %w", item.GetName(), err)
			}
		}
	}
	return nil
}
