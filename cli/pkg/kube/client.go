package kube

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	odigoslabels "github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/spf13/cobra"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
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

func CreateClient(cmd *cobra.Command) (*Client, error) {
	kc := cmd.Flag("kubeconfig").Value.String()

	config, err := k8sutils.GetClientConfig(kc)
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

// this function is a temporary hack upgrade from versions <v1.0.23 to >v1.0.23
// we changed the selector label on DaemonSets and Deployments from "app" to "app.kubernetes.io/name",
// and apparently this field id immutable:
// ERROR Deployment.apps "odigos-instrumentor" is invalid: spec.selector: Invalid value: v1.LabelSelector{MatchLabels:map[string]string{"app.kubernetes.io/name":"odigos-instrumentor"}, MatchExpressions:[]v1.LabelSelectorRequirement(nil)}: field is immutable
// Once we end support for odigos versions <v1.0.23 we can remove this function
func (c *Client) deleteResourceBeforeAppending(ctx context.Context, obj Object) error {
	gvk := obj.GetObjectKind().GroupVersionKind()
	switch gvk.Kind {

	case "DaemonSet":
		dm, err := c.AppsV1().DaemonSets(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}
		currentSelectorLabels := dm.Spec.Selector.MatchLabels

		newDs, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			return fmt.Errorf("could not cast object to DaemonSet")
		}
		newSelectorLabels := newDs.Spec.Selector.MatchLabels

		// compare the labels by value and delete the current ds if they are different
		if !k8slabels.SelectorFromSet(currentSelectorLabels).Matches(k8slabels.Set(newSelectorLabels)) {
			err = c.AppsV1().DaemonSets(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}

	case "Deployment":
		dep, err := c.AppsV1().Deployments(obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return nil
			}
			return err
		}

		currentSelectorLabels := dep.Spec.Selector.MatchLabels

		newDep, ok := obj.(*appsv1.Deployment)
		if !ok {
			return fmt.Errorf("could not cast object to Deployment")
		}

		newSelectorLabels := newDep.Spec.Selector.MatchLabels

		// compare the labels by value and delete the current deployment if they are different
		if !k8slabels.SelectorFromSet(currentSelectorLabels).Matches(k8slabels.Set(newSelectorLabels)) {
			err = c.AppsV1().Deployments(obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{})
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (c *Client) ApplyResource(ctx context.Context, configVersion int, obj Object) error {

	err := c.deleteResourceBeforeAppending(ctx, obj)
	if err != nil {
		return err
	}

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
	_, err = c.Dynamic.Resource(resource).Namespace(ns).Patch(ctx, resourceName, k8stypes.ApplyPatchType, depBytes, patchOptions)
	return err
}

func (c *Client) DeleteOldOdigosSystemObjects(ctx context.Context, resourceAndNamespace ResourceAndNs, configVersion int) error {
	systemObject, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemLabelKey, selection.Equals, []string{odigoslabels.OdigosSystemLabelValue})
	notLatestVersion, _ := k8slabels.NewRequirement(odigoslabels.OdigosSystemConfigLabelKey, selection.NotEquals, []string{strconv.Itoa(configVersion)})
	labelSelector := k8slabels.NewSelector().Add(*systemObject).Add(*notLatestVersion).String()
	resource := resourceAndNamespace.Resource
	ns := resourceAndNamespace.Namespace
	cfg := ctrl.GetConfigOrDie()
	versionSupported, err := isVersionSupported(cfg, 1, 23)
	if err != nil && versionSupported {
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
func isVersionSupported(cfg *rest.Config, minMajor int, minMinor int) (bool, error) {
	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return false, err
	}

	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, err
	}

	// Parse major and minor versions
	major, err := strconv.Atoi(serverVersion.Major)
	if err != nil {
		return false, fmt.Errorf("failed to parse major version: %w", err)
	}

	minor, err := strconv.Atoi(strings.TrimSuffix(serverVersion.Minor, "+"))
	if err != nil {
		return false, fmt.Errorf("failed to parse minor version: %w", err)
	}

	// Check if the server version meets or exceeds minMajor.minMinor
	return major > minMajor || (major == minMajor && minor >= minMinor), nil
}
