package controllers

import (
	"context"
	odigosv1 "github.com/keyval-dev/odigos/api/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func (r *CollectorReconciler) syncPods(ctx context.Context, collector *odigosv1.Collector) (bool, error) {
	podList, err := r.listPods(ctx, collector)
	if err != nil {
		return false, err
	}

	if len(podList.Items) == 0 {
		err = r.createPods(ctx, collector)
		if err != nil {
			return false, err
		}
		return true, nil
	}

	if !r.isPodsUpToDate(podList) {
		err = r.updatePods(ctx, podList)
		if err != nil {
			return false, err
		}

		return true, nil
	}

	return false, nil
}

func (r *CollectorReconciler) isPodsUpToDate(podList *v1.PodList) bool {
	for _, pod := range podList.Items {
		collectorVolfound := false
		for _, vol := range pod.Spec.Volumes {
			if vol.Name == "collector-conf" {
				collectorVolfound = true
				break
			}
		}

		if !collectorVolfound || len(pod.Spec.Containers) != 1 {
			return false
		}

		volMountFound := false
		for _, volMount := range pod.Spec.Containers[0].VolumeMounts {
			if volMount.Name == "collector-conf" {
				volMountFound = true
				break
			}
		}

		return volMountFound
	}

	return true
}

func (r *CollectorReconciler) updatePods(ctx context.Context, podList *v1.PodList) error {
	// Pods cannot be updated, delete the bad pods, they will be recreated in the next reconcile.
	for _, pod := range podList.Items {
		err := r.Delete(ctx, &pod)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *CollectorReconciler) createPods(ctx context.Context, collector *odigosv1.Collector) error {
	destList, err := r.listDestinations(ctx, collector)
	if err != nil {
		return err
	}

	img := r.getCollectorContainerImage(ctx, collector, destList)
	cmd := "/otelcol"
	if strings.Contains(img, "contrib") {
		cmd = "/otelcol-contrib"
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      collector.Name,
			Namespace: collector.Namespace,
			Labels:    commonLabels,
		},
		Spec: v1.PodSpec{
			Volumes: []v1.Volume{
				{
					Name: "collector-conf",
					VolumeSource: v1.VolumeSource{
						ConfigMap: &v1.ConfigMapVolumeSource{
							LocalObjectReference: v1.LocalObjectReference{
								Name: collector.Name,
							},
							Items: []v1.KeyToPath{
								{
									Key:  "collector-conf",
									Path: "collector-conf.yaml",
								},
							},
						},
					},
				},
			},
			Containers: []v1.Container{
				{
					Name:    "collector",
					Image:   img,
					Command: []string{cmd, "--config=/conf/collector-conf.yaml"},
					EnvFrom: r.getSecretsFromDests(destList),
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "collector-conf",
							MountPath: "/conf",
						},
					},
				},
			},
		},
	}

	err = ctrl.SetControllerReference(collector, pod, r.Scheme)
	if err != nil {
		return err
	}

	err = r.Create(ctx, pod)
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	return nil
}

func (r *CollectorReconciler) getSecretsFromDests(destList *odigosv1.DestinationList) []v1.EnvFromSource {
	var result []v1.EnvFromSource
	for _, dst := range destList.Items {
		result = append(result, v1.EnvFromSource{
			SecretRef: &v1.SecretEnvSource{
				LocalObjectReference: v1.LocalObjectReference{
					Name: dst.Spec.SecretRef.Name,
				},
			},
		})
	}

	return result
}

func (r *CollectorReconciler) listPods(ctx context.Context, collector *odigosv1.Collector) (*v1.PodList, error) {
	var podList v1.PodList
	err := r.List(ctx, &podList, client.InNamespace(collector.Namespace), client.MatchingFields{ownerKey: collector.Name})
	if err != nil {
		return nil, err
	}

	return &podList, nil
}

func (r *CollectorReconciler) getCollectorContainerImage(ctx context.Context, collector *odigosv1.Collector, destList *odigosv1.DestinationList) string {
	for _, dst := range destList.Items {
		if dst.Spec.Type == odigosv1.DatadogDestinationType {
			// TODO: Use more minimal image that contains only datadog exporter
			return "otel/opentelemetry-collector-contrib:0.53.0"
		}
	}

	return "otel/opentelemetry-collector:0.53.0"
}
