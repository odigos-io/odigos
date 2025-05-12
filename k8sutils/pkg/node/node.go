package node

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

func AddLabelToNode(clientset *kubernetes.Clientset, nodeName string, labelKey string, labelValue string) error {
	// Add odiglet installed label to node
	patch := []byte(`{"metadata": {"labels": {"` + labelKey + `": "` + labelValue + `"}}}`)
	_, err := clientset.CoreV1().Nodes().Patch(context.Background(), nodeName, types.StrategicMergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return err
	}
	return nil
}

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

func RemoveStartupTaint(clientset *kubernetes.Clientset, nodeName string) error {
	const (
		startupTaintKey    = consts.KarpenterStartupTaintKey
		startupTaintEffect = v1.TaintEffectNoSchedule
	)

	ctx := context.Background()

	err := retry.OnError(retry.DefaultBackoff, apierrors.IsConflict, func() error {
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("failed to get node %s: %w", nodeName, err)
		}

		originalTaints := node.Spec.Taints
		newTaints := make([]v1.Taint, 0, len(originalTaints))
		removed := false

		for _, taint := range originalTaints {
			if taint.Key == startupTaintKey && taint.Effect == startupTaintEffect {
				removed = true
				continue
			}
			newTaints = append(newTaints, taint)
		}

		if !removed {
			// Taint not found, nothing to remove
			return nil
		}

		node.Spec.Taints = newTaints

		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to remove startup taint from node %s: %w", nodeName, err)
	}

	return nil
}
