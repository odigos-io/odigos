package node

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
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
