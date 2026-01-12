package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/odigos-io/odigos/common/consts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errNoOdigosNamespaceFound = errors.New("Odigos installation is not found in any namespace")

func getNamespaceFromConfigMap(client *kube.Client, ctx context.Context, configMapName string) (string, error) {
	configMap, err := client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
		FieldSelector: fmt.Sprintf("metadata.name=%s", configMapName),
	})

	if err != nil {
		return "", fmt.Errorf("failed to get odigos namespace: %w", err)
	}

	if len(configMap.Items) == 0 {
		return "", errNoOdigosNamespaceFound
	}
	if len(configMap.Items) != 1 {
		return "", fmt.Errorf("expected to get 1 namespace got %d", len(configMap.Items))
	}

	return configMap.Items[0].Namespace, nil
}

func GetOdigosNamespace(client *kube.Client, ctx context.Context) (string, error) {
	configMapName, err := getNamespaceFromConfigMap(client, ctx, consts.OdigosConfigurationName)
	if err == nil {
		return configMapName, nil
	}
	if !IsErrNoOdigosNamespaceFound(err) {
		return "", fmt.Errorf("failed to get odigos namespace: %w", err)
	}
	// we need this fallback because old versions of odigos has legacy config map called "odigos-config", and
	// several commands needs to get current namespace from it.
	legacyConfigMap, err := getNamespaceFromConfigMap(client, ctx, consts.OdigosLegacyConfigName)
	if err == nil {
		return legacyConfigMap, nil
	}
	if !IsErrNoOdigosNamespaceFound(err) {
		return "", fmt.Errorf("failed to get odigos namespace: %w", err)
	}

	return "", errNoOdigosNamespaceFound
}

func IsErrNoOdigosNamespaceFound(err error) bool {
	return errors.Is(err, errNoOdigosNamespaceFound)
}
