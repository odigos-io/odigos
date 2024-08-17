package utils

import (
	"context"
	"errors"
	"strings"

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var errKindNotSupported = errors.New("kind not supported")

func IsErrorKindNotSupported(err error) bool {
	return err == errKindNotSupported
}

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

func GetResourceAttributes(podWorkload *workload.PodWorkload, podName string) []attribute.KeyValue {
	attrs := []attribute.KeyValue{
		semconv.K8SNamespaceName(podWorkload.Namespace),
		semconv.K8SPodName(podName),
	}

	switch podWorkload.Kind {
	case workload.WorkloadKindDeployment:
		attrs = append(attrs, semconv.K8SDeploymentName(podWorkload.Name))
	case workload.WorkloadKindStatefulSet:
		attrs = append(attrs, semconv.K8SStatefulSetName(podWorkload.Name))
	case workload.WorkloadKindDaemonSet:
		attrs = append(attrs, semconv.K8SDaemonSetName(podWorkload.Name))
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
	return "", "", errKindNotSupported
}
