package gateway

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/odigos-io/odigos/autoscaler/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	containerName        = "gateway"
	containerImage       = "keyval/odigos-collector"
	containerCommand     = "/odigosotelcol"
	confDir              = "/conf"
	configHashAnnotation = "odigos.io/config-hash"
)

func syncDeployment(dests *odigosv1.DestinationList, gateway *odigosv1.CollectorsGroup, configData string,
	ctx context.Context, c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string, memConfig *memoryConfigurations) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	secretsVersionHash, err := destinationsSecretsVersionsHash(ctx, c, dests)
	if err != nil {
		logger.Error(err, "Failed to get secrets hash")
		return nil, err
	}

	// Calculate the hash of the config data and the secrets version hash, this is used to make sure the gateway will restart when the config changes
	configDataHash := common.Sha256Hash(fmt.Sprintf("%s-%s", configData, secretsVersionHash))
	desiredDeployment, err := getDesiredDeployment(dests, configDataHash, gateway, scheme, imagePullSecrets, odigosVersion, memConfig)
	if err != nil {
		logger.Error(err, "Failed to get desired deployment")
		return nil, err
	}

	existing := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Name: gateway.Name, Namespace: gateway.Namespace}, existing); err != nil {
		if apierrors.IsNotFound(err) {
			logger.V(0).Info("Creating deployment")
			newDeployment, err := createDeployment(desiredDeployment, ctx, c)
			if err != nil {
				logger.Error(err, "failed to create deployment")
				return nil, err
			}
			return newDeployment, nil
		} else {
			logger.Error(err, "failed to get deployment")
			return nil, err
		}
	}

	logger.V(0).Info("Patching deployment")
	newDep, err := patchDeployment(existing, desiredDeployment, ctx, c)
	if err != nil {
		logger.Error(err, "failed to patch deployment")
		return nil, err
	}

	return newDep, nil
}

func createDeployment(desired *appsv1.Deployment, ctx context.Context, c client.Client) (*appsv1.Deployment, error) {
	if err := c.Create(ctx, desired); err != nil {
		return nil, err
	}
	return desired, nil
}

func patchDeployment(existing *appsv1.Deployment, desired *appsv1.Deployment, ctx context.Context, c client.Client) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)
	res, err := controllerutil.CreateOrPatch(ctx, c, existing, func() error {
		existing.Spec.Template = desired.Spec.Template
		return nil
	})

	if err != nil {
		logger.Error(err, "Failed to patch deployment")
		return nil, err
	}

	logger.V(0).Info("Deployment patched", "result", res)
	return existing, nil
}

func getDesiredDeployment(dests *odigosv1.DestinationList, configDataHash string,
	gateway *odigosv1.CollectorsGroup, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string, memConfig *memoryConfigurations) (*appsv1.Deployment, error) {

	requestMemoryQuantity := resource.MustParse(fmt.Sprintf("%dMi", memConfig.memoryRequestMiB))

	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorDeploymentName,
			Namespace: gateway.Namespace,
			Labels:    CommonLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: intPtr(1),
			Selector: &v1.LabelSelector{
				MatchLabels: CommonLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: CommonLabels,
					Annotations: map[string]string{
						configHashAnnotation: configDataHash,
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: consts.OdigosClusterCollectorConfigMapKey,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: gateway.Name,
									},
									Items: []corev1.KeyToPath{
										{
											Key:  consts.OdigosClusterCollectorConfigMapKey,
											Path: fmt.Sprintf("%s.yaml", consts.OdigosClusterCollectorConfigMapKey),
										},
									},
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:    containerName,
							Image:   utils.GetCollectorContainerImage(containerImage, odigosVersion),
							Command: []string{containerCommand, fmt.Sprintf("--config=%s/%s.yaml", confDir, consts.OdigosClusterCollectorConfigMapKey)},
							EnvFrom: getSecretsFromDests(dests),
							// Add the ODIGOS_VERSION environment variable from the ConfigMap
							Env: []corev1.EnvVar{
								{
									Name: "ODIGOS_VERSION",
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "odigos-deployment",
											},
											Key: "ODIGOS_VERSION",
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
									Value: fmt.Sprintf("%dMiB", memConfig.gomemlimitMiB),
								},
							},
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      consts.OdigosClusterCollectorConfigMapKey,
									MountPath: confDir,
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
									corev1.ResourceMemory: requestMemoryQuantity,
								},
							},
						},
					},
				},
			},
		},
	}

	if len(imagePullSecrets) > 0 {
		desiredDeployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{}
		for _, secret := range imagePullSecrets {
			desiredDeployment.Spec.Template.Spec.ImagePullSecrets = append(desiredDeployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
	}

	err := ctrl.SetControllerReference(gateway, desiredDeployment, scheme)
	if err != nil {
		return nil, err
	}

	return desiredDeployment, nil
}

func getSecretsFromDests(destList *odigosv1.DestinationList) []corev1.EnvFromSource {
	var result []corev1.EnvFromSource
	for _, dst := range destList.Items {
		if dst.Spec.SecretRef != nil {
			result = append(result, corev1.EnvFromSource{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: dst.Spec.SecretRef.Name,
					},
				},
			})
		}
	}

	return result
}

func boolPtr(b bool) *bool {
	return &b
}

func intPtr(n int32) *int32 {
	return &n
}

func int64Ptr(n int64) *int64 {
	return &n
}
