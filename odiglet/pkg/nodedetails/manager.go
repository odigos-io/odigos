package nodedetails

import (
	"context"
	"fmt"
	"os"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/generated/odigos/applyconfiguration/odigos/v1alpha1"
	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	applymetav1 "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CollectAndPersist runs all registered features and creates the NodeDetails CRD.
// This function is called during the odiglet init phase.
// The NodeDetails has owner reference to the node, so it will be garbage collected when the node is deleted.
func CollectAndPersist(ctx context.Context, odigosClient odigosclientset.Interface, node *v1.Node, podName string, podUID string, namespace string) error {
	nodeName := node.Name
	log.Logger.V(0).Info("Checking node features", "node", nodeName, "features", len(features))

	// Initialize the spec
	spec := &v1alpha1.NodeDetailsSpec{
		WaspRequired: false, // Default to false, enterprise can override
	}

	// Check all features
	for _, feature := range features {
		log.Logger.V(1).Info("Checking feature", "feature", feature.Name())
		if err := feature.Check(ctx, node, spec); err != nil {
			log.Logger.Error(err, "Feature check failed", "feature", feature.Name())
			// Continue with other features even if one fails
		}
	}

	// Create NodeDetails with owner reference to the Node
	// We use Server-Side Apply so we don't need to define the full object structure here
	// The apply configuration below handles it all

	nodeDetailsClient := odigosClient.OdigosV1alpha1().NodeDetailses(namespace)

	// Use Server-Side Apply to Create or Update
	applyConfig := odigosv1alpha1.NodeDetails(nodeName, namespace).
		WithKind("NodeDetails").
		WithAPIVersion("odigos.io/v1alpha1").
		WithOwnerReferences(applymetav1.OwnerReference().
			WithAPIVersion("v1").
			WithKind("Node").
			WithName(nodeName).
			WithUID(node.UID)).
		WithSpec(odigosv1alpha1.NodeDetailsSpec().
			WithWaspRequired(spec.WaspRequired).
			WithKernelVersion(spec.KernelVersion).
			WithCPUCapacity(spec.CPUCapacity).
			WithMemoryCapacity(spec.MemoryCapacity).
			WithDiscoveryOdigletPodName(podName))

	if _, err := nodeDetailsClient.Apply(ctx, applyConfig, metav1.ApplyOptions{FieldManager: "odiglet"}); err != nil {
		return fmt.Errorf("failed to apply NodeDetails: %w", err)
	}

	log.Logger.V(0).Info("Successfully created NodeDetails",
		"node", nodeName,
		"pod", podName,
		"kernelVersion", spec.KernelVersion,
		"cpuCapacity", spec.CPUCapacity,
		"memoryCapacity", spec.MemoryCapacity,
		"waspRequired", spec.WaspRequired)

	return nil
}

// PrepareAndCollect orchestrates the complete node details collection process.
// It handles reading environment, creating clients, and persisting the NodeDetails.
// This is the main entry point called during odiglet init phase.
func PrepareAndCollect(config *rest.Config, clientset *kubernetes.Clientset, node *v1.Node) error {
	ctx := context.Background()

	// Read pod information from environment
	podName, ok := os.LookupEnv("POD_NAME")
	if !ok {
		return fmt.Errorf("env var POD_NAME is not set")
	}
	podUID, ok := os.LookupEnv("POD_UID")
	if !ok {
		return fmt.Errorf("env var POD_UID is not set")
	}

	// Create Odigos clientset
	odigosClient, err := odigosclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create odigos clientset: %w", err)
	}

	// Get current namespace
	namespace := env.GetCurrentNamespace()

	// Collect features and persist NodeDetails
	return CollectAndPersist(ctx, odigosClient, node, podName, podUID, namespace)
}
