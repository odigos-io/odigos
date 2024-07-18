package utils

import (
	"context"
	"errors"
	"fmt"
	"strings"

	odigosclientset "github.com/odigos-io/odigos/api/generated/odigos/clientset/versioned"
	v1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func IsPodInCurrentNode(pod *corev1.Pod) bool {
	return pod.Spec.NodeName == env.Current.NodeName
}

func GetRunningPods(ctx context.Context, labels map[string]string, ns string, kubeClient client.Client) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := kubeClient.List(ctx, &podList, client.MatchingLabels(labels), client.InNamespace(ns))

	var filteredPods []corev1.Pod
	for _, pod := range podList.Items {
		if IsPodInCurrentNode(&pod) && pod.Status.Phase == corev1.PodRunning && pod.DeletionTimestamp == nil {
			filteredPods = append(filteredPods, pod)
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredPods, nil
}

func GetDestinations(ctx context.Context, odigosKubeClient *odigosclientset.Clientset, namespace string) (*v1alpha1.DestinationList, error) {
	destinations, err := odigosKubeClient.OdigosV1alpha1().Destinations(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return destinations, nil
}

func GetResourceAttributes(workload *common.PodWorkload, podName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(workload.Namespace),
		semconv.K8SPodName(podName),
	}

	switch workload.Kind {
	case "Deployment":
		attrs = append(attrs, semconv.K8SDeploymentName(workload.Name))
	case "StatefulSet":
		attrs = append(attrs, semconv.K8SStatefulSetName(workload.Name))
	case "DaemonSet":
		attrs = append(attrs, semconv.K8SDaemonSetName(workload.Name))
	}

	return attrs
}

func GetWorkloadNameFromOwnerReference(ownerReference metav1.OwnerReference) (string, string, error) {
	name := ownerReference.Name
	kind := ownerReference.Kind
	if kind == "ReplicaSet" {
		// ReplicaSet name is in the format <deployment-name>-<random-string>
		hyphenIndex := strings.LastIndex(name, "-")
		if hyphenIndex == -1 {
			// It is possible for a user to define a bare ReplicaSet without a deployment, currently not supporting this
			return "", "", errors.New("replicaset name does not contain a hyphen")
		}
		// Extract deployment name from ReplicaSet name
		return name[:hyphenIndex], "Deployment", nil
	} else if kind == "DaemonSet" || kind == "Deployment" || kind == "StatefulSet" {
		return name, kind, nil
	}
	return "", "", fmt.Errorf("kind %s not supported", kind)
}
