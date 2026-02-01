package diagnose

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

// FetchConfigMaps collects all ConfigMaps from the Odigos namespace
func FetchConfigMaps(ctx context.Context, client kubernetes.Interface, builder Builder, configMapDir, odigosNamespace string) error {
	klog.V(2).InfoS("Fetching ConfigMaps", "namespace", odigosNamespace)

	configMaps, err := client.CoreV1().ConfigMaps(odigosNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list configmaps: %w", err)
	}

	for i := 0; i < len(configMaps.Items); i++ {
		cm := &configMaps.Items[i]
		// Clean managedFields for cleaner output
		cm.ManagedFields = nil

		yamlData, err := yaml.Marshal(&cm)
		if err != nil {
			klog.V(1).ErrorS(err, "Failed to marshal ConfigMap to YAML", "name", cm.Name)
			continue
		}

		filename := fmt.Sprintf("configmap-%s.yaml", cm.Name)
		if err := builder.AddFile(configMapDir, filename, yamlData); err != nil {
			klog.V(1).ErrorS(err, "Failed to save ConfigMap", "name", cm.Name)
			continue
		}
	}

	return nil
}
