package kube

import (
	"sync"

	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/typed/actions/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
)

var (
	DefaultClient                       *Client
	Scheme                              = runtime.NewScheme()
	deploymentConfigAvailable           bool
	deploymentConfigAvailabilityChecked bool
	deploymentConfigCheckMu             sync.Mutex
)

func init() {
	// Register OpenShift types with the scheme
	_ = openshiftappsv1.AddToScheme(Scheme)
}

func SetDefaultClient(client *Client) {
	DefaultClient = client
}

type Client struct {
	kubernetes.Interface
	OdigosClient   odigosv1alpha1.OdigosV1alpha1Interface
	ActionsClient  actionsv1alpha1.ActionsV1alpha1Interface
	MetadataClient metadata.Interface
	DynamicClient  dynamic.Interface
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

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &Client{
		Interface:      clientset,
		OdigosClient:   odigosClient,
		ActionsClient:  actionsClient,
		MetadataClient: metadataClient,
		DynamicClient:  dynamicClient,
	}, nil
}

// IsDeploymentConfigAvailable checks if the DeploymentConfig resource is available in the cluster.
// This is cached after the first check to avoid repeated API calls.
func IsDeploymentConfigAvailable() bool {
	deploymentConfigCheckMu.Lock()
	defer deploymentConfigCheckMu.Unlock()

	if deploymentConfigAvailabilityChecked {
		return deploymentConfigAvailable
	}

	// Check if DeploymentConfig resource exists using discovery API
	// This avoids permission issues since discovery is a read-only operation
	gv := schema.GroupVersion{
		Group:   "apps.openshift.io",
		Version: "v1",
	}

	resourceList, err := DefaultClient.Discovery().ServerResourcesForGroupVersion(gv.String())
	if err != nil {
		// Resource group doesn't exist
		deploymentConfigAvailable = false
		deploymentConfigAvailabilityChecked = true
		return false
	}

	// Check if deploymentconfigs resource exists in the group
	for _, resource := range resourceList.APIResources {
		if resource.Name == "deploymentconfigs" {
			deploymentConfigAvailable = true
			deploymentConfigAvailabilityChecked = true
			return true
		}
	}

	deploymentConfigAvailable = false
	deploymentConfigAvailabilityChecked = true
	return false
}
