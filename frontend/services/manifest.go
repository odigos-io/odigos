package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/odigos-io/odigos/frontend/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
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
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	case model.K8sResourceKindDaemonSet:
		obj, err := kube.DefaultClient.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	case model.K8sResourceKindStatefulSet:
		obj, err := kube.DefaultClient.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	case model.K8sResourceKindCronJob:
		obj, err := kube.DefaultClient.BatchV1().CronJobs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	case model.K8sResourceKindPod:
		obj, err := kube.DefaultClient.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	case model.K8sResourceKindConfigMap:
		obj, err := kube.DefaultClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return strings.ReplaceAll(string(yb), ": |", ":"), nil

	case model.K8sResourceKindDeploymentConfig:
		dcClient := kube.DefaultClient.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "apps.openshift.io",
			Version:  "v1",
			Resource: "deploymentconfigs",
		}).Namespace(namespace)

		dc, err := dcClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		dc.SetManagedFields(nil)

		dcYamlBytes, err := syaml.Marshal(dc.Object)
		if err != nil {
			return "", err
		}
		return string(dcYamlBytes), nil

	case model.K8sResourceKindRollout:
		rolloutClient := kube.DefaultClient.DynamicClient.Resource(schema.GroupVersionResource{
			Group:    "argoproj.io",
			Version:  "v1alpha1",
			Resource: "rollouts",
		}).Namespace(namespace)

		rollout, err := rolloutClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}

		rollout.SetManagedFields(nil)

		rolloutYamlBytes, err := syaml.Marshal(rollout.Object)
		if err != nil {
			return "", err
		}
		return string(rolloutYamlBytes), nil

	case model.K8sResourceKindInstrumentationConfig:
		obj, err := kube.DefaultClient.OdigosClient.InstrumentationConfigs(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return "", err
		}
		obj.ObjectMeta.ManagedFields = nil
		yb, err := syaml.Marshal(obj)
		if err != nil {
			return "", err
		}
		return string(yb), nil

	default:
		return "", fmt.Errorf("unsupported kind: %s", kind)
	}
}
