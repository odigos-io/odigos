package cache

import (
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StripAnnotation is used to keep only the relevant annotations for us before saving an object in a cache.
// Objects can have a large amount of annotations with large values,
// currently only keeping annotations relating to kubernetes.io or odigos.io
func StripAnnotations(obj metav1.Object) {
	currentAnnotations := obj.GetAnnotations()
	newAnnotations := make(map[string]string)

	for k, v := range currentAnnotations {
		if strings.Contains(k, "kubernetes.io") {
			newAnnotations[k] = v
		}
		if strings.Contains(k, "odigos.io") {
			newAnnotations[k] = v
		}
	}
	obj.SetAnnotations(newAnnotations)
}

// StripPod removes un-relevant fields such as container statuses fields that we don't care about
// (Resources, Volumes), and in addition removing un-used annotations.
// This should be used inside a cache transform function to reduce the memory overhead of saving pods in a cache.
func StripPod(p *v1.Pod) {
	stripContainerStatuses(p.Status.ContainerStatuses)
	stripContainerStatuses(p.Status.InitContainerStatuses)
	stripContainerStatuses(p.Status.EphemeralContainerStatuses)

	StripAnnotations(p)
}

// StripWorkloadSpecTemplate remove un-relevant fields such as annotations, full container specs, resources etc'.
// This should be used inside a cache transform function to reduce the memory overhead of saving deployments/statefulsets/daemonsets in a cache.
func StripWorkloadSpecTemplate(o client.Object) {
	StripAnnotations(o)
	var template *v1.PodTemplateSpec
	switch t := o.(type) {
	case *appsv1.Deployment:
		template = &t.Spec.Template
	case *appsv1.StatefulSet:
		template = &t.Spec.Template
	case *appsv1.DaemonSet:
		template = &t.Spec.Template
	}

	if template == nil {
		return
	}

	StripAnnotations(template)
	currentContainers := template.Spec.Containers
	if len(currentContainers) > 0 {
		minimalContainers := make([]v1.Container, len(currentContainers))
		for i := range currentContainers {
			// for each container keep only its name and probes
			// probes are used for auto-head-sampling feature
			minimalContainers[i] = v1.Container{
				Name:           currentContainers[i].Name,
				StartupProbe:   currentContainers[i].StartupProbe,
				LivenessProbe:  currentContainers[i].LivenessProbe,
				ReadinessProbe: currentContainers[i].ReadinessProbe,
			}
		}
		template.Spec = v1.PodSpec{
			Containers: minimalContainers,
		}
	} else {
		template.Spec = v1.PodSpec{}
	}
}

func stripContainerStatuses(containerStatuses []v1.ContainerStatus) {
	for i := range containerStatuses {
		containerStatuses[i].AllocatedResources = nil
		containerStatuses[i].Resources = nil
		containerStatuses[i].VolumeMounts = nil
		containerStatuses[i].ImageID = ""
		containerStatuses[i].ContainerID = ""
		containerStatuses[i].Image = ""
	}
}
