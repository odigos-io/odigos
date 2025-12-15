package clustercollector

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"errors"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/autoscaler/k8sconfig"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
)

const (
	containerCommand     = "/odigosotelcol"
	confDir              = "/conf"
	configHashAnnotation = "odigos.io/config-hash"
)

func syncDeployment(enabledDests *odigosv1.DestinationList, gateway *odigosv1.CollectorsGroup,
	ctx context.Context, c client.Client, scheme *runtime.Scheme, odigosVersion string) (*appsv1.Deployment, error) {
	logger := log.FromContext(ctx)

	autoscalerDeployment := &appsv1.Deployment{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: gateway.Namespace, Name: k8sconsts.AutoScalerDeploymentName}, autoscalerDeployment); err != nil {
		return nil, err
	}
	autoScalerTopologySpreadConstraints := autoscalerDeployment.Spec.Template.Spec.TopologySpreadConstraints

	secretsVersionHash, err := destinationsSecretsVersionsHash(ctx, c, enabledDests)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get secrets hash"))
	}

	// Use the hash of the secrets  to make sure the gateway will restart when the secrets (mounted as environment variables) changes
	configDataHash := commonconfig.Sha256Hash(secretsVersionHash)
	desiredDeployment, err := getDesiredDeployment(ctx, c, enabledDests, configDataHash, gateway,
		scheme, odigosVersion, autoScalerTopologySpreadConstraints)
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

func getDesiredDeployment(ctx context.Context, c client.Client, enabledDests *odigosv1.DestinationList, configDataHash string,
	gateway *odigosv1.CollectorsGroup, scheme *runtime.Scheme, odigosVersion string, topologySpreadConstraints []corev1.TopologySpreadConstraint) (*appsv1.Deployment, error) {

	nodeSelector := gateway.Spec.NodeSelector
	if nodeSelector == nil {
		nodeSelector = &map[string]string{}
	}

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

	extraEnvVars := []corev1.EnvVar{}
	if gateway.Spec.HttpsProxyAddress != nil {
		odigosNs := env.GetCurrentNamespace()
		extraEnvVars = append(extraEnvVars, corev1.EnvVar{
			Name:  "HTTPS_PROXY",
			Value: *gateway.Spec.HttpsProxyAddress,
		}, corev1.EnvVar{
			// prevent the own telemetry metrics from using the https proxy if set.
			// gRPC uses the HTTPS_PROXY even for non tls connections
			// since it's always uses HTTP CONNECT, so we need to blacklist the ui service.
			Name:  "NO_PROXY",
			Value: fmt.Sprintf("%s.%s:%d", k8sconsts.UIServiceName, odigosNs, odigosconsts.OTLPPort),
		})
	}

	desiredDeployment := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{
			Name:      k8sconsts.OdigosClusterCollectorDeploymentName,
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
					NodeSelector:       *nodeSelector,
					ServiceAccountName: k8sconsts.OdigosClusterCollectorDeploymentName,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: boolPtr(true),
						RunAsUser:    int64Ptr(65534), // nobody user
						RunAsGroup:   int64Ptr(65534), // nobody group
						FSGroup:      int64Ptr(65534),
					},
					Containers: []corev1.Container{
						{
							Name:    k8sconsts.OdigosClusterCollectorContainerName,
							Image:   commonconfig.ControllerConfig.CollectorImage,
							Command: []string{containerCommand},
							Args: []string{fmt.Sprintf("--config=%s:%s/%s/%s",
								k8sconsts.OdigosCollectorConfigMapProviderScheme,
								gateway.Namespace,
								k8sconsts.OdigosClusterCollectorConfigMapName,
								k8sconsts.OdigosClusterCollectorConfigMapKey),
							},
							EnvFrom: getSecretsFromDests(enabledDests),
							// Add the ODIGOS_VERSION environment variable from the ConfigMap
							Env: append([]corev1.EnvVar{
								{
									Name: odigosconsts.OdigosVersionEnvVarName,
									ValueFrom: &corev1.EnvVarSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: k8sconsts.OdigosDeploymentConfigMapName,
											},
											Key: k8sconsts.OdigosDeploymentConfigMapVersionKey,
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
								{
									// let the Go runtime know how many CPUs are available,
									// without this, Go will assume all the cores are available.
									Name: "GOMAXPROCS",
									ValueFrom: &corev1.EnvVarSource{
										ResourceFieldRef: &corev1.ResourceFieldSelector{
											ContainerName: k8sconsts.OdigosClusterCollectorContainerName,
											// limitCPU, Kubernetes automatically rounds up the value to an integer
											// (700m -> 1, 1200m -> 2)
											Resource: "limits.cpu",
										},
									},
								},
							}, extraEnvVars...),
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
								FailureThreshold: 3,
								PeriodSeconds:    10,
								SuccessThreshold: 1,
								TimeoutSeconds:   5,
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt(13133),
									},
								},
								FailureThreshold: 3,
								PeriodSeconds:    10,
								SuccessThreshold: 1,
								TimeoutSeconds:   5,
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

	k8sConfigers := k8sconfig.LoadK8sConfigers()
	for _, dest := range enabledDests.Items {
		if k8sConfiger, exists := k8sConfigers[dest.GetType()]; exists {
			err := k8sConfiger.ModifyGatewayCollectorDeployment(ctx, c, dest, desiredDeployment)
			if err != nil {
				return nil, errors.Join(err, errors.New("failed to modify gateway collector deployment"))
			}
		}
	}

	odigosConfiguration, err := k8sutils.GetCurrentOdigosConfiguration(ctx, c)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get current odigos configuration"))
	}

	if len(odigosConfiguration.ImagePullSecrets) > 0 {
		desiredDeployment.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{}
		for _, secret := range odigosConfiguration.ImagePullSecrets {
			desiredDeployment.Spec.Template.Spec.ImagePullSecrets = append(desiredDeployment.Spec.Template.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
		}
	}

	if topologySpreadConstraints != nil && len(topologySpreadConstraints) > 0 {
		adjusted := make([]corev1.TopologySpreadConstraint, 0, len(topologySpreadConstraints))
		for _, c := range topologySpreadConstraints {
			c.LabelSelector = &metav1.LabelSelector{
				MatchLabels: ClusterCollectorGateway,
			}
			adjusted = append(adjusted, c)
		}
		desiredDeployment.Spec.Template.Spec.TopologySpreadConstraints = adjusted
	}

	if odigosConfiguration.ClickhouseJsonTypeEnabledProperty != nil && *odigosConfiguration.ClickhouseJsonTypeEnabledProperty {
		desiredDeployment.Spec.Template.Spec.Containers[0].Args = append(
			desiredDeployment.Spec.Template.Spec.Containers[0].Args,
			"--feature-gates=clickhouse.json",
		)
	}

	err = ctrl.SetControllerReference(gateway, desiredDeployment, scheme)
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
