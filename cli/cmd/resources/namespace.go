package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/labels"
	"github.com/odigos-io/odigos/common/consts"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var errNoOdigosNamespaceFound = errors.New("Odigos installation is not found in any namespace")

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels.OdigosSystem,
		},
	}
}

func GetOdigosNamespace(client *kube.Client, ctx context.Context) (string, error) {
	configMap, err := client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
		FieldSelector: fmt.Sprintf("metadata.name=%s", consts.OdigosConfigurationName),
	})
	if err != nil {
		return "", err
	}

	if len(configMap.Items) == 0 {
		return "", errNoOdigosNamespaceFound
	} else if len(configMap.Items) != 1 {
		return "", fmt.Errorf("expected to get 1 namespace got %d", len(configMap.Items))
	}

	return configMap.Items[0].Namespace, nil
}

func IsErrNoOdigosNamespaceFound(err error) bool {
	return errors.Is(err, errNoOdigosNamespaceFound)
}
