package datacollection

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/autoscaler/utils"

	"github.com/odigos-io/odigos/autoscaler/controllers/datacollection/custom"
	"k8s.io/apimachinery/pkg/util/intstr"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
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
	containerImage       = "keyval/odigos-collector"
	containerCommand     = "/odigosotelcol"
	confDir              = "/conf"
	configHashAnnotation = "odigos.io/config-hash"
	dataCollectionSA     = "odigos-data-collection"
	odigletDaemonSetName = "odiglet"
)

var (
	commonLabels = map[string]string{
		collectorLabel: "true",
	}
)

func getOdigletDaemonsetPodSpec(ctx context.Context, c client.Client, namespace string) (*corev1.PodSpec, error) {
	odigletDaemonset := &appsv1.DaemonSet{}

	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: odigletDaemonSetName}, odigletDaemonset); err != nil {
		return nil, err
	}

	return &odigletDaemonset.Spec.Template.Spec, nil
}

func syncDaemonSet(apps *odigosv1.InstrumentedApplicationList, dests *odigosv1.DestinationList, datacollection *odigosv1.CollectorsGroup, configData string, ctx context.Context,
	c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) (*appsv1.DaemonSet, error) {
	logger := log.FromContext(ctx)

	odigletDaemonsetPodSpec, err := getOdigletDaemonsetPodSpec(ctx, c, datacollection.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get Odiglet DaemonSet")
		return nil, err
	}

	desiredDs, err := getDesiredDaemonSet(datacollection, configData, scheme, imagePullSecrets, odigosVersion, odigletDaemonsetPodSpec)
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

func getDesiredDaemonSet(datacollection *odigosv1.CollectorsGroup, configData string,
	scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string,
	odigletDaemonsetPodSpec *corev1.PodSpec,
) (*appsv1.DaemonSet, error) {
	// TODO(edenfed): add log volumes only if needed according to apps or dests

	// 50% of the nodes can be unavailable during the update.
	// if we do not set it, the default value is 1.
	// 1 means that if 1 daemonset pod fails to update, the whole rollout will be broken.
	// this can happen when a single node has memory pressure, scheduling issues, not enough resources, etc.
	// by setting it to 50% we can tolerate more failures and the rollout will be more stable.
	maxUnavailable := intstr.FromString("50%")
	// maxSurge is the number of pods that can be created above the desired number of pods.
	// we do not want more then 1 datacollection pod on the same node as they need to bind to oltp ports.
	maxSurge := intstr.FromInt(0)

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
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &maxUnavailable,
					MaxSurge:       &maxSurge,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: commonLabels,
					Annotations: map[string]string{
						configHashAnnotation: common.Sha256Hash(configData),
					},
				},
				Spec: corev1.PodSpec{
					NodeSelector:       odigletDaemonsetPodSpec.NodeSelector,
					Affinity:           odigletDaemonsetPodSpec.Affinity,
					Tolerations:        odigletDaemonsetPodSpec.Tolerations,
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
						{
							Name: "kubeletpodresources",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/pod-resources",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    containerName,
							Image:   utils.GetCollectorContainerImage(containerImage, odigosVersion),
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
								{
									Name:      "kubeletpodresources",
									MountPath: "/var/lib/kubelet/pod-resources",
									ReadOnly:  true,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "NODE_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
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
							SecurityContext: &corev1.SecurityContext{
								Privileged: boolPtr(true),
							},
						},
					},
					HostNetwork: true,
					DNSPolicy:   corev1.DNSClusterFirstWithHostNet,
				},
			},
		},
	}

	if len(imagePullSecrets) > 0 {
		desiredDs.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{}
		for _, secret := range imagePullSecrets {
			desiredDs.Spec.Template.Spec.ImagePullSecrets = append(desiredDs.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
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

func boolPtr(b bool) *bool {
	return &b
}
