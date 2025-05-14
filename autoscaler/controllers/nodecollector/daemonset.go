package nodecollector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	containerName                    = "data-collection"
	containerCommand                 = "/odigosotelcol"
	confDir                          = "/conf"
	configHashAnnotation             = "odigos.io/config-hash"
	logsSymlinkTargetMountVolumeName = "logs-symlink-target"
)

var (
	NodeCollectorsLabels = map[string]string{
		k8sconsts.OdigosCollectorRoleLabel: string(k8sconsts.CollectorsRoleNodeCollector),
	}
)

type DelayManager struct {
	mu         sync.Mutex
	inProgress bool
}

// RunSyncDaemonSetWithDelayAndSkipNewCalls runs the function with the specified delay and skips new calls until the function execution is finished
func (dm *DelayManager) RunSyncDaemonSetWithDelayAndSkipNewCalls(delay time.Duration, retries int,
	collection *odigosv1.CollectorsGroup, ctx context.Context, c client.Client, scheme *runtime.Scheme, secrets []string, version string) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// Skip new calls if the function is already in progress
	if dm.inProgress {
		return
	}

	dm.inProgress = true

	// Finish the function execution after the delay
	time.AfterFunc(delay, func() {
		var err error
		logger := log.FromContext(ctx)

		dm.mu.Lock()
		defer dm.mu.Unlock()
		defer dm.finishProgress()
		defer func() {
			statusPatchString := common.GetCollectorsGroupDeployedConditionsPatch(err)
			statusErr := c.Status().Patch(ctx, collection, client.RawPatch(types.MergePatchType, []byte(statusPatchString)))
			if statusErr != nil {
				logger.Error(statusErr, "Failed to patch collectors group status")
				// just log the error, do not fail the reconciliation
			}
		}()

		for i := 0; i < retries; i++ {
			_, err = syncDaemonSet(ctx, collection, c, scheme, secrets, version)
			if err == nil {
				return
			}
		}

		log.FromContext(ctx).Error(err, "Failed to sync DaemonSet")
	})
}

func (dm *DelayManager) finishProgress() {
	dm.inProgress = false
}

func syncDaemonSet(ctx context.Context, datacollection *odigosv1.CollectorsGroup,
	c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) (*appsv1.DaemonSet, error) {
	logger := log.FromContext(ctx)

	odigletDaemonsetPodSpec, err := getOdigletDaemonsetPodSpec(ctx, c, datacollection.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get Odiglet DaemonSet")
		return nil, err
	}

	configMap, err := getConfigMap(ctx, c, datacollection.Namespace)
	if err != nil {
		logger.Error(err, "Failed to get Config Map data")
		return nil, err
	}

	otelcolConfigContent := configMap.Data[k8sconsts.OdigosNodeCollectorConfigMapKey]
	signals, err := getSignalsFromOtelcolConfig(otelcolConfigContent)
	if err != nil {
		logger.Error(err, "Failed to get signals from otelcol config")
		return nil, err
	}
	desiredDs, err := getDesiredDaemonSet(datacollection, scheme, imagePullSecrets, odigosVersion, odigletDaemonsetPodSpec)
	if err != nil {
		logger.Error(err, "Failed to get desired DaemonSet")
		return nil, err
	}

	existing := &appsv1.DaemonSet{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: datacollection.Namespace, Name: datacollection.Name}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Creating DaemonSet")
			if err := c.Create(ctx, desiredDs); err != nil {
				logger.Error(err, "Failed to create DaemonSet")
				return nil, err
			}
			err = common.UpdateCollectorGroupReceiverSignals(ctx, c, datacollection, signals)
			if err != nil {
				logger.Error(err, "Failed to update node collectors group received signals")
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

	err = common.UpdateCollectorGroupReceiverSignals(ctx, c, datacollection, signals)
	if err != nil {
		logger.Error(err, "Failed to update node collectors group received signals")
		return nil, err
	}

	return updated, nil
}

func getOdigletDaemonsetPodSpec(ctx context.Context, c client.Client, namespace string) (*corev1.PodSpec, error) {
	odigletDaemonset := &appsv1.DaemonSet{}

	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: k8sconsts.OdigletDaemonSetName}, odigletDaemonset); err != nil {
		return nil, err
	}

	return &odigletDaemonset.Spec.Template.Spec, nil
}

func getDesiredDaemonSet(datacollection *odigosv1.CollectorsGroup,
	scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string,
	odigletDaemonsetPodSpec *corev1.PodSpec) (*appsv1.DaemonSet, error) {
	// TODO(edenfed): add log volumes only if needed according to apps or dests

	// 50% of the nodes can be unavailable during the update.
	// if we do not set it, the default value is 1.
	// 1 means that if 1 daemonset pod fails to update, the whole rollout will be broken.
	// this can happen when a single node has memory pressure, scheduling issues, not enough resources, etc.
	// by setting it to 50% we can tolerate more failures and the rollout will be more stable.
	maxUnavailable := intstr.FromString("50%")
	// maxSurge is the number of pods that can be created above the desired number of pods.
	// we do not want more then 1 datacollection pod on the same node as they need to bind to oltp ports.
	rollingUpdate := &appsv1.RollingUpdateDaemonSet{
		MaxUnavailable: &maxUnavailable,
	}
	// maxSurge was added to the Kubernetes api at version 1.21.alpha1, we want to be sure so we used 1.22 for the check, the fallback is without it
	if feature.DaemonSetUpdateSurge(feature.Beta) {
		maxSurge := intstr.FromInt(0)
		rollingUpdate.MaxSurge = &maxSurge
	}

	resourceMemoryRequestQuantity := resource.MustParse(fmt.Sprintf("%dMi", datacollection.Spec.ResourcesSettings.MemoryRequestMiB))
	resourceMemoryLimitQuantity := resource.MustParse(fmt.Sprintf("%dMi", datacollection.Spec.ResourcesSettings.MemoryLimitMiB))
	resourceCpuRequestQuantity := resource.MustParse(fmt.Sprintf("%dm", datacollection.Spec.ResourcesSettings.CpuRequestMillicores))
	resourceCpuLimitQuantity := resource.MustParse(fmt.Sprintf("%dm", datacollection.Spec.ResourcesSettings.CpuLimitMillicores))

	additionalVolumes := []corev1.Volume{}
	additionalContainerVolumeMounts := []corev1.VolumeMount{}
	if datacollection.Spec.K8sNodeLogsDirectory != "" {
		additionalVolumes = append(additionalVolumes, corev1.Volume{
			Name: logsSymlinkTargetMountVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: datacollection.Spec.K8sNodeLogsDirectory,
				},
			},
		})
		additionalContainerVolumeMounts = append(additionalContainerVolumeMounts, corev1.VolumeMount{
			Name:      logsSymlinkTargetMountVolumeName,
			MountPath: datacollection.Spec.K8sNodeLogsDirectory,
			ReadOnly:  true,
		})
	}

	desiredDs := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
			Namespace: datacollection.Namespace,
			Labels:    NodeCollectorsLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: NodeCollectorsLabels,
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type:          appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: rollingUpdate,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: NodeCollectorsLabels,
				},
				Spec: corev1.PodSpec{
					NodeSelector:       odigletDaemonsetPodSpec.NodeSelector,
					Affinity:           odigletDaemonsetPodSpec.Affinity,
					Tolerations:        odigletDaemonsetPodSpec.Tolerations,
					ServiceAccountName: k8sconsts.OdigosNodeCollectorServiceAccountName,
					Volumes: append([]corev1.Volume{
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
							Name: "hostfs",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/",
								},
							},
						},
					}, additionalVolumes...),
					Containers: []corev1.Container{
						{
							Name:  containerName,
							Image: common.ControllerConfig.CollectorImage,
							Command: []string{containerCommand, fmt.Sprintf("--config=%s:%s/%s/%s",
								k8sconsts.OdigosCollectorConfigMapProviderScheme,
								datacollection.Namespace,
								k8sconsts.OdigosNodeCollectorConfigMapName,
								k8sconsts.OdigosNodeCollectorConfigMapKey),
							},
							VolumeMounts: append([]corev1.VolumeMount{
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
									Name:      "hostfs",
									MountPath: "/hostfs",
									ReadOnly:  true,
								},
							}, additionalContainerVolumeMounts...),
							Env: []corev1.EnvVar{
								{
									Name: k8sconsts.NodeNameEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name: "POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.name",
										},
									},
								},
								{
									Name:  "GOMEMLIMIT",
									Value: fmt.Sprintf("%dMiB", datacollection.Spec.ResourcesSettings.GomemlimitMiB),
								},
								{
									// let the Go runtime know how many CPUs are available,
									// without this, Go will assume all the cores are available.
									Name: "GOMAXPROCS",
									ValueFrom: &corev1.EnvVarSource{
										ResourceFieldRef: &corev1.ResourceFieldSelector{
											ContainerName: containerName,
											// limitCPU, Kubernetes automatically rounds up the value to an integer
											// (700m -> 1, 1200m -> 2)
											Resource: "limits.cpu",
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
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resourceMemoryRequestQuantity,
									corev1.ResourceCPU:    resourceCpuRequestQuantity,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resourceMemoryLimitQuantity,
									corev1.ResourceCPU:    resourceCpuLimitQuantity,
								},
							},
							SecurityContext: &corev1.SecurityContext{
								Privileged: boolPtr(true),
							},
						},
					},
					DNSPolicy:         corev1.DNSClusterFirstWithHostNet,
					PriorityClassName: "system-node-critical",
				},
			},
		},
	}

	// if the service internal traffic policy is not enabled, the datacollection pod should be host networked
	if !feature.ServiceInternalTrafficPolicy(feature.GA) {
		desiredDs.Spec.Template.Spec.HostNetwork = true
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

func boolPtr(b bool) *bool {
	return &b
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
