package k8sconfig

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
)

const (
	gcpApplicationCredentialsKey    = "GCP_APPLICATION_CREDENTIALS"
	gcpApplicationCredentialsEnvVar = "GOOGLE_APPLICATION_CREDENTIALS"
	gcpCredentialsMountPath         = "/secrets"
	gcpSecretVolumeName             = "gcp-credentials-secret"
)

type GoogleCloud struct{}

func (g *GoogleCloud) DestType() common.DestinationType {
	return common.GoogleCloudDestinationType
}

// ModifyGatewayCollectorDeployment modifies the gateway collector deployment to mount the GCP credentials secret and set the environment variable.
// When running on GCP, the credentials are automatically mounted by the GCP Collector Exporter.
// However, when running outside of GCP, credentials need to be supplied in the form of a JSON file to the Collector Pod.
// The location of this file then needs to be set as the GOOGLE_APPLICATION_CREDENTIALS environment variable.
// This function adds the volume mount and environment variable to the Collector Pod.
// This is also required for on-GCP deployments where the user wants to use a different project ID for billing and quota consumption
// (by providing a Service Account credentials file with access to the different project).
func (g *GoogleCloud) ModifyGatewayCollectorDeployment(ctx context.Context, k8sClient client.Client, dest K8sExporterConfigurer, currentDeployment *appsv1.Deployment) error {
	// Check if destination has a non-empty secret ref for Application Credentials
	if dest.GetSecretRef() == nil || dest.GetSecretRef().Name == "" {
		return nil
	}

	// Try to get the secret
	secret := &corev1.Secret{}
	err := k8sClient.Get(ctx, client.ObjectKey{
		Name:      dest.GetSecretRef().Name,
		Namespace: currentDeployment.Namespace,
	}, secret)
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", dest.GetSecretRef().Name, err)
	}

	// Check if the application credentials key is set in the secret
	if secret.Data == nil {
		return nil
	}

	// If GCP_APPLICATION_CREDENTIALS is set, mount the secret and set the environment variable
	// NOTE: Currently, only one GCP Destination may have Application Credentials configured. This is a limitation of the GCP Collector Exporter,
	// which relies on the GOOGLE_APPLICATION_CREDENTIALS environment variable to be set.
	// To support multiple GCP Destinations with different credentials (which is uncommon but not totally unreasonable), we would need to
	// create multiple Gateway Collector Deployments, one for each GCP Destination.
	if _, keyExists := secret.Data[gcpApplicationCredentialsKey]; keyExists {
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

		// Add volume mount if it doesn't exist
		for _, volumeMount := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts {
			if volumeMount.Name == gcpSecretVolumeName {
				return fmt.Errorf("GCP credentials volume mount %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", volumeMount.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts = append(currentDeployment.Spec.Template.Spec.Containers[containerIndex].VolumeMounts, corev1.VolumeMount{
			Name:      gcpSecretVolumeName,
			MountPath: gcpCredentialsMountPath,
		})

		// Add volume if it doesn't exist
		for i := range currentDeployment.Spec.Template.Spec.Volumes {
			if currentDeployment.Spec.Template.Spec.Volumes[i].Name == gcpSecretVolumeName {
				return fmt.Errorf("GCP credentials volume %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", currentDeployment.Spec.Template.Spec.Volumes[i].Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Volumes = append(currentDeployment.Spec.Template.Spec.Volumes, corev1.Volume{
			Name: gcpSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dest.GetSecretRef().Name,
					Items: []corev1.KeyToPath{
						{
							Key:  gcpApplicationCredentialsKey,
							Path: gcpApplicationCredentialsKey,
						},
					},
				},
			},
		})

		// Add environment variable pointing to the mounted credentials if it doesn't exist
		for _, env := range currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env {
			if env.Name == gcpApplicationCredentialsEnvVar {
				return fmt.Errorf("GCP credentials environment variable %s already exists."+
					"Only one GCP Destination may have Application Credentials configured", env.Name)
			}
		}
		currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env = append(currentDeployment.Spec.Template.Spec.Containers[containerIndex].Env, corev1.EnvVar{
			Name:  gcpApplicationCredentialsEnvVar,
			Value: gcpCredentialsMountPath + "/" + gcpApplicationCredentialsKey,
		})
	}

	return nil
}
