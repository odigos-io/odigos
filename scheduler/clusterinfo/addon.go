package clusterinfo

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

// InitializeAddonState owns keys deliberately omitted from the EKS add-on manifests.
// Updating with resourceVersion preserves concurrent user changes and existing offsets.
func InitializeAddonState(ctx context.Context, client kubernetes.Interface, namespace string) error {
	for _, name := range []string{k8sconsts.OdigosDeploymentConfigMapName, k8sconsts.GoOffsetsConfigMap} {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			cm, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return err
			}
			key, value := k8sconsts.GoOffsetsFileName, ""
			if name == k8sconsts.OdigosDeploymentConfigMapName {
				key, value = k8sconsts.OdigosDeploymentConfigMapOdigosDeploymentIDKey, string(cm.UID)
				if value == "" {
					return fmt.Errorf("deployment ConfigMap has no Kubernetes UID")
				}
			}
			if _, exists := cm.Data[key]; exists {
				return nil
			}
			if cm.Data == nil {
				cm.Data = make(map[string]string)
			}
			cm.Data[key] = value
			_, err = client.CoreV1().ConfigMaps(namespace).Update(ctx, cm, metav1.UpdateOptions{FieldManager: "odigos-addon-state"})
			return err
		})
		if err != nil {
			return fmt.Errorf("initialize add-on state in %s: %w", name, err)
		}
	}
	return nil
}
