package node

import (
	"context"
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func DetermineNodeOdigletInstalledLabelByTier() string {
	odigosTier := env.GetOdigosTierFromEnv()
	switch string(odigosTier) {
	case string(common.CommunityOdigosTier):
		return k8sconsts.OdigletOSSInstalledLabel
	case string(common.OnPremOdigosTier):
		return k8sconsts.OdigletEnterpriseInstalledLabel
	default:
		return k8sconsts.OdigletOSSInstalledLabel
	}
}

func PrepareNodeForOdigosInstallation(clientset *kubernetes.Clientset, nodeName string) error {
	ctx := context.Background()

	// Determine Odigos Installed label [OSS/Enterprise]
	labelKey := DetermineNodeOdigletInstalledLabelByTier()

	return retry.OnError(retry.DefaultBackoff, apierrors.IsConflict, func() error {
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", nodeName, err)
		}

		// Remove startup taint if exists
		newTaints := make([]v1.Taint, 0, len(node.Spec.Taints))
		for _, taint := range node.Spec.Taints {
			if taint.Key == consts.KarpenterStartupTaintKey && taint.Effect == v1.TaintEffectNoSchedule {
				continue
			}
			newTaints = append(newTaints, taint)
		}
		node.Spec.Taints = newTaints

		// Add Odigos Installed label
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}

		// For migration between tiers: ensure only the correct odiglet label is set
		// by removing the other tier's label if it exists.
		switch labelKey {
		case k8sconsts.OdigletOSSInstalledLabel:
			delete(node.Labels, k8sconsts.OdigletEnterpriseInstalledLabel)
		case k8sconsts.OdigletEnterpriseInstalledLabel:
			delete(node.Labels, k8sconsts.OdigletOSSInstalledLabel)
		}

		node.Labels[labelKey] = k8sconsts.OdigletInstalledLabelValue

		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		return err
	})
}

// DetectAndLabelMountMethodOverride checks if the node's OS requires a mount method
// override (e.g., Bottlerocket with SELinux where device plugin host mounts may fail).
// If so, it labels the node with the appropriate override so the instrumentor webhook
// can automatically switch to a compatible mount method.
func DetectAndLabelMountMethodOverride(clientset *kubernetes.Clientset, nodeName string) error {
	ctx := context.Background()

	return retry.OnError(retry.DefaultBackoff, apierrors.IsConflict, func() error {
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", nodeName, err)
		}

		// Check if this node runs Bottlerocket OS.
		// Bottlerocket's SELinux enforcement can prevent the device plugin from
		// correctly bind-mounting host directories into containers, causing agent
		// files at /var/odigos to be inaccessible. The init-container mount method
		// (which uses emptyDir) avoids this entirely.
		osImage := node.Status.NodeInfo.OSImage
		if !strings.Contains(osImage, "Bottlerocket") {
			// Not Bottlerocket, no override needed
			return nil
		}

		// Already labeled -- nothing to do
		if node.Labels[k8sconsts.MountMethodOverrideNodeLabel] == string(common.K8sInitContainerMountMethod) {
			return nil
		}

		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}
		node.Labels[k8sconsts.MountMethodOverrideNodeLabel] = string(common.K8sInitContainerMountMethod)

		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		return err
	})
}
