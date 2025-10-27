package services

import (
    "context"
    "encoding/json"

    "github.com/odigos-io/odigos/frontend/graph/model"
    "github.com/odigos-io/odigos/frontend/kube"
    "github.com/odigos-io/odigos/k8sutils/pkg/env"

    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "sigs.k8s.io/yaml"
    "fmt"
)

// GetManifest returns YAML/JSON manifest for supported workloads by kind/name/namespace.
func GetManifest(ctx context.Context, kind model.K8sResourceKind, name string, namespace *string, format *model.ManifestFormat) (string, error) {
    ns := env.GetCurrentNamespace()
    if namespace != nil && *namespace != "" {
        ns = *namespace
    }

    var obj any
    var err error

    switch kind {
    case model.K8sResourceKindDeployment:
        obj, err = kube.DefaultClient.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
    case model.K8sResourceKindDaemonSet:
        obj, err = kube.DefaultClient.AppsV1().DaemonSets(ns).Get(ctx, name, metav1.GetOptions{})
    case model.K8sResourceKindConfigMap:
        obj, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
    default:
        return "", fmt.Errorf("unsupported resource kind: %s", kind)
    }
    if err != nil { return "", err }

    var u unstructured.Unstructured
    b, err := json.Marshal(obj)
    if err != nil { return "", err }
    if err := json.Unmarshal(b, &u.Object); err != nil { return "", err }
    unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")

    if format != nil && *format == model.ManifestFormatJSON {
        out, err := json.MarshalIndent(u.Object, "", "  ")
        if err != nil { return "", err }
        return string(out), nil
    }
    out, err := yaml.Marshal(u.Object)
    if err != nil { return "", err }
    return string(out), nil
}


