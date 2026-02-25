package kube

import (
	"context"
	"sync"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	actionsv1alpha1 "github.com/odigos-io/odigos/api/generated/actions/clientset/versioned/typed/actions/v1alpha1"
	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned/typed/odigos/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/metadata"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/restmapper"
)

var (
	DefaultClient                        *Client
	Scheme                               = runtime.NewScheme()
	deploymentConfigAvailable            bool
	deploymentConfigAvailabilityChecked  bool
	deploymentConfigCheckMu              sync.Mutex
	IsOpenShiftDeploymentConfigAvailable bool
	IsArgoRolloutAvailable               bool
	argoRolloutCheckOnce                 sync.Once
)

func init() {
	// Register OpenShift types with the scheme
	_ = openshiftappsv1.AddToScheme(Scheme)
	// Register Argo Rollouts types with the scheme
	_ = argorolloutsv1alpha1.AddToScheme(Scheme)
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
	RESTMapper     meta.RESTMapper
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

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return nil, err
	}

	groupResources, err := restmapper.GetAPIGroupResources(discoveryClient)
	if err != nil {
		return nil, err
	}

	mapper := restmapper.NewDiscoveryRESTMapper(groupResources)

	return &Client{
		Interface:      clientset,
		OdigosClient:   odigosClient,
		ActionsClient:  actionsClient,
		MetadataClient: metadataClient,
		DynamicClient:  dynamicClient,
		RESTMapper:     mapper,
	}, nil
}

// IsDeploymentConfigAvailable checks if the DeploymentConfig resource is available in the cluster
// and if we have permission to list it. This is cached after the first check to avoid repeated API calls.
func IsDeploymentConfigAvailable() bool {
	deploymentConfigCheckMu.Lock()
	defer deploymentConfigCheckMu.Unlock()

	if deploymentConfigAvailabilityChecked {
		return deploymentConfigAvailable
	}

	// Try to actually list DeploymentConfigs with a limit of 0 to check both:
	// 1. If the resource type exists
	// 2. If we have permission to access it
	gvr := schema.GroupVersionResource{
		Group:    "apps.openshift.io",
		Version:  "v1",
		Resource: "deploymentconfigs",
	}

	// Use a limit of 0 to make this a cheap check
	listOptions := metav1.ListOptions{Limit: 0}
	_, err := DefaultClient.DynamicClient.Resource(gvr).List(context.Background(), listOptions)

	if err != nil {
		// Resource doesn't exist or we don't have permission to access it
		deploymentConfigAvailable = false
		deploymentConfigAvailabilityChecked = true
		return false
	}

	// Resource exists and we have permission
	deploymentConfigAvailable = true
	deploymentConfigAvailabilityChecked = true
	return true
}

// InitWorkloadKindsAvailability checks if the Argo Rollout and open shift deployment config resource is available in the cluster
// and sets IsArgoRolloutAvailable and IsOpenShiftDeploymentConfigAvailable. This should be called once during initialization.
func InitWorkloadKindsAvailability() {
	argoRolloutCheckOnce.Do(func() {
		IsArgoRolloutAvailable = k8sutils.IsResourceAvailable(DefaultClient.RESTMapper, k8sconsts.ArgoRolloutGVK)
		IsOpenShiftDeploymentConfigAvailable = k8sutils.IsResourceAvailable(DefaultClient.RESTMapper, k8sconsts.DeploymentConfigGVK)
	})
}
