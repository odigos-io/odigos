package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	syaml "sigs.k8s.io/yaml"
)

func K8sManifest(ctx context.Context, namespace string, kind model.K8sResourceKind, name string) (string, error) {

	// this can be extended to support other kinds of resources in the future.
	switch kind {
	case model.K8sResourceKindDeployment:
		obj, err := kube.DefaultClient.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		jb, _ := json.Marshal(obj)
		yb, _ := syaml.JSONToYAML(jb)
		return string(yb), nil
	case model.K8sResourceKindDaemonSet:
		obj, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		jb, _ := json.Marshal(obj)
		yb, _ := syaml.JSONToYAML(jb)
		return string(yb), nil
	case model.K8sResourceKindPod:
		obj, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		jb, _ := json.Marshal(obj)
		yb, _ := syaml.JSONToYAML(jb)
		return string(yb), nil
	default:
		return "", fmt.Errorf("unsupported kind: %s", kind)
	}
}
