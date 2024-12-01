package gateway

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"errors"

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
	ctx context.Context, c client.Client, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	secretsVersionHash, err := destinationsSecretsVersionsHash(ctx, c, dests)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get secrets hash"))
	}

	// Calculate the hash of the config data and the secrets version hash, this is used to make sure the gateway will restart when the config changes
	configDataHash := common.Sha256Hash(fmt.Sprintf("%s-%s", configData, secretsVersionHash))
	desiredDeployment, err := getDesiredDeployment(dests, configDataHash, gateway, scheme, imagePullSecrets, odigosVersion)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get desired deployment"))
	}

	existingDeployment := &appsv1.Deployment{}
	getError := c.Get(ctx, client.ObjectKey{Name: gateway.Name, Namespace: gateway.Namespace}, existingDeployment)
	if getError != nil && !apierrors.IsNotFound(getError) {
		return nil, errors.Join(getError, errors.New("failed to get gateway deployment"))
	}

	if apierrors.IsNotFound(getError) {
		logger.V(0).Info("Creating new gateway deployment")
		err := c.Create(ctx, desiredDeployment)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to create gateway deployment"))
		}
		return desiredDeployment, nil
	} else {
		logger.V(0).Info("Patching existing gateway deployment")
		newDep, err := patchDeployment(existingDeployment, desiredDeployment, ctx, c)
		if err != nil {
			return nil, errors.Join(err, errors.New("failed to patch gateway deployment"))
		}
		return newDep, nil
	}
}

func patchDeployment(existing *appsv1.Deployment, desired *appsv1.Deployment, ctx context.Context, c client.Client) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)
	res, err := controllerutil.CreateOrPatch(ctx, c, existing, func() error {
		existing.Spec.Template = desired.Spec.Template
		return nil
	})

	if err != nil {
		return nil, err
	}

	logger.V(0).Info("Deployment patched", "result", res)
	return existing, nil
}

func getDesiredDeployment(dests *odigosv1.DestinationList, configDataHash string,
	gateway *odigosv1.CollectorsGroup, scheme *runtime.Scheme, imagePullSecrets []string, odigosVersion string) (*appsv1.Deployment, error) {

	// request + limits for memory and cpu
	requestMemoryQuantity := resource.MustParse(fmt.Sprintf("%dMi", gateway.Spec.ResourcesSettings.MemoryRequestMiB))
	limitMemoryQuantity := resource.MustParse(fmt.Sprintf("%dMi", gateway.Spec.ResourcesSettings.MemoryLimitMiB))

	requestCPU := resource.MustParse(fmt.Sprintf("%dm", gateway.Spec.ResourcesSettings.CpuRequestMillicores))
	limitCPU := resource.MustParse(fmt.Sprintf("%dm", gateway.Spec.ResourcesSettings.CpuLimitMillicores))

	// deployment replicas
	var gatewayReplicas int32 = 1
	if gateway.Spec.ResourcesSettings.MinReplicas != nil {
		gatewayReplicas = int32(*gateway.Spec.ResourcesSettings.MinReplicas)
	}

	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      consts.OdigosClusterCollectorDeploymentName,
			Namespace: gateway.Namespace,
			Labels:    ClusterCollectorGateway,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: intPtr(gatewayReplicas),
			Selector: &v1.LabelSelector{
				MatchLabels: ClusterCollectorGateway,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: ClusterCollectorGateway,
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
									Value: fmt.Sprintf("%dMiB", gateway.Spec.ResourcesSettings.GomemlimitMiB),
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
									corev1.ResourceCPU:    requestCPU,
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: limitMemoryQuantity,
									corev1.ResourceCPU:    limitCPU,
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
