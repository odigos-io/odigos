package clusterinfo

import (
	"context"
	"encoding/json"

	"github.com/odigos-io/odigos/api/k8sconsts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

func RecordClusterInfo(ctx context.Context, k8sClient *kubernetes.Clientset, odigosNs string) error {
	serverVersion, err := k8sClient.Discovery().ServerVersion()
	if err != nil {
		return err
	}

	patch := []map[string]string{
		{
			"op":    "add",
			"path":  "/data/" + k8sconsts.OdigosDeploymentConfigMapKubernetesVersionKey,
			"value": serverVersion.String(),
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	_, err = k8sClient.CoreV1().ConfigMaps(odigosNs).Patch(ctx, k8sconsts.OdigosDeploymentConfigMapName, types.JSONPatchType, patchBytes, metav1.PatchOptions{})
	return err
}
