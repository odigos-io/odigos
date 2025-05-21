package node

import (
	"context"
	"fmt"

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
		node.Labels[labelKey] = k8sconsts.OdigletInstalledLabelValue

		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		return err
	})
}
