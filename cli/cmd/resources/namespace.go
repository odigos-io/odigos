package resources

import (
	"context"
	"fmt"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
	"github.com/keyval-dev/odigos/cli/pkg/labels"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewNamespace(name string) *v1.Namespace {
	return &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels.OdigosSystem,
		},
	}
}

func GetOdigosNamespace(client *kube.Client, ctx context.Context) (string, error) {
	namespaces, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(&metav1.LabelSelector{
			MatchLabels: labels.OdigosSystem,
		}),
	})

	if err != nil {
		return "", err
	}

	if len(namespaces.Items) == 0 {
		return "", fmt.Errorf("odigos is not currently installed in the cluster, so there is nothing to uninstall")
	} else if len(namespaces.Items) != 1 {
		return "", fmt.Errorf("expected to get 1 namespace got %d", len(namespaces.Items))
	}

	return namespaces.Items[0].Name, nil
}
