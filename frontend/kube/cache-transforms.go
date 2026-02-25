package kube

import (
	"fmt"
	"maps"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func podsTransformFunc(obj interface{}) (interface{}, error) {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return nil, fmt.Errorf("expected a Pod, got %T", obj)
	}

	// Strip unnecessary fields to reduce memory usage.
	// Keep only fields needed for computing CachedPod in loader.go and status calculations.
	minimalContainers := make([]corev1.Container, len(pod.Spec.Containers))
	for i, c := range pod.Spec.Containers {
		relevantEnvVars := make([]corev1.EnvVar, 0, 1)
		for _, env := range c.Env {
			if env.Name == k8sconsts.OdigosEnvVarDistroName {
				relevantEnvVars = append(relevantEnvVars, env)
				break
			}
		}
		minimalContainers[i] = corev1.Container{
			Name:      c.Name,
			Env:       relevantEnvVars,
			Resources: c.Resources,
		}
	}

	annotations := map[string]string{}
	labels := pod.Labels
	if workload.IsStaticPod(pod) {
		annotations = pod.Annotations
		labels = maps.Clone(labels)
		labels[k8sconsts.OdigosVirtualStaticPodNameLabel] = pod.Name // this is a virtual, computed value that lives only in the cache
	}

	minimalPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         pod.Namespace,
			Name:              pod.Name,
			CreationTimestamp: pod.CreationTimestamp,
			Annotations:       annotations,
			Labels:            labels,              // to follow the selector and get pods from workload
			OwnerReferences:   pod.OwnerReferences, // needed to discover the workload from the pod
		},
		Spec: corev1.PodSpec{
			NodeName:   pod.Spec.NodeName,
			Containers: minimalContainers,
		},
		Status: corev1.PodStatus{
			ContainerStatuses:     pod.Status.ContainerStatuses,
			InitContainerStatuses: pod.Status.InitContainerStatuses,
		},
	}

	return minimalPod, nil
}

func deploymentsTransformFunc(obj interface{}) (interface{}, error) {
	deployment, ok := obj.(*appsv1.Deployment)
	if !ok {
		return nil, fmt.Errorf("expected a Deployment, got %T", obj)
	}

	// copy just what we need from the deployment
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: deployment.Namespace,
			Name:      deployment.Name,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: deployment.Spec.Selector,
		},
		Status: deployment.Status,
	}, nil
}

func daemonsetsTransformFunc(obj interface{}) (interface{}, error) {
	daemonset, ok := obj.(*appsv1.DaemonSet)
	if !ok {
		return nil, fmt.Errorf("expected a DaemonSet, got %T", obj)
	}

	// copy just what we need from the daemonset
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: daemonset.Namespace,
			Name:      daemonset.Name,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: daemonset.Spec.Selector,
		},
		Status: daemonset.Status,
	}, nil
}

func statefulsetsTransformFunc(obj interface{}) (interface{}, error) {
	statefulset, ok := obj.(*appsv1.StatefulSet)
	if !ok {
		return nil, fmt.Errorf("expected a StatefulSet, got %T", obj)
	}

	// copy just what we need from the statefulset
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: statefulset.Namespace,
			Name:      statefulset.Name,
		},
		Spec: appsv1.StatefulSetSpec{
			Selector: statefulset.Spec.Selector,
		},
		Status: statefulset.Status,
	}, nil
}

func cronjobsTransformFunc(obj interface{}) (interface{}, error) {
	cronjob, ok := obj.(*batchv1.CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob, got %T", obj)
	}

	// copy just what we need from the cronjob
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: cronjob.Namespace,
			Name:      cronjob.Name,
		},
		Spec: batchv1.CronJobSpec{ // keep just the selector which is all we use.
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Selector: cronjob.Spec.JobTemplate.Spec.Selector,
				},
			},
		},
		Status: cronjob.Status,
	}, nil
}

func argoRolloutsTransformFunc(obj interface{}) (interface{}, error) {
	rollout, ok := obj.(*argorolloutsv1alpha1.Rollout)
	if !ok {
		return nil, fmt.Errorf("expected a Rollout, got %T", obj)
	}

	// copy just what we need from the rollout
	return &argorolloutsv1alpha1.Rollout{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: rollout.Namespace,
			Name:      rollout.Name,
		},
		Spec: argorolloutsv1alpha1.RolloutSpec{
			Selector: rollout.Spec.Selector,
		},
		Status: rollout.Status,
	}, nil
}

func deploymentConfigsTransformFunc(obj interface{}) (interface{}, error) {
	deploymentConfig, ok := obj.(*openshiftappsv1.DeploymentConfig)
	if !ok {
		return nil, fmt.Errorf("expected a DeploymentConfig, got %T", obj)
	}

	// copy just what we need from the deployment config
	return &openshiftappsv1.DeploymentConfig{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: deploymentConfig.Namespace,
			Name:      deploymentConfig.Name,
		},
		Spec: openshiftappsv1.DeploymentConfigSpec{
			Selector: deploymentConfig.Spec.Selector,
		},
		Status: deploymentConfig.Status,
	}, nil
}
