package runtime_details

import (
	"context"
	"errors"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func findDeploymentNameInOwnerReferences(ownerReferences []metav1.OwnerReference) string {
	for _, owner := range ownerReferences {
		if owner.Kind == "Deployment" {
			return owner.Name
		}
	}
	return ""
}

func getGenerationFromReplicaSet(ctx context.Context, kubeClient *kubernetes.Clientset, rsName string, ns string) (int64, error) {
	rs, err := kubeClient.AppsV1().ReplicaSets(ns).Get(ctx, rsName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	deploymentRevision, found := rs.Annotations["deployment.kubernetes.io/revision"]
	if !found {
		return 0, nil
	}

	deploymentName := findDeploymentNameInOwnerReferences(rs.ObjectMeta.OwnerReferences)
	if deploymentName == "" {
		return 0, errors.New("replica set with deployment revision has no deployment owner")
	}

	deployment, err := kubeClient.AppsV1().Deployments(ns).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return 0, err
	}
	if deployment.Annotations["deployment.kubernetes.io/revision"] != deploymentRevision {
		return 0, nil
	}
	if deployment.Status.ObservedGeneration == 0 {
		return 0, errors.New("deployment with revision has no observed generation")
	}
	return deployment.Status.ObservedGeneration, nil
}

func getGenerationForDsOrSs(pod *corev1.Pod) (int64, error) {
	// the "pod-template-generation" seems to be deprecated, but still valid and populated.
	// I tried to use the "controller-revision-hash" label, find the controller revision object
	// and extract the generation from it, but it seems to be more complex and not worth the effort.
	podTemplateGeneration, ok := pod.Labels["pod-template-generation"]
	if !ok {
		return 0, errors.New("pod from ds or ss has no pod-template-generation label")
	}

	return strconv.ParseInt(podTemplateGeneration, 10, 64)
}

// Givin a pod, this function calculates it's workload generation.
// e.g. it will return the generation of the deployment/ds/ss manifest that created the pod.
// This function uses clientset and not the controller-runtime client, so not to pull these objects into the cache,
// potentially increasing the memory usage of the process.
// Avoid calling this function too often, as it will make an API call to the k8s API server.
func GetPodGeneration(ctx context.Context, kubeClient *kubernetes.Clientset, pod *corev1.Pod) (int64, error) {
	for _, owner := range pod.ObjectMeta.OwnerReferences {
		if owner.Kind == "ReplicaSet" {
			return getGenerationFromReplicaSet(ctx, kubeClient, owner.Name, pod.Namespace)
		} else if owner.Kind == "DaemonSet" || owner.Kind == "StatefulSet" {
			return getGenerationForDsOrSs(pod)
		}
	}

	// the pod is not owned by a workload odigos supports, so no need to handle it.
	return 0, nil
}
