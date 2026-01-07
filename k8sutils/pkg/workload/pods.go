package workload

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsStaticPod return true whether the pod is static or not
// https://kubernetes.io/docs/tasks/configure-pod-container/static-pod/
func IsStaticPod(p *corev1.Pod) bool {
	var nodeOwner *metav1.OwnerReference
	for _, owner := range p.OwnerReferences {
		if owner.Kind == "Node" {
			nodeOwner = &owner
			break
		}
	}

	// static pods are owned by nodes
	if nodeOwner == nil {
		return false
	}

	// https://kubernetes.io/docs/reference/labels-annotations-taints/#kubernetes-io-config-source
	// This annotation is added by the kubelet to indicate where the Pod comes from.
	// For static Pods, the annotation value could be one of file or http depending on where the Pod manifest is located.
	// For a Pod created on the API server and then scheduled to the current node, the annotation value is api.
	if p.Annotations == nil {
		return false
	}
	configSource, ok := p.Annotations["kubernetes.io/config.source"]
	if !ok {
		return false
	}
	return configSource == "file" || configSource == "http"
}

func PodUID(p *corev1.Pod) string {
	if IsStaticPod(p) {
		// https://kubernetes.io/docs/reference/labels-annotations-taints/#kubernetes-io-config-hash
		// When the kubelet creates a static Pod based on a given manifest,
		// it attaches this annotation to the static Pod. The value of the annotation is the UID of the Pod
		return p.Annotations["kubernetes.io/config.hash"]
	}

	return string(p.UID)
}

// StaticPodName returns the value of the static pod name without its node name
// since static pods name are of the form
// <static-pod-name>-<node-name>
// if the pod is not static, or its name does not match the expected pattern,
// an empty string will be returned
func StaticPodName(p *corev1.Pod) string {
	if p == nil {
		return ""
	}

	if !IsStaticPod(p) {
		return ""
	}

	nodeName := p.Spec.NodeName
	if nodeName == "" {
		return ""
	}

	staticPodName, found := strings.CutSuffix(p.Name, "-"+nodeName)
	if !found {
		return ""
	}

	return staticPodName
}
