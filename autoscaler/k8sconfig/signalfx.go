package k8sconfig

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
)

type SignalFx struct{}

func (s *SignalFx) DestType() common.DestinationType {
	return common.SignalFxDestinationType
}

// ModifyGatewayCollectorDeployment modifies the gateway collector deployment to:
// 1. Mount the SignalFx CA certificate (either inline via secret or from an existing ConfigMap)
// 2. Mount an existing secret for the access token if configured
func (s *SignalFx) ModifyGatewayCollectorDeployment(ctx context.Context, k8sClient client.Client, dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error {
	destConfig := dest.GetConfig()

	// Find the collector container
	containerIndex := -1
	for i := 0; i < len(currentDeployment.Spec.Template.Spec.Containers); i++ {
		if currentDeployment.Spec.Template.Spec.Containers[i].Name == k8sconsts.OdigosClusterCollectorContainerName {
			containerIndex = i
			break
		}
	}
	if containerIndex == -1 {
		return fmt.Errorf("gateway collector container '%s' not found", k8sconsts.OdigosClusterCollectorContainerName)
	}

	// Mount existing secret for access token if configured
	if err := s.mountAccessTokenSecret(destConfig, currentDeployment, containerIndex); err != nil {
		return err
	}

	// Handle CA certificate mounting
	return s.mountCaCertificate(ctx, k8sClient, dest, destConfig, currentDeployment, containerIndex)
}

// mountAccessTokenSecret mounts an existing secret's key as the SIGNALFX_ACCESS_TOKEN environment variable
func (s *SignalFx) mountAccessTokenSecret(destConfig map[string]string, currentDeployment *appsv1.Deployment, containerIndex int) error {
	// Check if using existing secret for access token
	secretRefEnabled := destConfig[config.SignalfxAccessTokenSecretRefEnabled]
	if secretRefEnabled != "true" {
		return nil
	}

	secretName := destConfig[config.SignalfxAccessTokenSecretName]
	if secretName == "" {
		return fmt.Errorf("SIGNALFX_ACCESS_TOKEN_SECRET_NAME must be specified when using existing secret for access token")
	}

	secretKey := destConfig[config.SignalfxAccessTokenSecretKey]
	if secretKey == "" {
		secretKey = config.SignalfxAccessTokenDefaultKey
	}

	// Check if env var already exists
	for _, env := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env {
		if env.Name == config.SignalfxAccessTokenDefaultKey {
			// Already configured, skip
			return nil
		}
	}

	// Add environment variable from the existing secret
	envVar := corev1.EnvVar{
		Name: config.SignalfxAccessTokenDefaultKey,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: secretKey,
			},
		},
	}

	currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env = append(
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env,
		envVar,
	)

	return nil
}

// mountCaCertificate handles CA certificate mounting from either inline secret or existing ConfigMap
func (s *SignalFx) mountCaCertificate(ctx context.Context, k8sClient client.Client, dest K8sExporterConfigurer, destConfig map[string]string, currentDeployment *appsv1.Deployment, containerIndex int) error {
	// Check if CA ConfigMap is specified
	configMapName := destConfig[config.SignalfxCaConfigMapName]
	hasConfigMap := configMapName != ""

	// Check if inline CA PEM is provided via secret
	hasSecretCaPem := false
	if dest.GetSecretRef() != nil && dest.GetSecretRef().Name != "" {
		secret := &corev1.Secret{}
		err := k8sClient.Get(ctx, client.ObjectKey{
			Name:      dest.GetSecretRef().Name,
			Namespace: currentDeployment.Namespace,
		}, secret)
		if err != nil {
			return fmt.Errorf("failed to get secret %s: %w", dest.GetSecretRef().Name, err)
		}
		if secret.Data != nil {
			_, hasSecretCaPem = secret.Data[config.SignalfxCaPemKey]
		}
	}

	// If neither CA source is configured, nothing to do
	if !hasSecretCaPem && !hasConfigMap {
		return nil
	}

	// Secret-based CA PEM takes precedence over ConfigMap
	if hasSecretCaPem {
		return s.mountSecretCaCert(dest, currentDeployment, containerIndex)
	}

	// Mount from ConfigMap
	return s.mountConfigMapCaCert(destConfig, currentDeployment, containerIndex)
}

// mountSecretCaCert mounts the CA certificate from the destination's secret
func (s *SignalFx) mountSecretCaCert(dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment, containerIndex int) error {
	// Check if volume mount already exists
	volumeMountExists := false
	for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts {
		if volumeMount.Name == config.SignalfxCaSecretVolumeName {
			volumeMountExists = true
			break
		}
	}
	if !volumeMountExists {
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(
			currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts,
			corev1.VolumeMount{
				Name:      config.SignalfxCaSecretVolumeName,
				MountPath: config.SignalfxCaMountPath,
				ReadOnly:  true,
			},
		)
	}

	// Check if volume already exists
	volumeExists := false
	for i := range currentDeployment.Spec.Template.Spec.Volumes {
		if currentDeployment.Spec.Template.Spec.Volumes[i].Name == config.SignalfxCaSecretVolumeName {
			volumeExists = true
			break
		}
	}
	if !volumeExists {
		currentDeployment.Spec.Template.Spec.Volumes = append(currentDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: config.SignalfxCaSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dest.GetSecretRef().Name,
					Items: []corev1.KeyToPath{
						{
							Key:  config.SignalfxCaPemKey,
							Path: config.SignalfxCaPemKey,
						},
					},
				},
			},
		})
	}

	return nil
}

// mountConfigMapCaCert mounts the CA certificate from an existing ConfigMap
func (s *SignalFx) mountConfigMapCaCert(destConfig map[string]string, currentDeployment *appsv1.Deployment, containerIndex int) error {
	configMapName := destConfig[config.SignalfxCaConfigMapName]
	configMapKey := destConfig[config.SignalfxCaConfigMapKey]
	if configMapKey == "" {
		configMapKey = config.SignalfxCaConfigMapDefaultKey
	}

	// Check if volume mount already exists
	volumeMountExists := false
	for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts {
		if volumeMount.Name == config.SignalfxCaConfigMapVolumeName {
			volumeMountExists = true
			break
		}
	}
	if !volumeMountExists {
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(
			currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts,
			corev1.VolumeMount{
				Name:      config.SignalfxCaConfigMapVolumeName,
				MountPath: config.SignalfxCaMountPath,
				ReadOnly:  true,
			},
		)
	}

	// Check if volume already exists
	volumeExists := false
	for i := range currentDeployment.Spec.Template.Spec.Volumes {
		if currentDeployment.Spec.Template.Spec.Volumes[i].Name == config.SignalfxCaConfigMapVolumeName {
			volumeExists = true
			break
		}
	}
	if !volumeExists {
		currentDeployment.Spec.Template.Spec.Volumes = append(currentDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: config.SignalfxCaConfigMapVolumeName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: configMapName,
					},
					Items: []corev1.KeyToPath{
						{
							Key:  configMapKey,
							Path: config.SignalfxCaConfigMapMountedFile,
						},
					},
				},
			},
		})
	}

	return nil
}
