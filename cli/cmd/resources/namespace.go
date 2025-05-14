package resources

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/labels"
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
	configMaps, err := client.CoreV1().ConfigMaps("").List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})
	if err != nil {
		return "", err
	}

	if len(configMaps.Items) == 0 {
		return "", errNoOdigosNamespaceFound
	} 
	return configMaps.Items[0].Namespace, nil
}

func IsErrNoOdigosNamespaceFound(err error) bool {
	return errors.Is(err, errNoOdigosNamespaceFound)
}
