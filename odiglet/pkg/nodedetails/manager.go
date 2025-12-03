package nodedetails

import (
	"context"
	"fmt"
	"os"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// CollectAndPersist runs all registered features and creates the NodeDetails CRD.
// This function is called during the odiglet init phase.
// The NodeDetails has owner references to both the odiglet pod and the node,
// so it will be garbage collected when the pod is deleted.
func CollectAndPersist(ctx context.Context, odigosClient odigosclientset.Interface, node *v1.Node, podName string, podUID string, namespace string) error {
	nodeName := node.Name
	log.Logger.V(0).Info("Checking node features", "node", nodeName, "features", len(features))

	// Initialize the spec
	spec := &v1alpha1.NodeDetailsSpec{
		WaspEnabled: false, // Default to false, enterprise can override
	}

	// Check all features
	for _, feature := range features {
		log.Logger.V(1).Info("Checking feature", "feature", feature.Name())
		if err := feature.Check(ctx, node, spec); err != nil {
			log.Logger.Error(err, "Feature check failed", "feature", feature.Name())
			// Continue with other features even if one fails
		}
	}

	// Create NodeDetails with owner reference to the odiglet pod
	// When the Odiglet pod is deleted (e.g., during daemonset update or node drain),
	// the NodeDetails will be automatically deleted.
	blockOwnerDeletion := true
	nodeDetails := &v1alpha1.NodeDetails{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nodeName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         "v1",
					Kind:               "Pod",
					Name:               podName,
					UID:                types.UID(podUID),
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
		Spec: *spec,
	}

	nodeDetailsClient := odigosClient.OdigosV1alpha1().NodeDetailses(namespace)
	if _, err := nodeDetailsClient.Create(ctx, nodeDetails, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			log.Logger.V(0).Info("NodeDetails already exists", "node", nodeName)
			return nil
		}
		return fmt.Errorf("failed to create NodeDetails: %w", err)
	}

	log.Logger.V(0).Info("Successfully created NodeDetails",
		"node", nodeName,
		"pod", podName,
		"kernelVersion", spec.KernelVersion,
		"cpuCapacity", spec.CPUCapacity,
		"memoryCapacity", spec.MemoryCapacity,
		"waspEnabled", spec.WaspEnabled)

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
