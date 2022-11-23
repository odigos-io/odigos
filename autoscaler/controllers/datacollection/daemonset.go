package datacollection

import (
	"context"
	"fmt"
	"github.com/keyval-dev/odigos/autoscaler/controllers/datacollection/custom"
	"k8s.io/apimachinery/pkg/util/intstr"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/autoscaler/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	collectorLabel       = "odigos.io/data-collection"
	containerName        = "data-collection"
	containerImage       = "otel/opentelemetry-collector-contrib:0.55.0"
	containerCommand     = "/otelcol-contrib"
	confDir              = "/conf"
	configHashAnnotation = "odigos.io/config-hash"
	dataCollectionSA     = "odigos-data-collection"
)

var (
	commonLabels = map[string]string{
		collectorLabel: "true",
	}
)

func syncDaemonSet(apps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, datacollection *odigosv1.CollectorsGroup, configData string, ctx context.Context,
	c client.Client, scheme *runtime.Scheme) (*appsv1.DaemonSet, error) {
	logger := log.FromContext(ctx)
	desiredDs, err := getDesiredDaemonSet(datacollection, configData, scheme)
	if err != nil {
		logger.Error(err, "Failed to get desired DaemonSet")
		return nil, err
	}

	if custom.ShouldApplyCustomDataCollection(dests) {
		custom.ApplyCustomChangesToDaemonSet(desiredDs, dests)
	}

	existing := &appsv1.DaemonSet{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: datacollection.Namespace, Name: datacollection.Name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating DaemonSet")
			if err := c.Create(ctx, desiredDs); err != nil {
				logger.Error(err, "Failed to create DaemonSet")
				return nil, err
			}
			return desiredDs, nil
		} else {
			logger.Error(err, "Failed to get DaemonSet")
			return nil, err
		}
	}

	logger.V(0).Info("Patching DaemonSet")
	updated, err := patchDaemonSet(existing, desiredDs, ctx, c)
	if err != nil {
		logger.Error(err, "Failed to patch DaemonSet")
		return nil, err
	}

	return updated, nil
}

func getDesiredDaemonSet(datacollection *odigosv1.CollectorsGroup, configData string, scheme *runtime.Scheme) (*appsv1.DaemonSet, error) {
	// TODO(edenfed): add log volumes only if needed according to apps or dests
	desiredDs := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      datacollection.Name,
			Namespace: datacollection.Namespace,
			Labels:    commonLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: commonLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: commonLabels,
					Annotations: map[string]string{
						configHashAnnotation: common.Sha256Hash(configData),
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: dataCollectionSA,
					Volumes: []corev1.Volume{
						{
							Name: configKey,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: datacollection.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  configKey,
											Path: fmt.Sprintf("%s.yaml", configKey),
										},
									},
								},
							},
						},
						{
							Name: "varlog",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/log",
								},
							},
						},
						{
							Name: "varlibdockercontainers",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/docker/containers",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    containerName,
							Image:   containerImage,
							Command: []string{containerCommand, fmt.Sprintf("--config=%s/%s.yaml", confDir, configKey)},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      configKey,
									MountPath: confDir,
								},
								{
									Name:      "varlibdockercontainers",
									MountPath: "/var/lib/docker/containers",
									ReadOnly:  true,
								},
								{
									Name:      "varlog",
									MountPath: "/var/log",
									ReadOnly:  true,
								},
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
							},
						},
					},
					HostNetwork: true,
					DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
				},
			},
		},
	}

	err := ctrl.SetControllerReference(datacollection, desiredDs, scheme)
	if err != nil {
		return nil, err
	}

	return desiredDs, nil
}

func patchDaemonSet(existing *appsv1.DaemonSet, desired *appsv1.DaemonSet, ctx context.Context, c client.Client) (*appsv1.DaemonSet, error) {
	updated := existing.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = map[string]string{}
	}
	if updated.Labels == nil {
		updated.Labels = map[string]string{}
	}

	updated.Spec = desired.Spec
	updated.ObjectMeta.OwnerReferences = desired.ObjectMeta.OwnerReferences
	for k, v := range desired.ObjectMeta.Annotations {
		updated.ObjectMeta.Annotations[k] = v
	}
	for k, v := range desired.ObjectMeta.Labels {
		updated.ObjectMeta.Labels[k] = v
	}

	patch := client.MergeFrom(existing)
	if err := c.Patch(ctx, updated, patch); err != nil {
		return nil, err
	}

	return updated, nil
}
